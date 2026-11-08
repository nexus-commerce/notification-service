package kafka

import (
	"context"
	"errors"
	"fmt"

	"github.com/nexus-commerce/notification-service/internal/event"
	"github.com/nexus-commerce/notification-service/internal/messaging"
	"github.com/nexus-commerce/notification-service/internal/usecase"
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

func (h *EventHandler[T]) Handle(ctx context.Context, msg messaging.Message) error {
	e, err := h.decode(msg.Value)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrPermanent, err)
	}

	err = h.uc.Execute(ctx, e)
	if errors.Is(err, usecase.ErrUndeliverable) {
		return fmt.Errorf("%w: %w", ErrPermanent, err)
	}
	return err
}

func NewRegistrationRequested(uc executor[event.RegistrationRequested]) *EventHandler[event.RegistrationRequested] {
	return &EventHandler[event.RegistrationRequested]{typ: messaging.TypeRegistrationRequested, decode: messaging.DecodeRegistrationRequested, uc: uc}
}

func NewRegistered(uc executor[event.Registered]) *EventHandler[event.Registered] {
	return &EventHandler[event.Registered]{typ: messaging.TypeRegistered, decode: messaging.DecodeRegistered, uc: uc}
}

func NewDuplicateRegistrationAttempted(uc executor[event.DuplicateRegistrationAttempted]) *EventHandler[event.DuplicateRegistrationAttempted] {
	return &EventHandler[event.DuplicateRegistrationAttempted]{typ: messaging.TypeDuplicateRegistrationAttempted, decode: messaging.DecodeDuplicateRegistrationAttempted, uc: uc}
}

func NewEmailChangeRequested(uc executor[event.EmailChangeRequested]) *EventHandler[event.EmailChangeRequested] {
	return &EventHandler[event.EmailChangeRequested]{typ: messaging.TypeEmailChangeRequested, decode: messaging.DecodeEmailChangeRequested, uc: uc}
}

func NewEmailChangeToTakenAddressAttempted(uc executor[event.EmailChangeToTakenAddressAttempted]) *EventHandler[event.EmailChangeToTakenAddressAttempted] {
	return &EventHandler[event.EmailChangeToTakenAddressAttempted]{typ: messaging.TypeEmailChangeToTakenAddressAttempted, decode: messaging.DecodeEmailChangeToTakenAddressAttempted, uc: uc}
}

func NewEmailChanged(uc executor[event.EmailChanged]) *EventHandler[event.EmailChanged] {
	return &EventHandler[event.EmailChanged]{typ: messaging.TypeEmailChanged, decode: messaging.DecodeEmailChanged, uc: uc}
}

func NewOrderConfirmed(uc executor[event.OrderConfirmed]) *EventHandler[event.OrderConfirmed] {
	return &EventHandler[event.OrderConfirmed]{typ: messaging.TypeOrderConfirmed, decode: messaging.DecodeOrderConfirmed, uc: uc}
}
