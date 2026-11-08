package notification

import (
	"bytes"
	"errors"
	"fmt"
	"time"
)

type Opener interface {
	Open(sealed []byte) ([]byte, error)
	KeyID() string
}

var (
	ErrTokenExpired    = errors.New("confirmation link expired")
	ErrKeyMismatch     = errors.New("sealing key mismatch")
	ErrInvalidToken    = errors.New("sealed value cannot be opened")
	ErrIncompleteToken = errors.New("incomplete confirmation link")
)

type ConfirmationToken struct {
	sealed    []byte
	keyID     string
	expiresAt time.Time
}

func NewConfirmationToken(sealed []byte, keyID string, expiresAt time.Time) (ConfirmationToken, error) {
	if len(sealed) == 0 {
		return ConfirmationToken{}, fmt.Errorf("%w: missing sealed value", ErrIncompleteToken)
	}
	if expiresAt.IsZero() {
		return ConfirmationToken{}, fmt.Errorf("%w: missing expiry", ErrIncompleteToken)
	}
	return ConfirmationToken{
		sealed:    bytes.Clone(sealed),
		keyID:     keyID,
		expiresAt: expiresAt,
	}, nil
}

func (t ConfirmationToken) ExpiresAt() time.Time { return t.expiresAt }

func (t ConfirmationToken) Reveal(o Opener, now time.Time) (RevealedToken, error) {
	if t.keyID != o.KeyID() {
		return RevealedToken{}, fmt.Errorf("%w: key id %q", ErrKeyMismatch, t.keyID)
	}
	if !now.Before(t.expiresAt) {
		return RevealedToken{}, ErrTokenExpired
	}
	plain, err := o.Open(t.sealed)
	if err != nil {
		return RevealedToken{}, ErrInvalidToken
	}
	return RevealedToken{value: string(plain)}, nil
}
