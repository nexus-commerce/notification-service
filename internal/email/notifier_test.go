package email

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/nexus-commerce/notification-service/internal/notification"
	"github.com/nexus-commerce/notification-service/internal/render"
)

type fakeRenderer struct {
	template string
	data     any
	err      error
}

func (f *fakeRenderer) Render(template string, data any) (string, error) {
	f.template = template
	f.data = data
	if f.err != nil {
		return "", f.err
	}
	return "<html>" + template, nil
}

type fakeMailer struct {
	sent []Email
	err  error
}

func (f *fakeMailer) Send(_ context.Context, e Email) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, e)
	return nil
}

var (
	when           = time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
	jane           = mustRecipient("Jane", "Doe", "jane@example.com")
	janeNewAddress = mustRecipient("Jane", "Doe", "new@example.com")
	anonymous      = mustRecipient("", "", "jane@example.com")
)

func newLinks(t *testing.T) Links {
	t.Helper()
	l, err := NewLinks("http://front.test")
	if err != nil {
		t.Fatalf("NewLinks() error: %v", err)
	}
	return l
}

func revealed(t *testing.T, plain string) notification.RevealedToken {
	t.Helper()
	tok, err := notification.NewConfirmationToken([]byte("sealed"), "k", when.Add(time.Hour))
	if err != nil {
		t.Fatalf("NewConfirmationToken() error: %v", err)
	}
	r, err := tok.Reveal(plainOpener(plain), when)
	if err != nil {
		t.Fatalf("Reveal() error: %v", err)
	}
	return r
}

type plainOpener string

func (p plainOpener) Open([]byte) ([]byte, error) { return []byte(p), nil }
func (plainOpener) KeyID() string                 { return "k" }

func order(paidByCard bool) notification.OrderConfirmation {
	return notification.OrderConfirmation{
		To:                jane,
		OrderID:           42,
		OrderDate:         when,
		PaidByCard:        paidByCard,
		Items:             []notification.OrderItem{{Quantity: 1, Price: 2.5, SKU: "S1", Name: "Mug", TotalPrice: 2.5}},
		Total:             2.5,
		ShippingAddress:   "1 Main St",
		EstimatedDelivery: when.Add(72 * time.Hour),
	}
}

type wantEmail struct {
	template  string
	subject   string
	plainText string
	data      any
}

type messageCase struct {
	msg  notification.Message
	want wantEmail
}

func messages(t *testing.T) map[string]messageCase {
	const (
		regLink   = "http://front.test/confirm-registration?token=tok+en%2F1"
		emailLink = "http://front.test/confirm-email?token=tok+en%2F1"
		loginURL  = "http://front.test/login"
	)
	return map[string]messageCase{
		"registration confirmation": {
			notification.RegistrationConfirmation{To: jane, Token: revealed(t, "tok en/1"), ExpiresAt: when},
			wantEmail{templateConfirmRegistration, "Confirm your registration", regLink,
				confirmRegistrationData{Greeting: "Hi Jane,", Link: regLink, ExpiresAt: when}},
		},
		"welcome": {
			notification.Welcome{To: jane},
			wantEmail{templateWelcome, "Welcome to MyEcom", "Your account is ready",
				welcomeData{Greeting: "Hi Jane,", LoginURL: loginURL}},
		},
		"duplicate registration": {
			notification.DuplicateRegistrationNotice{To: jane},
			wantEmail{templateDuplicateRegistration, "Someone tried to register with your email", "already have an account",
				duplicateRegistrationData{Greeting: "Hi Jane,", LoginURL: loginURL}},
		},
		"email change confirmation": {
			notification.EmailChangeConfirmation{To: janeNewAddress, Token: revealed(t, "tok en/1"), ExpiresAt: when},
			wantEmail{templateConfirmEmail, "Confirm your new email address", emailLink,
				confirmEmailData{Greeting: "Hi Jane,", Link: emailLink, NewEmail: "new@example.com", ExpiresAt: when}},
		},
		"email taken": {
			notification.EmailTakenNotice{To: anonymous},
			wantEmail{templateEmailTaken, "Someone tried to use your email address", "change their account email",
				emailTakenData{Email: "jane@example.com"}},
		},
		"email changed": {
			notification.EmailChangedNotice{To: jane, NewEmail: "new@example.com"},
			wantEmail{templateEmailChanged, "Your email address was changed", "was changed",
				emailChangedData{Greeting: "Hi Jane,", NewEmail: "new@example.com"}},
		},
		"order confirmation": {
			order(true),
			wantEmail{templateOrderConfirmed, "Your Order Confirmation - #42", "Thank you for your order", nil},
		},
	}
}

