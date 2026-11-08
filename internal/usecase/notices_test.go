package usecase_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/nexus-commerce/notification-service/internal/notification"
	"github.com/nexus-commerce/notification-service/internal/usecase"
)

func TestSendWelcome(t *testing.T) {
	n := &fakeNotifier{}

	err := usecase.NewSendWelcome(n).Execute(t.Context(), notification.Registered{Recipient: jane})

	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if got, want := only[notification.Welcome](t, n), (notification.Welcome{To: jane}); got != want {
		t.Errorf("sent %+v, want %+v", got, want)
	}
}

func TestSendDuplicateRegistrationNotice(t *testing.T) {
	n := &fakeNotifier{}
	ev := notification.DuplicateRegistrationAttempted{Recipient: jane}

	err := usecase.NewSendDuplicateRegistrationNotice(n).Execute(t.Context(), ev)

	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if got, want := only[notification.DuplicateRegistrationNotice](t, n), (notification.DuplicateRegistrationNotice{To: jane}); got != want {
		t.Errorf("sent %+v, want %+v", got, want)
	}
}

func TestSendEmailTakenNotice(t *testing.T) {
	n := &fakeNotifier{}
	ev := notification.EmailChangeToTakenAddressAttempted{Recipient: anonymous}

	err := usecase.NewSendEmailTakenNotice(n).Execute(t.Context(), ev)

	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	if got, want := only[notification.EmailTakenNotice](t, n), (notification.EmailTakenNotice{To: anonymous}); got != want {
		t.Errorf("sent %+v, want %+v", got, want)
	}
}

func TestSendEmailChangedNotice(t *testing.T) {
	n := &fakeNotifier{}
	ev := notification.EmailChanged{Recipient: janeOldAddress, NewEmail: "new@example.com"}

	err := usecase.NewSendEmailChangedNotice(n).Execute(t.Context(), ev)

	if err != nil {
		t.Fatalf("Execute() error: %v", err)
	}
	want := notification.EmailChangedNotice{To: janeOldAddress, NewEmail: "new@example.com"}
	if got := only[notification.EmailChangedNotice](t, n); got != want {
		t.Errorf("sent %+v, want %+v", got, want)
	}
}

func TestSendOrderConfirmation(t *testing.T) {
	n := &fakeNotifier{}
	ordered := time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC)
	ev := notification.OrderConfirmed{
		Recipient:         jane,
		OrderID:           42,
		OrderDate:         ordered,
		PaidByCard:        true,
		Items:             []notification.OrderItem{{Quantity: 1, Price: 19.99, SKU: "S1", Name: "Mug", TotalPrice: 19.99}},
		Total:             19.99,
		ShippingAddress:   "1 Main St",
		EstimatedDelivery: ordered.Add(72 * time.Hour),
	}

	if err := usecase.NewSendOrderConfirmation(n).Execute(t.Context(), ev); err != nil {
		t.Fatalf("Execute() error: %v", err)
	}

	got := only[notification.OrderConfirmation](t, n)
	want := notification.OrderConfirmation{
		To:                jane,
		OrderID:           42,
		OrderDate:         ordered,
		PaidByCard:        true,
		Items:             []notification.OrderItem{{Quantity: 1, Price: 19.99, SKU: "S1", Name: "Mug", TotalPrice: 19.99}},
		Total:             19.99,
		ShippingAddress:   "1 Main St",
		EstimatedDelivery: ordered.Add(72 * time.Hour),
	}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("sent %+v, want %+v", got, want)
	}

	ev.Items[0].Name = "changed"
	if got.Items[0].Name != "Mug" {
		t.Errorf("message items share memory with the event")
	}
}
