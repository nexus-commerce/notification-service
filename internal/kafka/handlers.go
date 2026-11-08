package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/nexus-commerce/notification-service/internal/messaging"
	"github.com/nexus-commerce/notification-service/internal/notification"
)

type executor[T any] interface {
	Execute(ctx context.Context, e T) error
}

type EventHandler[T any] struct {
	typ    string
	decode func([]byte) (T, error)
	uc     executor[T]
}

func (h *EventHandler[T]) EventType() string { return h.typ }

func (h *EventHandler[T]) Handle(ctx context.Context, rec messaging.Record) error {
	e, err := h.decode(rec.Value)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrPermanent, err)
	}

	err = h.uc.Execute(ctx, e)
	if errors.Is(err, notification.ErrUndeliverable) {
		return fmt.Errorf("%w: %w", ErrPermanent, err)
	}
	return err
}

func newEventHandler[T any](typ string, decode func([]byte) (T, error), uc executor[T]) *EventHandler[T] {
	return &EventHandler[T]{typ: typ, decode: decode, uc: uc}
}

func NewRegistrationRequested(uc executor[notification.RegistrationRequested]) *EventHandler[notification.RegistrationRequested] {
	return newEventHandler(messaging.TypeRegistrationRequested, messaging.DecodeRegistrationRequested, uc)
}

func NewRegistered(uc executor[notification.Registered]) *EventHandler[notification.Registered] {
	return newEventHandler(messaging.TypeRegistered, messaging.DecodeRegistered, uc)
}

func NewDuplicateRegistrationAttempted(uc executor[notification.DuplicateRegistrationAttempted]) *EventHandler[notification.DuplicateRegistrationAttempted] {
	return newEventHandler(messaging.TypeDuplicateRegistrationAttempted, messaging.DecodeDuplicateRegistrationAttempted, uc)
}

func NewEmailChangeRequested(uc executor[notification.EmailChangeRequested]) *EventHandler[notification.EmailChangeRequested] {
	return newEventHandler(messaging.TypeEmailChangeRequested, messaging.DecodeEmailChangeRequested, uc)
}

func NewEmailChangeToTakenAddressAttempted(uc executor[notification.EmailChangeToTakenAddressAttempted]) *EventHandler[notification.EmailChangeToTakenAddressAttempted] {
	return newEventHandler(messaging.TypeEmailChangeToTakenAddressAttempted, messaging.DecodeEmailChangeToTakenAddressAttempted, uc)
}

func NewEmailChanged(uc executor[notification.EmailChanged]) *EventHandler[notification.EmailChanged] {
	return newEventHandler(messaging.TypeEmailChanged, messaging.DecodeEmailChanged, uc)
}

func NewOrderConfirmed(uc executor[notification.OrderConfirmed]) *EventHandler[notification.OrderConfirmed] {
	return newEventHandler(messaging.TypeOrderConfirmed, messaging.DecodeOrderConfirmed, uc)
}
