package usecase

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"
)

// Page names understood by a Renderer.
const (
	PageConfirmRegistration   = "confirm-registration"
	PageWelcome               = "welcome"
	PageDuplicateRegistration = "duplicate-registration"
	PageConfirmEmail          = "confirm-email"
	PageEmailTaken            = "email-taken"
	PageEmailChanged          = "email-changed"
	PageOrderConfirmed        = "order-confirmed"
)

// ErrUndeliverable marks a permanent failure: retrying the same message will
// not succeed. Adapters signal permanent failure by wrapping it.
var ErrUndeliverable = errors.New("undeliverable")

// Token errors. Their messages avoid the word "token" so that a test can
// assert no token material reaches error strings.
var (
	ErrTokenExpired = fmt.Errorf("confirmation link expired: %w", ErrUndeliverable)
	ErrKeyMismatch  = fmt.Errorf("sealing key mismatch: %w", ErrUndeliverable)
	ErrInvalidToken = fmt.Errorf("sealed value cannot be opened: %w", ErrUndeliverable)
)

type Email struct {
	ToName    string
	ToAddress string
	Subject   string
	PlainText string
	HTML      string
}

type Mailer interface {
	Send(ctx context.Context, e Email) error
}

type Renderer interface {
	Render(page string, data any) (string, error)
}

type Opener interface {
	Open(sealed []byte) ([]byte, error)
	KeyID() string
}

// Links builds absolute URLs pointing at the frontend.
type Links struct {
	base *url.URL
}

func NewLinks(frontendURL string) (Links, error) {
	base, err := url.Parse(frontendURL)
	if err != nil {
		return Links{}, fmt.Errorf("parse frontend url: %w", err)
	}
	if base.Scheme == "" || base.Host == "" {
		return Links{}, fmt.Errorf("frontend url %q must be absolute", frontendURL)
	}
	base.Path = strings.TrimRight(base.Path, "/")
	base.RawPath = ""
	base.RawQuery = ""
	base.Fragment = ""
	return Links{base: base}, nil
}

func (l Links) ConfirmRegistration(token string) string {
	return l.withToken("/confirm-registration", token)
}

func (l Links) ConfirmEmail(token string) string {
	return l.withToken("/confirm-email", token)
}

func (l Links) Login() string {
	return l.at("/login").String()
}

func (l Links) withToken(path, token string) string {
	u := l.at(path)
	u.RawQuery = url.Values{"token": {token}}.Encode()
	return u.String()
}

func (l Links) at(path string) *url.URL {
	u := *l.base
	u.Path += path
	return &u
}

func fullName(firstName, lastName string) string {
	return strings.TrimSpace(firstName + " " + lastName)
}

// openToken checks the key ID and expiry, then opens the sealed token. The
// opener's error is deliberately dropped so plaintext never leaks.
func openToken(o Opener, now func() time.Time, keyID string, sealed []byte, expiresAt time.Time) (string, error) {
	if keyID != o.KeyID() {
		return "", fmt.Errorf("key id %q: %w", keyID, ErrKeyMismatch)
	}
	if !now().Before(expiresAt) {
		return "", ErrTokenExpired
	}
	plain, err := o.Open(sealed)
	if err != nil {
		return "", ErrInvalidToken
	}
	return string(plain), nil
}

func send(ctx context.Context, r Renderer, m Mailer, page string, data any, e Email) error {
	html, err := r.Render(page, data)
	if err != nil {
		return fmt.Errorf("render %s: %w: %w", page, ErrUndeliverable, err)
	}
	e.HTML = html
	return m.Send(ctx, e)
}