func TestNotify(t *testing.T) {
	for name, tt := range messages(t) {
		t.Run(name, func(t *testing.T) {
			fakeR := &fakeRenderer{}
			fakeM := &fakeMailer{}

			if err := NewNotifier(fakeR, fakeM, newLinks(t)).Notify(t.Context(), tt.msg); err != nil {
				t.Fatalf("Notify() error: %v", err)
			}

			if len(fakeM.sent) != 1 {
				t.Fatalf("len(sent) = %d, want 1", len(fakeM.sent))
			}
			got := fakeM.sent[0]
			if got.To != tt.msg.Recipient() {
				t.Errorf("To = %+v, want %+v", got.To, tt.msg.Recipient())
			}
			if got.Subject != tt.want.subject {
				t.Errorf("Subject = %q, want %q", got.Subject, tt.want.subject)
			}
			if !strings.Contains(got.PlainText, tt.want.plainText) {
				t.Errorf("PlainText = %q, want it to contain %q", got.PlainText, tt.want.plainText)
			}
			if want := "<html>" + tt.want.template; got.HTML != want {
				t.Errorf("HTML = %q, want %q", got.HTML, want)
			}
			if fakeR.template != tt.want.template {
				t.Errorf("template = %q, want %q", fakeR.template, tt.want.template)
			}
			if tt.want.data != nil && fakeR.data != tt.want.data {
				t.Errorf("data = %+v, want %+v", fakeR.data, tt.want.data)
			}
		})
	}
}

func TestNotifyOrderData(t *testing.T) {
	fakeR := &fakeRenderer{}
	msg := order(false)

	if err := NewNotifier(fakeR, &fakeMailer{}, newLinks(t)).Notify(t.Context(), msg); err != nil {
		t.Fatalf("Notify() error: %v", err)
	}

	d, ok := fakeR.data.(orderConfirmedData)
	if !ok {
		t.Fatalf("data type = %T, want orderConfirmedData", fakeR.data)
	}
	want := orderConfirmedData{
		Greeting:          "Hi Jane,",
		OrderID:           42,
		OrderDate:         when,
		ShippingAddress:   "1 Main St",
		EstimatedDelivery: when.Add(72 * time.Hour),
		PaidByCard:        false,
		Items:             []orderItemData{{Quantity: 1, Price: 2.5, SKU: "S1", Name: "Mug", TotalPrice: 2.5}},
		Total:             2.5,
	}
	if diff := cmp.Diff(want, d); diff != "" {
		t.Errorf("Notify() render data mismatch (-want +got):\n%s", diff)
	}
}

func TestNotifyGreetsAnonymousRecipient(t *testing.T) {
	fakeR := &fakeRenderer{}

	n := NewNotifier(fakeR, &fakeMailer{}, newLinks(t))
	notice := notification.EmailChangedNotice{To: anonymous, NewEmail: "n@x"}

	if err := n.Notify(t.Context(), notice); err != nil {
		t.Fatalf("Notify() error: %v", err)
	}

	if d := fakeR.data.(emailChangedData); d.Greeting != "Hello," {
		t.Errorf("Greeting = %q, want %q", d.Greeting, "Hello,")
	}
}

func TestNotifyRenderFailureIsUndeliverable(t *testing.T) {
	fakeM := &fakeMailer{}
	n := NewNotifier(&fakeRenderer{err: errors.New("template broken")}, fakeM, newLinks(t))

	err := n.Notify(t.Context(), notification.Welcome{To: jane})

	if !errors.Is(err, notification.ErrUndeliverable) {
		t.Errorf("Notify() error = %v, want %v", err, notification.ErrUndeliverable)
	}
	if len(fakeM.sent) != 0 {
		t.Errorf("len(sent) = %d, want 0", len(fakeM.sent))
	}
}

