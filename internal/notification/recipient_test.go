package notification_test

import (
	"errors"
	"testing"

	"github.com/nexus-commerce/notification-service/internal/notification"
)

func TestRecipient(t *testing.T) {
	tests := map[string]struct {
		first, last  string
		wantName     string
		wantGreeting string
	}{
		"full name":    {"Jane", "Doe", "Jane Doe", "Hi Jane,"},
		"first only":   {"Jane", "", "Jane", "Hi Jane,"},
		"last only":    {"", "Doe", "Doe", "Hello,"},
		"anonymous":    {"", "", "", "Hello,"},
		"blank names":  {"  ", " ", "", "Hello,"},
		"padded names": {" Jane ", " Doe ", "Jane Doe", "Hi Jane,"},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			r, err := notification.NewRecipient(tt.first, tt.last, "jane@example.com")
			if err != nil {
				t.Fatalf("NewRecipient() error: %v", err)
			}

			if got := r.Name(); got != tt.wantName {
				t.Errorf("Name() = %q, want %q", got, tt.wantName)
			}
			if got := r.Greeting(); got != tt.wantGreeting {
				t.Errorf("Greeting() = %q, want %q", got, tt.wantGreeting)
			}
			if got := r.Address(); got != "jane@example.com" {
				t.Errorf("Address() = %q, want %q", got, "jane@example.com")
			}
		})
	}
}

func TestNewRecipientTrimsAddress(t *testing.T) {
	r, err := notification.NewRecipient("Jane", "Doe", " jane@example.com ")
	if err != nil {
		t.Fatalf("NewRecipient() error: %v", err)
	}
	if got := r.Address(); got != "jane@example.com" {
		t.Errorf("Address() = %q, want %q", got, "jane@example.com")
	}
}

func TestNewRecipientRequiresAddress(t *testing.T) {
	for name, address := range map[string]string{"empty": "", "blank": "  "} {
		t.Run(name, func(t *testing.T) {
			_, err := notification.NewRecipient("Jane", "Doe", address)
			if !errors.Is(err, notification.ErrMissingAddress) {
				t.Errorf("err = %v, want ErrMissingAddress", err)
			}
		})
	}
}
