package kafka

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"time"

	"github.com/segmentio/kafka-go"

	"github.com/nexus-commerce/notification-service/internal/messaging"
)

const (
	minRetryDelay = 100 * time.Millisecond
	maxRetryDelay = 30 * time.Second
)

var ErrPermanent = errors.New("permanent message failure")

type Handler interface {
	EventType() string
	Handle(ctx context.Context, rec messaging.Record) error
}

type ConsumerOptions struct {
	Addr    string
	GroupID string
	Topic   string

	DefaultEventType string
}

// messageReader is the part of *kafka.Reader that Consumer uses. NewConsumer
// wires in the real reader; tests substitute a fake to drive retries and
// commits without a broker.
type messageReader interface {
	FetchMessage(ctx context.Context) (kafka.Message, error)
	CommitMessages(ctx context.Context, msgs ...kafka.Message) error
	Close() error
}

type Consumer struct {
	reader      messageReader
	router      router
	groupID     string
	topic       string
	defaultType string
}

func NewConsumer(opts ConsumerOptions, handlers ...Handler) (*Consumer, error) {
	r, err := newRouter(handlers...)
	if err != nil {
		return nil, err
	}
	return &Consumer{
		reader: kafka.NewReader(kafka.ReaderConfig{
			Brokers:     []string{opts.Addr},
			GroupID:     opts.GroupID,
			Topic:       opts.Topic,
			StartOffset: kafka.LastOffset,
		}),
		router:      r,
		groupID:     opts.GroupID,
		topic:       opts.Topic,
		defaultType: opts.DefaultEventType,
	}, nil
}

func (c *Consumer) Run(ctx context.Context) error {
	log.Printf("Starting kafka consumer %s on topic %s", c.groupID, c.topic)

	for {
		var km kafka.Message
		err := retry(ctx, "fetch message", temporary, func() error {
			var err error
			km, err = c.reader.FetchMessage(ctx)
			return err
		})
		if err != nil {
			if ctx.Err() != nil {
				log.Println("Stopping kafka consumer")
				return nil
			}
			return fmt.Errorf("fetch message: %w", err)
		}

		if err = c.router.route(ctx, toRecord(km, c.defaultType)); err != nil {
			if ctx.Err() != nil {
				log.Println("Stopping kafka consumer")
				return nil
			}
			return err
		}

		err = retry(ctx, "commit message", temporary, func() error {
			return c.reader.CommitMessages(ctx, km)
		})
		if err != nil {
			if ctx.Err() != nil {
				log.Println("Stopping kafka consumer")
				return nil
			}
			return fmt.Errorf("commit message %d/%d: %w", km.Partition, km.Offset, err)
		}
	}
}

func (c *Consumer) Close() error {
	return c.reader.Close()
}

func toRecord(km kafka.Message, defaultType string) messaging.Record {
	rec := messaging.Record{
		Type:  defaultType,
		Topic: km.Topic,
		Key:   string(km.Key),
		Value: km.Value,
	}
	for _, h := range km.Headers {
		if h.Key == messaging.HeaderEventType {
			rec.Type = string(h.Value)
		}
	}
	return rec
}

type router map[string]Handler

func newRouter(handlers ...Handler) (router, error) {
	r := make(router, len(handlers))
	for _, h := range handlers {
		if _, ok := r[h.EventType()]; ok {
			return nil, fmt.Errorf("duplicate handler for event type %q", h.EventType())
		}
		r[h.EventType()] = h
	}
	return r, nil
}

func (r router) route(ctx context.Context, rec messaging.Record) error {
	h, ok := r[rec.Type]
	if !ok {
		return nil
	}

	op := fmt.Sprintf("handle %q record with key %q", rec.Type, rec.Key)
	err := retry(ctx, op, func(err error) bool { return !errors.Is(err, ErrPermanent) }, func() error {
		return h.Handle(ctx, rec)
	})
	if errors.Is(err, ErrPermanent) {
		log.Printf("skip %q record with key %q: %v", rec.Type, rec.Key, err)
		return nil
	}
	return err
}

func temporary(err error) bool {
	if kerr, ok := errors.AsType[kafka.Error](err); ok {
		return kerr.Temporary()
	}
	_, ok := errors.AsType[net.Error](err)
	return ok
}

func retry(ctx context.Context, op string, retryable func(error) bool, fn func() error) error {
	delay := minRetryDelay
	for {
		err := fn()
		if err == nil || !retryable(err) {
			return err
		}
		log.Printf("%s, retrying in %v: %v", op, delay, err)

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(delay):
		}
		delay = min(delay*2, maxRetryDelay)
	}
}