func TestNotifyUnknownMessageIsUndeliverable(t *testing.T) {
	fakeM := &fakeMailer{}

	err := NewNotifier(&fakeRenderer{}, fakeM, newLinks(t)).Notify(t.Context(), nil)

	if !errors.Is(err, notification.ErrUndeliverable) {
		t.Errorf("Notify() error = %v, want %v", err, notification.ErrUndeliverable)
	}
	if len(fakeM.sent) != 0 {
		t.Errorf("len(sent) = %d, want 0", len(fakeM.sent))
	}
}

func TestNotifyReturnsMailerError(t *testing.T) {
	mailErr := errors.New("smtp down")
	n := NewNotifier(&fakeRenderer{}, &fakeMailer{err: mailErr}, newLinks(t))

	err := n.Notify(t.Context(), notification.Welcome{To: jane})

	if !errors.Is(err, mailErr) {
		t.Errorf("Notify() error = %v, want %v", err, mailErr)
	}
	if errors.Is(err, notification.ErrUndeliverable) {
		t.Errorf("Notify() error = %v, want not %v", err, notification.ErrUndeliverable)
	}
}

func TestEveryMessageRendersWithRealTemplates(t *testing.T) {
	r, err := render.New("help@shop.example")
	if err != nil {
		t.Fatalf("render.New() error: %v", err)
	}
	for name, tt := range messages(t) {
		t.Run(name, func(t *testing.T) {
			fakeM := &fakeMailer{}

			if err := NewNotifier(r, fakeM, newLinks(t)).Notify(t.Context(), tt.msg); err != nil {
				t.Fatalf("Notify() error: %v", err)
			}

			want := "Hi Jane,"
			if tt.msg.Recipient() == anonymous {
				want = "Hello,"
			}
			if html := fakeM.sent[0].HTML; !strings.Contains(html, want) {
				t.Errorf("HTML = %q, want it to contain greeting %q", html, want)
			}
		})
	}
}

func TestOrderEmailShowsPaymentMethod(t *testing.T) {
	r, err := render.New("help@shop.example")
	if err != nil {
		t.Fatalf("render.New() error: %v", err)
	}
	for name, tc := range map[string]struct {
		paidByCard    bool
		want, notWant string
	}{
		"card":        {true, "Credit Card", "On Delivery"},
		"on delivery": {false, "On Delivery", "Credit Card"},
	} {
		t.Run(name, func(t *testing.T) {
			fakeM := &fakeMailer{}

			if err := NewNotifier(r, fakeM, newLinks(t)).Notify(t.Context(), order(tc.paidByCard)); err != nil {
				t.Fatalf("Notify() error: %v", err)
			}

			html := fakeM.sent[0].HTML
			if !strings.Contains(html, tc.want) {
				t.Errorf("HTML = %q, want it to contain %q", html, tc.want)
			}
			if strings.Contains(html, tc.notWant) {
				t.Errorf("HTML = %q, want it not to contain %q", html, tc.notWant)
			}
		})
	}
}

func TestConfirmationEmailShowsLinkAndExpiry(t *testing.T) {
	r, err := render.New("help@shop.example")
	if err != nil {
		t.Fatalf("render.New() error: %v", err)
	}
	fakeM := &fakeMailer{}
	msg := notification.RegistrationConfirmation{To: jane, Token: revealed(t, "abc"), ExpiresAt: when}

	if err := NewNotifier(r, fakeM, newLinks(t)).Notify(t.Context(), msg); err != nil {
		t.Fatalf("Notify() error: %v", err)
	}

	html := fakeM.sent[0].HTML
	if link := "http://front.test/confirm-registration?token=abc"; !strings.Contains(html, link) {
		t.Errorf("HTML = %q, want it to contain %q", html, link)
	}
	if !strings.Contains(html, "Nov 8, 2026 12:00 UTC") {
		t.Errorf("HTML = %q, want formatted expiry", html)
	}
}

// mustRecipient builds a package-level test recipient and panics on an
// invalid one. Inside a test, construct recipients with a helper that calls
// t.Fatal instead.
func mustRecipient(firstName, lastName, address string) notification.Recipient {
	r, err := notification.NewRecipient(firstName, lastName, address)
	if err != nil {
		panic(err)
	}
	return r
}
