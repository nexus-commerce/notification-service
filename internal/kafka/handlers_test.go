package kafka

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/nexus-commerce/notification-service/internal/event"
	"github.com/nexus-commerce/notification-service/internal/messaging"
	"github.com/nexus-commerce/notification-service/internal/usecase"
)

type fakeUseCase[T any] struct {
	got T
	err error
}

func (f *fakeUseCase[T]) Execute(_ context.Context, e T) error {
	f.got = e
	return f.err
}

const registeredPayload = `{"eventType":"user.registered","data":{"userId":"u1","email":"jane@example.com","firstName":"Jane","lastName":"Doe"}}`

func newRegisteredHandler(uc *fakeUseCase[event.Registered]) Handler {
	return NewRegistered(uc)
}

func TestHandlerPassesDecodedEvent(t *testing.T) {
	uc := &fakeUseCase[event.Registered]{}
	h := newRegisteredHandler(uc)

	err := h.Handle(context.Background(), messaging.Message{Type: messaging.TypeRegistered, Value: []byte(registeredPayload)})
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}
	if uc.got.Email != "jane@example.com" {
		t.Errorf("Email = %q, want %q", uc.got.Email, "jane@example.com")
	}
}

func TestHandlerMalformedIsPermanent(t *testing.T) {
	uc := &fakeUseCase[event.Registered]{}
	h := newRegisteredHandler(uc)

	err := h.Handle(context.Background(), messaging.Message{Value: []byte("not json")})
	if !errors.Is(err, ErrPermanent) {
		t.Errorf("Handle() error = %v, want ErrPermanent", err)
	}
	if !errors.Is(err, messaging.ErrMalformedMessage) {
		t.Errorf("Handle() error = %v, want ErrMalformedMessage", err)
	}
}

func TestHandlerUndeliverableIsPermanent(t *testing.T) {
	uc := &fakeUseCase[event.Registered]{err: fmt.Errorf("send: %w", usecase.ErrUndeliverable)}
	h := newRegisteredHandler(uc)

	err := h.Handle(context.Background(), messaging.Message{Value: []byte(registeredPayload)})
	if !errors.Is(err, ErrPermanent) {
		t.Errorf("Handle() error = %v, want ErrPermanent", err)
	}
	if !errors.Is(err, usecase.ErrUndeliverable) {
		t.Errorf("Handle() error = %v, want ErrUndeliverable", err)
	}
}

func TestHandlerTransientErrorPassesThrough(t *testing.T) {
	boom := errors.New("boom")
	uc := &fakeUseCase[event.Registered]{err: boom}
	h := newRegisteredHandler(uc)

	err := h.Handle(context.Background(), messaging.Message{Value: []byte(registeredPayload)})
	if err != boom {
		t.Errorf("Handle() error = %v, want %v", err, boom)
	}
	if errors.Is(err, ErrPermanent) {
		t.Error("Handle() error is permanent, want transient")
	}
}

func TestHandlerEventTypes(t *testing.T) {
	tests := []struct {
		name string
		h    Handler
		want string
	}{
		{"registration requested", NewRegistrationRequested(&fakeUseCase[event.RegistrationRequested]{}), messaging.TypeRegistrationRequested},
		{"registered", NewRegistered(&fakeUseCase[event.Registered]{}), messaging.TypeRegistered},
		{"duplicate registration", NewDuplicateRegistrationAttempted(&fakeUseCase[event.DuplicateRegistrationAttempted]{}), messaging.TypeDuplicateRegistrationAttempted},
		{"email change requested", NewEmailChangeRequested(&fakeUseCase[event.EmailChangeRequested]{}), messaging.TypeEmailChangeRequested},
		{"email change to taken", NewEmailChangeToTakenAddressAttempted(&fakeUseCase[event.EmailChangeToTakenAddressAttempted]{}), messaging.TypeEmailChangeToTakenAddressAttempted},
		{"email changed", NewEmailChanged(&fakeUseCase[event.EmailChanged]{}), messaging.TypeEmailChanged},
		{"order confirmed", NewOrderConfirmed(&fakeUseCase[event.OrderConfirmed]{}), messaging.TypeOrderConfirmed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.h.EventType(); got != tt.want {
				t.Errorf("EventType() = %q, want %q", got, tt.want)
			}
		})
	}
}
