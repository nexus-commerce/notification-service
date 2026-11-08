package usecase_test

import (
	"context"
	"testing"

	"github.com/nexus-commerce/notification-service/internal/event"
	"github.com/nexus-commerce/notification-service/internal/usecase"
)

func orderConfirmed() event.OrderConfirmed {
	return event.OrderConfirmed{
		CustomerFirstName: "Jane",
		CustomerLastName:  "Doe",
		CustomerEmail:     "jane@example.com",
		OrderID:           42,
		Amount:            19.99,
		ShippingAddress:   "1 Main St",
	}
}

func TestSendOrderConfirmation(t *testing.T) {
	r := &fakeRenderer{}
	m := &fakeMailer{}
	uc := usecase.NewSendOrderConfirmation(r, m)

	if err := uc.Execute(context.Background(), orderConfirmed()); err != nil {
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
	if got.PlainText == "" {
		t.Errorf("PlainText is empty")
	}
	if r.page != usecase.PageOrderConfirmed {
		t.Errorf("page = %q, want %q", r.page, usecase.PageOrderConfirmed)
	}
	data, ok := r.data.(event.OrderConfirmed)
	if !ok {
		t.Fatalf("data type = %T, want event.OrderConfirmed", r.data)
	}
	if data.OrderID != 42 || data.ShippingAddress != "1 Main St" || data.Amount != 19.99 {
		t.Errorf("data = %+v, want the order event", data)
	}
}

func TestSendOrderConfirmationSubject(t *testing.T) {
	m := &fakeMailer{}
	uc := usecase.NewSendOrderConfirmation(&fakeRenderer{}, m)

	if err := uc.Execute(context.Background(), orderConfirmed()); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	if got, want := m.sent[0].Subject, "Your Order Confirmation - #42"; got != want {
		t.Errorf("Subject = %q, want %q", got, want)
	}
}
