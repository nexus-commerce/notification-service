package kafka

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"github.com/nexus-commerce/notification-service/internal/messaging"
	"github.com/nexus-commerce/notification-service/internal/notification"
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

func newRegisteredHandler(fakeUC *fakeUseCase[notification.Registered]) Handler {
	return NewRegistered(fakeUC)
}

func TestHandlerPassesDecodedEvent(t *testing.T) {
	fakeUC := &fakeUseCase[notification.Registered]{}
	h := newRegisteredHandler(fakeUC)

	err := h.Handle(t.Context(), messaging.Record{Type: messaging.TypeRegistered, Value: []byte(registeredPayload)})
	if err != nil {
		t.Fatalf("Handle() error: %v", err)
	}
	if got := fakeUC.got.Recipient.Address(); got != "jane@example.com" {
		t.Errorf("Recipient.Address() = %q, want %q", got, "jane@example.com")
	}
}

func TestHandlerMalformedIsPermanent(t *testing.T) {
	fakeUC := &fakeUseCase[notification.Registered]{}
	h := newRegisteredHandler(fakeUC)

	err := h.Handle(t.Context(), messaging.Record{Value: []byte("not json")})
	if !errors.Is(err, ErrPermanent) {
		t.Errorf("Handle() error = %v, want ErrPermanent", err)
	}
	if !errors.Is(err, messaging.ErrMalformedMessage) {
		t.Errorf("Handle() error = %v, want ErrMalformedMessage", err)
	}
}

func TestHandlerUndeliverableIsPermanent(t *testing.T) {
	fakeUC := &fakeUseCase[notification.Registered]{err: fmt.Errorf("send: %w", notification.ErrUndeliverable)}
	h := newRegisteredHandler(fakeUC)

	err := h.Handle(t.Context(), messaging.Record{Value: []byte(registeredPayload)})
	if !errors.Is(err, ErrPermanent) {
		t.Errorf("Handle() error = %v, want ErrPermanent", err)
	}
	if !errors.Is(err, notification.ErrUndeliverable) {
		t.Errorf("Handle() error = %v, want ErrUndeliverable", err)
	}
}

func TestHandlerTransientErrorPassesThrough(t *testing.T) {
	boom := errors.New("boom")
	fakeUC := &fakeUseCase[notification.Registered]{err: boom}
	h := newRegisteredHandler(fakeUC)

	err := h.Handle(t.Context(), messaging.Record{Value: []byte(registeredPayload)})
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
		{"registration requested", NewRegistrationRequested(&fakeUseCase[notification.RegistrationRequested]{}), messaging.TypeRegistrationRequested},
		{"registered", NewRegistered(&fakeUseCase[notification.Registered]{}), messaging.TypeRegistered},
		{"duplicate registration", NewDuplicateRegistrationAttempted(&fakeUseCase[notification.DuplicateRegistrationAttempted]{}), messaging.TypeDuplicateRegistrationAttempted},
		{"email change requested", NewEmailChangeRequested(&fakeUseCase[notification.EmailChangeRequested]{}), messaging.TypeEmailChangeRequested},
		{"email change to taken", NewEmailChangeToTakenAddressAttempted(&fakeUseCase[notification.EmailChangeToTakenAddressAttempted]{}), messaging.TypeEmailChangeToTakenAddressAttempted},
		{"email changed", NewEmailChanged(&fakeUseCase[notification.EmailChanged]{}), messaging.TypeEmailChanged},
		{"order confirmed", NewOrderConfirmed(&fakeUseCase[notification.OrderConfirmed]{}), messaging.TypeOrderConfirmed},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.h.EventType(); got != tt.want {
				t.Errorf("EventType() = %q, want %q", got, tt.want)
			}
		})
	}
}
