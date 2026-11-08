package usecase_test

import (
	"errors"
	"testing"
	"time"

	"github.com/nexus-commerce/notification-service/internal/notification"
	"github.com/nexus-commerce/notification-service/internal/usecase"
)

func registrationRequested(t *testing.T) notification.RegistrationRequested {
	return notification.RegistrationRequested{
		Recipient: jane,
		Token:     newToken(t, testSealed, testKeyID, fixedNow().Add(time.Hour)),
	}
}

func TestSendRegistrationConfirmation(t *testing.T) {
	n := &fakeNotifier{}
	uc := usecase.NewSendRegistrationConfirmation(n, newOpener(), fixedNow)
	ev := registrationRequested(t)

	if err := uc.Execute(t.Context(), ev); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	got := only[notification.RegistrationConfirmation](t, n)
	if got.To != jane {
		t.Errorf("To = %+v, want %+v", got.To, jane)
	}
	if got.Token.Value() != testToken {
		t.Errorf("Token = %q, want %q", got.Token.Value(), testToken)
	}
	if !got.ExpiresAt.Equal(ev.Token.ExpiresAt()) {
		t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, ev.Token.ExpiresAt())
	}
}

func TestSendRegistrationConfirmationRejectsUnrevealableToken(t *testing.T) {
	for name, tt := range unrevealableTokens(t) {
		t.Run(name, func(t *testing.T) {
			n := &fakeNotifier{}
			uc := usecase.NewSendRegistrationConfirmation(n, newOpener(), fixedNow)
			ev := registrationRequested(t)
			ev.Token = tt.token

			assertUndeliverable(t, uc.Execute(t.Context(), ev), tt.wantErr, n)
		})
	}
}

func TestSendRegistrationConfirmationReturnsNotifierError(t *testing.T) {
	notifyErr := errors.New("smtp down")
	uc := usecase.NewSendRegistrationConfirmation(&fakeNotifier{err: notifyErr}, newOpener(), fixedNow)

	err := uc.Execute(t.Context(), registrationRequested(t))

	if !errors.Is(err, notifyErr) {
		t.Errorf("Execute() error = %v, want %v", err, notifyErr)
	}
	if errors.Is(err, notification.ErrUndeliverable) {
		t.Errorf("Execute() error = %v, want not %v", err, notification.ErrUndeliverable)
	}
}
