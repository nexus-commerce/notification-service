package usecase_test

import (
	"context"
	"testing"

	"github.com/nexus-commerce/notification-service/internal/event"
	"github.com/nexus-commerce/notification-service/internal/usecase"
)

func TestSendEmailTakenNotice(t *testing.T) {
	r := &fakeRenderer{}
	m := &fakeMailer{}
	uc := usecase.NewSendEmailTakenNotice(r, m)
	ev := event.EmailChangeToTakenAddressAttempted{UserID: "u-1", NewEmail: "jane@example.com"}

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
	if want := "Someone tried to use your email address"; got.Subject != want {
		t.Errorf("Subject = %q, want %q", got.Subject, want)
	}
	if got.PlainText == "" {
		t.Errorf("PlainText is empty")
	}
	if r.page != usecase.PageEmailTaken {
		t.Errorf("page = %q, want %q", r.page, usecase.PageEmailTaken)
	}
	data, ok := r.data.(struct{ Email string })
	if !ok {
		t.Fatalf("data type = %T, want struct{Email string}", r.data)
	}
	if data.Email != "jane@example.com" {
		t.Errorf("data.Email = %q, want %q", data.Email, "jane@example.com")
	}
}

func TestSendEmailTakenNoticeHasNoName(t *testing.T) {
	m := &fakeMailer{}
	uc := usecase.NewSendEmailTakenNotice(&fakeRenderer{}, m)

	if err := uc.Execute(context.Background(), event.EmailChangeToTakenAddressAttempted{NewEmail: "jane@example.com"}); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if len(m.sent) != 1 {
		t.Fatalf("len(sent) = %d, want 1", len(m.sent))
	}
	if got := m.sent[0].ToName; got != "" {
		t.Errorf("ToName = %q, want empty", got)
	}
}
