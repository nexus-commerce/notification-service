package usecase_test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/nexus-commerce/notification-service/internal/event"
	"github.com/nexus-commerce/notification-service/internal/usecase"
)

func registrationRequested() event.RegistrationRequested {
	return event.RegistrationRequested{
		UserID:         "u-1",
		Email:          "jane@example.com",
		FirstName:      "Jane",
		LastName:       "Doe",
		EncryptedToken: []byte(testSealed),
		KeyID:          testKeyID,
		ExpiresAt:      fixedNow().Add(time.Hour),
	}
}

func TestSendRegistrationConfirmation(t *testing.T) {
	r := &fakeRenderer{}
	m := &fakeMailer{}
	uc := usecase.NewSendRegistrationConfirmation(r, m, newOpener(), newLinks(t), fixedNow)
	ev := registrationRequested()

	if err := uc.Execute(context.Background(), ev); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if len(m.sent) != 1 {
		t.Fatalf("len(sent) = %d, want 1", len(m.sent))
	}
	got := m.sent[0]
	if got.ToAddress != "jane@example.com" {
		t.Errorf("ToAddress = %q, want %q", got.ToAddress, "jane@example.com")
	}
	if got.ToName != "Jane Doe" {
		t.Errorf("ToName = %q, want %q", got.ToName, "Jane Doe")
	}
	if got.Subject != "Confirm your registration" {
		t.Errorf("Subject = %q, want %q", got.Subject, "Confirm your registration")
	}
	if got.HTML != "<html>confirm-registration" {
		t.Errorf("HTML = %q, want %q", got.HTML, "<html>confirm-registration")
	}
	if r.page != usecase.PageConfirmRegistration {
		t.Errorf("page = %q, want %q", r.page, usecase.PageConfirmRegistration)
	}
	data, ok := r.data.(struct {
		FirstName string
		Link      string
		ExpiresAt time.Time
	})
	if !ok {
		t.Fatalf("data type = %T, want struct{FirstName, Link string; ExpiresAt time.Time}", r.data)
	}
	wantLink := "http://front.test/confirm-registration?token=tok+en%2F1"
	if data.Link != wantLink {
		t.Errorf("data.Link = %q, want %q", data.Link, wantLink)
	}
	if data.FirstName != "Jane" {
		t.Errorf("data.FirstName = %q, want %q", data.FirstName, "Jane")
	}
	if !data.ExpiresAt.Equal(ev.ExpiresAt) {
		t.Errorf("data.ExpiresAt = %v, want %v", data.ExpiresAt, ev.ExpiresAt)
	}
}

func TestSendRegistrationConfirmationSkipsExpired(t *testing.T) {
	m := &fakeMailer{}
	uc := usecase.NewSendRegistrationConfirmation(&fakeRenderer{}, m, newOpener(), newLinks(t), fixedNow)
	ev := registrationRequested()
	ev.ExpiresAt = fixedNow()

	err := uc.Execute(context.Background(), ev)

	if !errors.Is(err, usecase.ErrTokenExpired) {
		t.Errorf("Execute() error = %v, want %v", err, usecase.ErrTokenExpired)
	}
	if !errors.Is(err, usecase.ErrUndeliverable) {
		t.Errorf("Execute() error = %v, want %v", err, usecase.ErrUndeliverable)
	}
	if len(m.sent) != 0 {
		t.Errorf("len(sent) = %d, want 0", len(m.sent))
	}
}

func TestSendRegistrationConfirmationRejectsKeyMismatch(t *testing.T) {
	m := &fakeMailer{}
	uc := usecase.NewSendRegistrationConfirmation(&fakeRenderer{}, m, newOpener(), newLinks(t), fixedNow)
	ev := registrationRequested()
	ev.KeyID = "other-key"

	err := uc.Execute(context.Background(), ev)

	if !errors.Is(err, usecase.ErrKeyMismatch) {
		t.Errorf("Execute() error = %v, want %v", err, usecase.ErrKeyMismatch)
	}
	if !errors.Is(err, usecase.ErrUndeliverable) {
		t.Errorf("Execute() error = %v, want %v", err, usecase.ErrUndeliverable)
	}
	if len(m.sent) != 0 {
		t.Errorf("len(sent) = %d, want 0", len(m.sent))
	}
}

func TestSendRegistrationConfirmationRejectsUnopenableToken(t *testing.T) {
	m := &fakeMailer{}
	uc := usecase.NewSendRegistrationConfirmation(&fakeRenderer{}, m, newOpener(), newLinks(t), fixedNow)
	ev := registrationRequested()
	ev.EncryptedToken = []byte("garbage")

	err := uc.Execute(context.Background(), ev)

	if !errors.Is(err, usecase.ErrInvalidToken) {
		t.Errorf("Execute() error = %v, want %v", err, usecase.ErrInvalidToken)
	}
	if !errors.Is(err, usecase.ErrUndeliverable) {
		t.Errorf("Execute() error = %v, want %v", err, usecase.ErrUndeliverable)
	}
	if err != nil && strings.Contains(err.Error(), "tok") {
		t.Errorf("Execute() error = %q, must not contain token material", err.Error())
	}
	if len(m.sent) != 0 {
		t.Errorf("len(sent) = %d, want 0", len(m.sent))
	}
}

func TestSendRegistrationConfirmationReturnsMailerError(t *testing.T) {
	mailErr := errors.New("smtp down")
	m := &fakeMailer{err: mailErr}
	uc := usecase.NewSendRegistrationConfirmation(&fakeRenderer{}, m, newOpener(), newLinks(t), fixedNow)

	err := uc.Execute(context.Background(), registrationRequested())

	if !errors.Is(err, mailErr) {
		t.Errorf("Execute() error = %v, want %v", err, mailErr)
	}
	if errors.Is(err, usecase.ErrUndeliverable) {
		t.Errorf("Execute() error = %v, want not %v", err, usecase.ErrUndeliverable)
	}
}

func TestSendRegistrationConfirmationRenderFailureIsUndeliverable(t *testing.T) {
	m := &fakeMailer{}
	r := &fakeRenderer{err: errors.New("template broken")}
	uc := usecase.NewSendRegistrationConfirmation(r, m, newOpener(), newLinks(t), fixedNow)

	err := uc.Execute(context.Background(), registrationRequested())

	if !errors.Is(err, usecase.ErrUndeliverable) {
		t.Errorf("Execute() error = %v, want %v", err, usecase.ErrUndeliverable)
	}
	if len(m.sent) != 0 {
		t.Errorf("len(sent) = %d, want 0", len(m.sent))
	}
}

func TestNewLinksTrimsTrailingSlash(t *testing.T) {
	l, err := usecase.NewLinks("http://front.test/")
	if err != nil {
		t.Fatalf("NewLinks() error: %v", err)
	}

	if got, want := l.Login(), "http://front.test/login"; got != want {
		t.Errorf("Login() = %q, want %q", got, want)
	}
}

func TestNewLinksRejectsInvalidURL(t *testing.T) {
	if _, err := usecase.NewLinks("://"); err == nil {
		t.Errorf("NewLinks() error = nil, want error")
	}
}
