package main

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/joho/godotenv"

	"github.com/nexus-commerce/notification-service/internal/config"
	"github.com/nexus-commerce/notification-service/internal/kafka"
	"github.com/nexus-commerce/notification-service/internal/lifecycle"
	"github.com/nexus-commerce/notification-service/internal/messaging"
	"github.com/nexus-commerce/notification-service/internal/nacl"
	"github.com/nexus-commerce/notification-service/internal/render"
	"github.com/nexus-commerce/notification-service/internal/sendgrid"
	"github.com/nexus-commerce/notification-service/internal/usecase"
)

type consumerConfig struct {
	Kafka        config.Kafka
	Consumer     config.Consumer
	SendGrid     config.SendGrid
	Notification config.Notification
	Frontend     config.Frontend
}

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	if err := loadDotEnv(); err != nil {
		return err
	}

	cfg, err := config.Load[consumerConfig]()
	if err != nil {
		return fmt.Errorf("load config (check env vars or .env): %w", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	opener, err := nacl.NewOpener(cfg.Notification.PrivateKey, cfg.Notification.KeyID)
	if err != nil {
		return fmt.Errorf("create token opener: %w", err)
	}
	renderer, err := render.New()
	if err != nil {
		return fmt.Errorf("create renderer: %w", err)
	}
	mailer := sendgrid.New(cfg.SendGrid.APIKey, cfg.SendGrid.FromName, cfg.SendGrid.FromAddress)
	links, err := usecase.NewLinks(cfg.Frontend.URL)
	if err != nil {
		return fmt.Errorf("create links: %w", err)
	}

	users, err := kafka.NewConsumer(kafka.ConsumerOptions{
		Addr:    cfg.Kafka.Addr(),
		GroupID: cfg.Consumer.GroupID,
		Topic:   messaging.TopicUsers,
	},
		kafka.NewRegistrationRequested(usecase.NewSendRegistrationConfirmation(renderer, mailer, opener, links, time.Now)),
		kafka.NewRegistered(usecase.NewSendWelcome(renderer, mailer, links)),
		kafka.NewDuplicateRegistrationAttempted(usecase.NewSendDuplicateRegistrationNotice(renderer, mailer, links)),
		kafka.NewEmailChangeRequested(usecase.NewSendEmailChangeConfirmation(renderer, mailer, opener, links, time.Now)),
		kafka.NewEmailChangeToTakenAddressAttempted(usecase.NewSendEmailTakenNotice(renderer, mailer)),
		kafka.NewEmailChanged(usecase.NewSendEmailChangedNotice(renderer, mailer)),
	)
	if err != nil {
		return fmt.Errorf("create users consumer: %w", err)
	}
	defer func() {
		if err := users.Close(); err != nil {
			log.Printf("close users consumer: %v", err)
		}
	}()

	orders, err := kafka.NewConsumer(kafka.ConsumerOptions{
		Addr:             cfg.Kafka.Addr(),
		GroupID:          cfg.Consumer.GroupID,
		Topic:            messaging.TopicOrdersConfirmed,
		DefaultEventType: messaging.TypeOrderConfirmed,
	}, kafka.NewOrderConfirmed(usecase.NewSendOrderConfirmation(renderer, mailer)))
	if err != nil {
		return fmt.Errorf("create orders consumer: %w", err)
	}
	defer func() {
		if err := orders.Close(); err != nil {
			log.Printf("close orders consumer: %v", err)
		}
	}()

	return lifecycle.Run(ctx, users, orders)
}

func loadDotEnv() error {
	if err := godotenv.Load(".env"); err != nil {
		if !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("parse .env: %w", err)
		}
		log.Println("no .env file found, relying on environment variables")
	}
	return nil
}
