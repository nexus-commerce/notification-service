package notification_test

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nexus-commerce/notification-service/internal/notification"
)

type fakeOpener struct {
	keyID string
	plain map[string]string
}

func (f fakeOpener) Open(sealed []byte) ([]byte, error) {
	p, ok := f.plain[string(sealed)]
	if !ok {
		return nil, errors.New("cannot open sealed token tok en/1")
	}
	return []byte(p), nil
}

func (f fakeOpener) KeyID() string { return f.keyID }

var (
	now     = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	expires = now.Add(time.Hour)
	opener  = fakeOpener{keyID: "key-1", plain: map[string]string{"sealed-1": "tok en/1"}}
)

func mustToken(t *testing.T, sealed, keyID string, expiresAt time.Time) notification.ConfirmationToken {
	t.Helper()
	tok, err := notification.NewConfirmationToken([]byte(sealed), keyID, expiresAt)
	if err != nil {
		t.Fatalf("NewConfirmationToken() error: %v", err)
	}
	return tok
}

func TestConfirmationTokenReveal(t *testing.T) {
	tok := mustToken(t, "sealed-1", "key-1", expires)

	got, err := tok.Reveal(opener, now)

	if err != nil {
		t.Fatalf("Reveal() error: %v", err)
	}
	if got.Value() != "tok en/1" {
		t.Errorf("Reveal() = %q, want %q", got.Value(), "tok en/1")
	}
	if !tok.ExpiresAt().Equal(expires) {
		t.Errorf("ExpiresAt() = %v, want %v", tok.ExpiresAt(), expires)
	}
}

func TestConfirmationTokenRevealRejects(t *testing.T) {
	tests := map[string]struct {
		tok     func(t *testing.T) notification.ConfirmationToken
		wantErr error
	}{
		"expired exactly at expiry": {
			tok:     func(t *testing.T) notification.ConfirmationToken { return mustToken(t, "sealed-1", "key-1", now) },
			wantErr: notification.ErrTokenExpired,
		},
		"sealed with another key": {
			tok: func(t *testing.T) notification.ConfirmationToken {
				return mustToken(t, "sealed-1", "other-key", expires)
			},
			wantErr: notification.ErrKeyMismatch,
		},
		"cannot be opened": {
			tok:     func(t *testing.T) notification.ConfirmationToken { return mustToken(t, "garbage", "key-1", expires) },
			wantErr: notification.ErrInvalidToken,
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			got, err := tt.tok(t).Reveal(opener, now)

			if !errors.Is(err, tt.wantErr) {
				t.Errorf("Reveal() error = %v, want %v", err, tt.wantErr)
			}
			if got.Value() != "" {
				t.Errorf("Reveal() = %q, want empty", got.Value())
			}
			if err != nil && strings.Contains(err.Error(), "tok") {
				t.Errorf("Reveal() error = %q, must not contain token material", err.Error())
			}
		})
	}
}

func TestNewConfirmationTokenRejectsIncomplete(t *testing.T) {
	tests := map[string]struct {
		sealed    []byte
		expiresAt time.Time
	}{
		"no sealed value": {sealed: nil, expiresAt: expires},
		"no expiry":       {sealed: []byte("sealed-1"), expiresAt: time.Time{}},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := notification.NewConfirmationToken(tt.sealed, "key-1", tt.expiresAt)
			if !errors.Is(err, notification.ErrIncompleteToken) {
				t.Errorf("NewConfirmationToken() error = %v, want %v", err, notification.ErrIncompleteToken)
			}
		})
	}
}

func TestConfirmationTokenIsImmutable(t *testing.T) {
	sealed := []byte("sealed-1")
	tok, err := notification.NewConfirmationToken(sealed, "key-1", expires)
	if err != nil {
		t.Fatalf("NewConfirmationToken() error: %v", err)
	}
	sealed[0] = 'X'

	if _, err := tok.Reveal(opener, now); err != nil {
		t.Errorf("Reveal() after caller mutated input: %v", err)
	}
}
