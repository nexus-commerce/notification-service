package notification_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/nexus-commerce/notification-service/internal/notification"
)

func TestRevealedTokenIsRedactedWhenPrinted(t *testing.T) {
	tok, err := mustToken(t, "sealed-1", "key-1", expires).Reveal(opener, now)
	if err != nil {
		t.Fatalf("Reveal() error: %v", err)
	}
	msg := notification.RegistrationConfirmation{Token: tok, ExpiresAt: expires}

	for _, verb := range []string{"%v", "%+v", "%#v", "%s"} {
		if got := fmt.Sprintf(verb, msg); strings.Contains(got, "tok en/1") {
			t.Errorf("Sprintf(%q) = %q, must not contain the token", verb, got)
		}
	}
	if got := tok.Value(); got != "tok en/1" {
		t.Errorf("Value() = %q, want %q", got, "tok en/1")
	}
}
