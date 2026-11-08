package render

import (
	"strings"
	"testing"
	"time"

	"github.com/nexus-commerce/notification-service/internal/event"
	"github.com/nexus-commerce/notification-service/internal/usecase"
)

func TestRenderEveryPage(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	expires := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	pm := "pi_1"
	tests := map[string]any{
		usecase.PageConfirmRegistration: struct {
			FirstName string
			Link      string
			ExpiresAt time.Time
		}{"Jane", "https://shop.example/confirm-registration?token=abc", expires},
		usecase.PageConfirmEmail: struct {
			FirstName string
			Link      string
			NewEmail  string
			ExpiresAt time.Time
		}{"Jane", "https://shop.example/confirm-email?token=abc", "new@example.com", expires},
		usecase.PageWelcome: struct {
			FirstName string
			LoginURL  string
		}{"Jane", "https://shop.example/login"},
		usecase.PageDuplicateRegistration: struct {
			FirstName string
			LoginURL  string
		}{"Jane", "https://shop.example/login"},
		usecase.PageEmailTaken: struct{ Email string }{"jane@example.com"},
		usecase.PageEmailChanged: struct {
			FirstName string
			NewEmail  string
		}{"Jane", "new@example.com"},
		usecase.PageOrderConfirmed: event.OrderConfirmed{
			CustomerFirstName: "Jane",
			OrderID:           7,
			OrderDate:         expires,
			PaymentMethod:     "CARD",
			PaymentIntentID:   &pm,
			Items:             []*event.OrderItem{{Quantity: 1, Price: 2.5, Sku: "S", Name: "Thing", ItemTotalPrice: 2.5}},
			Amount:            2.5,
			ShippingAddress:   "1 Main St",
			EstimatedDelivery: expires,
		},
	}
	for page, data := range tests {
		t.Run(page, func(t *testing.T) {
			got, err := r.Render(page, data)
			if err != nil {
				t.Fatalf("Render(%q) error: %v", page, err)
			}
			if got == "" {
				t.Errorf("Render(%q) = empty, want non-empty", page)
			}
		})
	}
}

func TestRenderGreeting(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	for name, tc := range map[string]struct{ first, want string }{
		"anonymous": {"", "Hello,"},
		"named":     {"Jane", "Hi Jane,"},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := r.Render(usecase.PageWelcome, struct {
				FirstName string
				LoginURL  string
			}{tc.first, "https://shop.example/login"})
			if err != nil {
				t.Fatalf("Render() error: %v", err)
			}
			if !strings.Contains(got, tc.want) {
				t.Errorf("Render() = %q, want it to contain %q", got, tc.want)
			}
		})
	}
}

func TestRenderOrderGreeting(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	when := time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	for name, tc := range map[string]struct{ first, want, notWant string }{
		"anonymous": {"", "Hello,", "Hi ,"},
		"named":     {"Jane", "Hi Jane,", "Hello,"},
	} {
		t.Run(name, func(t *testing.T) {
			got, err := r.Render(usecase.PageOrderConfirmed, event.OrderConfirmed{
				CustomerFirstName: tc.first,
				OrderDate:         when,
				EstimatedDelivery: when,
				PaymentMethod:     "CARD",
			})
			if err != nil {
				t.Fatalf("Render() error: %v", err)
			}
			if !strings.Contains(got, tc.want) {
				t.Errorf("Render() = %q, want it to contain %q", got, tc.want)
			}
			if strings.Contains(got, tc.notWant) {
				t.Errorf("Render() = %q, want it not to contain %q", got, tc.notWant)
			}
		})
	}
}

func TestRenderConfirmationShowsLink(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	link := "https://shop.example/confirm-registration?token=abc"
	got, err := r.Render(usecase.PageConfirmRegistration, struct {
		FirstName string
		Link      string
		ExpiresAt time.Time
	}{"Jane", link, time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}
	if !strings.Contains(got, link) {
		t.Errorf("Render() = %q, want it to contain %q", got, link)
	}
	if !strings.Contains(got, "Nov 8, 2026 12:00 UTC") {
		t.Errorf("Render() = %q, want formatted expiry", got)
	}
}

func TestRenderUnknownPage(t *testing.T) {
	r, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if _, err := r.Render("nope", nil); err == nil {
		t.Error("Render(unknown) error = nil, want error")
	}
}
