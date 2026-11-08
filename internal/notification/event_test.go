package notification_test

import (
	"errors"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/nexus-commerce/notification-service/internal/notification"
)

func mustRecipient(t *testing.T) notification.Recipient {
	t.Helper()
	r, err := notification.NewRecipient("Jane", "Doe", "jane@example.com")
	if err != nil {
		t.Fatalf("NewRecipient() error: %v", err)
	}
	return r
}

// confirmationEvents covers every event that warrants a confirmation carrying
// a revealed token.
func confirmationEvents(t *testing.T, tok notification.ConfirmationToken) map[string]func() (notification.Recipient, notification.RevealedToken, time.Time, error) {
	jane := mustRecipient(t)
	return map[string]func() (notification.Recipient, notification.RevealedToken, time.Time, error){
		"registration": func() (notification.Recipient, notification.RevealedToken, time.Time, error) {
			m, err := notification.RegistrationRequested{Recipient: jane, Token: tok}.Confirmation(opener, now)
			return m.To, m.Token, m.ExpiresAt, err
		},
		"email change": func() (notification.Recipient, notification.RevealedToken, time.Time, error) {
			m, err := notification.EmailChangeRequested{Recipient: jane, Token: tok}.Confirmation(opener, now)
			return m.To, m.Token, m.ExpiresAt, err
		},
	}
}

func TestConfirmation(t *testing.T) {
	jane := mustRecipient(t)
	for name, confirm := range confirmationEvents(t, mustToken(t, "sealed-1", "key-1", expires)) {
		t.Run(name, func(t *testing.T) {
			to, tok, expiresAt, err := confirm()
			if err != nil {
				t.Fatalf("Confirmation() error: %v", err)
			}
			if to != jane {
				t.Errorf("To = %+v, want %+v", to, jane)
			}
			if tok.Value() != "tok en/1" {
				t.Errorf("Token = %q, want %q", tok.Value(), "tok en/1")
			}
			if !expiresAt.Equal(expires) {
				t.Errorf("ExpiresAt = %v, want %v", expiresAt, expires)
			}
		})
	}
}

func TestConfirmationIsUndeliverableWhenTokenCannotBeRevealed(t *testing.T) {
	tokens := map[string]struct {
		token   notification.ConfirmationToken
		wantErr error
	}{
		"expired":          {mustToken(t, "sealed-1", "key-1", now), notification.ErrTokenExpired},
		"other key":        {mustToken(t, "sealed-1", "other-key", expires), notification.ErrKeyMismatch},
		"cannot be opened": {mustToken(t, "garbage", "key-1", expires), notification.ErrInvalidToken},
	}
	for tokName, tt := range tokens {
		for evName, confirm := range confirmationEvents(t, tt.token) {
			t.Run(evName+"/"+tokName, func(t *testing.T) {
				_, _, _, err := confirm()
				if !errors.Is(err, notification.ErrUndeliverable) {
					t.Errorf("Confirmation() error = %v, want %v", err, notification.ErrUndeliverable)
				}
				if !errors.Is(err, tt.wantErr) {
					t.Errorf("Confirmation() error = %v, want %v", err, tt.wantErr)
				}
			})
		}
	}
}

func TestNotices(t *testing.T) {
	jane := mustRecipient(t)
	tests := map[string]struct {
		got  notification.Message
		want notification.Message
	}{
		"registered": {
			got:  notification.Registered{Recipient: jane}.Welcome(),
			want: notification.Welcome{To: jane},
		},
		"duplicate registration": {
			got:  notification.DuplicateRegistrationAttempted{Recipient: jane}.Notice(),
			want: notification.DuplicateRegistrationNotice{To: jane},
		},
		"email change to taken address": {
			got:  notification.EmailChangeToTakenAddressAttempted{Recipient: jane}.Notice(),
			want: notification.AddressTakenNotice{To: jane},
		},
		"email changed": {
			got:  notification.EmailChanged{Recipient: jane, NewAddress: "new@example.com"}.Notice(),
			want: notification.EmailChangedNotice{To: jane, NewAddress: "new@example.com"},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Errorf("got %+v, want %+v", tt.got, tt.want)
			}
		})
	}
}

func TestOrderConfirmation(t *testing.T) {
	jane := mustRecipient(t)
	ev := notification.OrderConfirmed{
		Recipient:         jane,
		OrderID:           42,
		OrderDate:         now,
		PaidByCard:        true,
		Items:             []notification.OrderItem{{Quantity: 2, Price: 5, SKU: "SKU-1", Name: "Mug", TotalPrice: 10}},
		Total:             10,
		ShippingAddress:   "1 Main St",
		EstimatedDelivery: expires,
	}

	got := ev.Confirmation()

	want := notification.OrderConfirmation{
		To:                jane,
		OrderID:           42,
		OrderDate:         now,
		PaidByCard:        true,
		Items:             []notification.OrderItem{{Quantity: 2, Price: 5, SKU: "SKU-1", Name: "Mug", TotalPrice: 10}},
		Total:             10,
		ShippingAddress:   "1 Main St",
		EstimatedDelivery: expires,
	}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(notification.Recipient{})); diff != "" {
		t.Fatalf("Confirmation() mismatch (-want +got):\n%s", diff)
	}
	ev.Items[0].Name = "changed"
	if got.Items[0].Name != "Mug" {
		t.Errorf("Items share memory with the event")
	}
}
