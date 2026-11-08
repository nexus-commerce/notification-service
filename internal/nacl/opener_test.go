package nacl

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"testing"

	"golang.org/x/crypto/nacl/box"
)

func newKeyPair(t *testing.T) (*[keySize]byte, *[keySize]byte) {
	t.Helper()
	pub, priv, err := box.GenerateKey(rand.Reader)
	if err != nil {
		t.Fatalf("box.GenerateKey() error: %v", err)
	}
	return pub, priv
}

func TestOpenRoundTrip(t *testing.T) {
	pub, priv := newKeyPair(t)
	privateKey := base64.StdEncoding.EncodeToString(priv[:])

	sealed, err := box.SealAnonymous(nil, []byte("token-123"), pub, rand.Reader)
	if err != nil {
		t.Fatalf("box.SealAnonymous() error: %v", err)
	}

	opener, err := NewOpener(privateKey, "key-1")
	if err != nil {
		t.Fatalf("NewOpener() error: %v", err)
	}

	got, err := opener.Open(sealed)
	if err != nil {
		t.Fatalf("Open() error: %v", err)
	}
	if string(got) != "token-123" {
		t.Errorf("Open() = %q, want %q", got, "token-123")
	}
	if opener.KeyID() != "key-1" {
		t.Errorf("KeyID() = %q, want %q", opener.KeyID(), "key-1")
	}
}

func TestOpenRejectsTampered(t *testing.T) {
	pub, priv := newKeyPair(t)
	privateKey := base64.StdEncoding.EncodeToString(priv[:])

	sealed, err := box.SealAnonymous(nil, []byte("token-123"), pub, rand.Reader)
	if err != nil {
		t.Fatalf("box.SealAnonymous() error: %v", err)
	}
	sealed[len(sealed)-1] ^= 0xff

	opener, err := NewOpener(privateKey, "key-1")
	if err != nil {
		t.Fatalf("NewOpener() error: %v", err)
	}

	_, err = opener.Open(sealed)
	if !errors.Is(err, ErrCannotOpen) {
		t.Errorf("Open() error = %v, want %v", err, ErrCannotOpen)
	}
}

func TestNewOpenerRejectsBadKey(t *testing.T) {
	_, err := NewOpener("nope", "key-1")
	if !errors.Is(err, ErrInvalidPrivateKey) {
		t.Errorf("NewOpener() error = %v, want %v", err, ErrInvalidPrivateKey)
	}
}

func TestNewOpenerRejectsEmptyKeyID(t *testing.T) {
	_, priv := newKeyPair(t)
	privateKey := base64.StdEncoding.EncodeToString(priv[:])

	_, err := NewOpener(privateKey, "")
	if !errors.Is(err, ErrEmptyKeyID) {
		t.Errorf("NewOpener() error = %v, want %v", err, ErrEmptyKeyID)
	}
}
