package nacl

import (
	"encoding/base64"
	"errors"
	"fmt"

	"golang.org/x/crypto/curve25519"
	"golang.org/x/crypto/nacl/box"
)

const keySize = 32

var (
	ErrInvalidPrivateKey = errors.New("invalid private key")
	ErrEmptyKeyID        = errors.New("empty key id")
	ErrCannotOpen        = errors.New("cannot open sealed box")
)

type Opener struct {
	publicKey  *[keySize]byte
	privateKey *[keySize]byte
	keyID      string
}

func NewOpener(privateKey, keyID string) (*Opener, error) {
	raw, err := base64.StdEncoding.DecodeString(privateKey)
	if err != nil || len(raw) != keySize {
		return nil, fmt.Errorf("%w: want base64 of %d bytes", ErrInvalidPrivateKey, keySize)
	}
	if keyID == "" {
		return nil, ErrEmptyKeyID
	}
	rawPub, err := curve25519.X25519(raw, curve25519.Basepoint)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrInvalidPrivateKey, err)
	}
	var priv, pub [keySize]byte
	copy(priv[:], raw)
	copy(pub[:], rawPub)
	return &Opener{publicKey: &pub, privateKey: &priv, keyID: keyID}, nil
}

func (o *Opener) Open(sealed []byte) ([]byte, error) {
	opened, ok := box.OpenAnonymous(nil, sealed, o.publicKey, o.privateKey)
	if !ok {
		return nil, ErrCannotOpen
	}
	return opened, nil
}

func (o *Opener) KeyID() string { return o.keyID }
