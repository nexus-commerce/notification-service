package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nexus-commerce/notification-service/internal/notification"
)

type fakeNotifier struct {
	sent []notification.Message
	err  error
}

func (f *fakeNotifier) Notify(_ context.Context, m notification.Message) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, m)
	return nil
}

func only[M notification.Message](t *testing.T, n *fakeNotifier) M {
	t.Helper()
	if len(n.sent) != 1 {
		t.Fatalf("len(sent) = %d, want 1", len(n.sent))
	}
	m, ok := n.sent[0].(M)
	if !ok {
		var want M
		t.Fatalf("sent %T, want %T", n.sent[0], want)
	}
	return m
}

type fakeOpener struct {
	keyID string
	plain map[string]string
}

func (f fakeOpener) Open(sealed []byte) ([]byte, error) {
	p, ok := f.plain[string(sealed)]
	if !ok {
		return nil, errors.New("cannot open sealed token")
	}
	return []byte(p), nil
}

func (f fakeOpener) KeyID() string {
	return f.keyID
}

func fixedNow() time.Time {
	return time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
}

const (
	testKeyID  = "key-1"
	testSealed = "sealed-1"
	testToken  = "tok en/1"
)

var (
	jane           = mustRecipient("Jane", "Doe", "jane@example.com")
	janeOldAddress = mustRecipient("Jane", "Doe", "old@example.com")
	janeNewAddress = mustRecipient("Jane", "Doe", "new@example.com")
	anonymous      = mustRecipient("", "", "jane@example.com")
)

func newOpener() fakeOpener {
	return fakeOpener{keyID: testKeyID, plain: map[string]string{testSealed: testToken}}
}

func newToken(t *testing.T, sealed, keyID string, expiresAt time.Time) notification.ConfirmationToken {
	t.Helper()
	tok, err := notification.NewConfirmationToken([]byte(sealed), keyID, expiresAt)
	if err != nil {
		t.Fatalf("NewConfirmationToken() error: %v", err)
	}
	return tok
}

type unrevealable struct {
	token   notification.ConfirmationToken
	wantErr error
}

func unrevealableTokens(t *testing.T) map[string]unrevealable {
	return map[string]unrevealable{
		"expired":          {newToken(t, testSealed, testKeyID, fixedNow()), notification.ErrTokenExpired},
		"other key":        {newToken(t, testSealed, "other-key", fixedNow().Add(time.Hour)), notification.ErrKeyMismatch},
		"cannot be opened": {newToken(t, "garbage", testKeyID, fixedNow().Add(time.Hour)), notification.ErrInvalidToken},
	}
}

func assertUndeliverable(t *testing.T, err, wantErr error, n *fakeNotifier) {
	t.Helper()
	if !errors.Is(err, wantErr) {
		t.Errorf("Execute() error = %v, want %v", err, wantErr)
	}
	if !errors.Is(err, notification.ErrUndeliverable) {
		t.Errorf("Execute() error = %v, want %v", err, notification.ErrUndeliverable)
	}
	if err != nil && strings.Contains(err.Error(), "tok") {
		t.Errorf("Execute() error = %q, must not contain token material", err.Error())
	}
	if len(n.sent) != 0 {
		t.Errorf("len(sent) = %d, want 0", len(n.sent))
	}
}

// mustRecipient builds a package-level test recipient and panics on an
// invalid one. Inside a test, construct recipients with a helper that calls
// t.Fatal instead.
func mustRecipient(firstName, lastName, address string) notification.Recipient {
	r, err := notification.NewRecipient(firstName, lastName, address)
	if err != nil {
		panic(err)
	}
	return r
}
