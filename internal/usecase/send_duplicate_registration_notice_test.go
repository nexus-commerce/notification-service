package usecase_test

import (
	"context"
	"testing"

	"github.com/nexus-commerce/notification-service/internal/event"
	"github.com/nexus-commerce/notification-service/internal/usecase"
)

func TestSendDuplicateRegistrationNotice(t *testing.T) {
	r := &fakeRenderer{}
	m := &fakeMailer{}
	uc := usecase.NewSendDuplicateRegistrationNotice(r, m, newLinks(t))
	ev := event.DuplicateRegistrationAttempted{UserID: "u-1", Email: "jane@example.com", FirstName: "Jane", LastName: "Doe"}

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
	if want := "Someone tried to register with your email"; got.Subject != want {
		t.Errorf("Subject = %q, want %q", got.Subject, want)
	}
	if got.PlainText == "" {
		t.Errorf("PlainText is empty")
	}
	if r.page != usecase.PageDuplicateRegistration {
		t.Errorf("page = %q, want %q", r.page, usecase.PageDuplicateRegistration)
	}
	data, ok := r.data.(struct {
		FirstName string
		LoginURL  string
	})
	if !ok {
		t.Fatalf("data type = %T, want struct{FirstName, LoginURL string}", r.data)
	}
	if data.FirstName != "Jane" {
		t.Errorf("data.FirstName = %q, want %q", data.FirstName, "Jane")
	}
	if want := "http://front.test/login"; data.LoginURL != want {
		t.Errorf("data.LoginURL = %q, want %q", data.LoginURL, want)
	}
}
