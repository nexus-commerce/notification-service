package email_test

import (
	"testing"

	"github.com/nexus-commerce/notification-service/internal/email"
)

func TestNewLinksTrimsTrailingSlash(t *testing.T) {
	l, err := email.NewLinks("http://front.test/")
	if err != nil {
		t.Fatalf("NewLinks() error: %v", err)
	}

	if got, want := l.LoginURL(), "http://front.test/login"; got != want {
		t.Errorf("LoginURL() = %q, want %q", got, want)
	}
}

func TestNewLinksRejectsInvalidURL(t *testing.T) {
	if _, err := email.NewLinks("://"); err == nil {
		t.Errorf("NewLinks() error = nil, want error")
	}
}
