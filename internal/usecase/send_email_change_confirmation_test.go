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

func emailChangeRequested() event.EmailChangeRequested {
	return event.EmailChangeRequested{
		UserID:         "u-1",
		NewEmail:       "new@example.com",
		FirstName:      "Jane",
		LastName:       "Doe",
		EncryptedToken: []byte(testSealed),
		KeyID:          testKeyID,
		ExpiresAt:      fixedNow().Add(time.Hour),
	}
}

func TestSendEmailChangeConfirmation(t *testing.T) {
	r := &fakeRenderer{}
	m := &fakeMailer{}
	uc := usecase.NewSendEmailChangeConfirmation(r, m, newOpener(), newLinks(t), fixedNow)
	ev := emailChangeRequested()

	if err := uc.Execute(context.Background(), ev); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if len(m.sent) != 1 {
		t.Fatalf("len(sent) = %d, want 1", len(m.sent))
	}
	got := m.sent[0]
	if got.ToAddress != "new@example.com" {
		t.Errorf("ToAddress = %q, want %q", got.ToAddress, "new@example.com")
	}
	if got.ToName != "Jane Doe" {
		t.Errorf("ToName = %q, want %q", got.ToName, "Jane Doe")
	}
	if got.Subject != "Confirm your new email address" {
		t.Errorf("Subject = %q, want %q", got.Subject, "Confirm your new email address")
	}
	if got.HTML != "<html>confirm-email" {
		t.Errorf("HTML = %q, want %q", got.HTML, "<html>confirm-email")
	}
	if wantLink := "http://front.test/confirm-email?token=tok+en%2F1"; !strings.Contains(got.PlainText, wantLink) {
		t.Errorf("PlainText = %q, want it to contain %q", got.PlainText, wantLink)
	}
	if r.page != usecase.PageConfirmEmail {
		t.Errorf("page = %q, want %q", r.page, usecase.PageConfirmEmail)
	}
	data, ok := r.data.(struct {
		FirstName string
		Link      string
		NewEmail  string
		ExpiresAt time.Time
	})
	if !ok {
		t.Fatalf("data type = %T, want struct{FirstName, Link, NewEmail string; ExpiresAt time.Time}", r.data)
	}
	wantLink := "http://front.test/confirm-email?token=tok+en%2F1"
	if data.Link != wantLink {
		t.Errorf("data.Link = %q, want %q", data.Link, wantLink)
	}
	if data.NewEmail != "new@example.com" {
		t.Errorf("data.NewEmail = %q, want %q", data.NewEmail, "new@example.com")
	}
	if data.FirstName != "Jane" {
		t.Errorf("data.FirstName = %q, want %q", data.FirstName, "Jane")
	}
	if !data.ExpiresAt.Equal(ev.ExpiresAt) {
		t.Errorf("data.ExpiresAt = %v, want %v", data.ExpiresAt, ev.ExpiresAt)
	}
}

func TestSendEmailChangeConfirmationSkipsExpired(t *testing.T) {
	m := &fakeMailer{}
	uc := usecase.NewSendEmailChangeConfirmation(&fakeRenderer{}, m, newOpener(), newLinks(t), fixedNow)
	ev := emailChangeRequested()
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

func TestSendEmailChangeConfirmationRejectsKeyMismatch(t *testing.T) {
	m := &fakeMailer{}
	uc := usecase.NewSendEmailChangeConfirmation(&fakeRenderer{}, m, newOpener(), newLinks(t), fixedNow)
	ev := emailChangeRequested()
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
