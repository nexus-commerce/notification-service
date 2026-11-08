package usecase_test

import (
	"testing"
	"time"

	"github.com/nexus-commerce/notification-service/internal/notification"
	"github.com/nexus-commerce/notification-service/internal/usecase"
)

func emailChangeRequested(t *testing.T) notification.EmailChangeRequested {
	return notification.EmailChangeRequested{
		Recipient: janeNewAddress,
		Token:     newToken(t, testSealed, testKeyID, fixedNow().Add(time.Hour)),
	}
}

func TestSendEmailChangeConfirmation(t *testing.T) {
	n := &fakeNotifier{}
	uc := usecase.NewSendEmailChangeConfirmation(n, newOpener(), fixedNow)
	ev := emailChangeRequested(t)

	if err := uc.Execute(t.Context(), ev); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	got := only[notification.EmailChangeConfirmation](t, n)
	if got.To != ev.Recipient {
		t.Errorf("To = %+v, want %+v", got.To, ev.Recipient)
	}
	if got.Token.Value() != testToken {
		t.Errorf("Token = %q, want %q", got.Token.Value(), testToken)
	}
	if !got.ExpiresAt.Equal(ev.Token.ExpiresAt()) {
		t.Errorf("ExpiresAt = %v, want %v", got.ExpiresAt, ev.Token.ExpiresAt())
	}
}

func TestSendEmailChangeConfirmationRejectsUnrevealableToken(t *testing.T) {
	for name, tt := range unrevealableTokens(t) {
		t.Run(name, func(t *testing.T) {
			n := &fakeNotifier{}
			uc := usecase.NewSendEmailChangeConfirmation(n, newOpener(), fixedNow)
			ev := emailChangeRequested(t)
			ev.Token = tt.token

			assertUndeliverable(t, uc.Execute(t.Context(), ev), tt.wantErr, n)
		})
	}
}
