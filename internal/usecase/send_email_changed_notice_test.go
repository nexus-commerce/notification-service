package usecase_test

import (
	"context"
	"testing"

	"github.com/nexus-commerce/notification-service/internal/event"
	"github.com/nexus-commerce/notification-service/internal/usecase"
)

func TestSendEmailChangedNotice(t *testing.T) {
	r := &fakeRenderer{}
	m := &fakeMailer{}
	uc := usecase.NewSendEmailChangedNotice(r, m)
	ev := event.EmailChanged{
		UserID: "u-1", OldEmail: "old@example.com", NewEmail: "new@example.com",
		FirstName: "Jane", LastName: "Doe",
	}

	if err := uc.Execute(context.Background(), ev); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if len(m.sent) != 1 {
		t.Fatalf("len(sent) = %d, want 1", len(m.sent))
	}
	got := m.sent[0]
	if got.ToAddress != "old@example.com" {
		t.Errorf("ToAddress = %q, want %q", got.ToAddress, "old@example.com")
	}
	if got.ToName != "Jane Doe" {
		t.Errorf("ToName = %q, want %q", got.ToName, "Jane Doe")
	}
	if want := "Your email address was changed"; got.Subject != want {
		t.Errorf("Subject = %q, want %q", got.Subject, want)
	}
	if got.PlainText == "" {
		t.Errorf("PlainText is empty")
	}
	if r.page != usecase.PageEmailChanged {
		t.Errorf("page = %q, want %q", r.page, usecase.PageEmailChanged)
	}
	data, ok := r.data.(struct {
		FirstName string
		NewEmail  string
	})
	if !ok {
		t.Fatalf("data type = %T, want struct{FirstName, NewEmail string}", r.data)
	}
	if data.FirstName != "Jane" {
		t.Errorf("data.FirstName = %q, want %q", data.FirstName, "Jane")
	}
	if data.NewEmail != "new@example.com" {
		t.Errorf("data.NewEmail = %q, want %q", data.NewEmail, "new@example.com")
	}
}

func TestSendEmailChangedNoticeWithoutName(t *testing.T) {
	r := &fakeRenderer{}
	m := &fakeMailer{}
	uc := usecase.NewSendEmailChangedNotice(r, m)

	err := uc.Execute(context.Background(), event.EmailChanged{OldEmail: "old@example.com", NewEmail: "new@example.com"})
	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if len(m.sent) != 1 {
		t.Fatalf("len(sent) = %d, want 1", len(m.sent))
	}
	if got := m.sent[0].ToName; got != "" {
		t.Errorf("ToName = %q, want empty", got)
	}
	data, ok := r.data.(struct {
		FirstName string
		NewEmail  string
	})
	if !ok {
		t.Fatalf("data type = %T, want struct{FirstName, NewEmail string}", r.data)
	}
	if data.FirstName != "" {
		t.Errorf("data.FirstName = %q, want empty", data.FirstName)
	}
}
