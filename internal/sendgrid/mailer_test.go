package sendgrid

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/nexus-commerce/notification-service/internal/email"
	"github.com/nexus-commerce/notification-service/internal/notification"
)

type fakeClient struct {
	resp *rest.Response
	err  error
	got  *mail.SGMailV3
}

func (f *fakeClient) SendWithContext(_ context.Context, m *mail.SGMailV3) (*rest.Response, error) {
	f.got = m
	return f.resp, f.err
}

var testEmail = email.Email{
	To:      mustRecipient("Jane", "Doe", "jane@example.com"),
	Subject: "Hi", PlainText: "plain", HTML: "<p>html</p>",
}

func TestSendClassifiesStatus(t *testing.T) {
	tests := []struct {
		code          int
		wantErr       bool
		undeliverable bool
	}{
		{http.StatusAccepted, false, false},
		{http.StatusTooManyRequests, true, false},
		{http.StatusServiceUnavailable, true, false},
		{http.StatusUnauthorized, true, false},
		{http.StatusForbidden, true, false},
		{http.StatusBadRequest, true, true},
	}
	for _, tc := range tests {
		t.Run(http.StatusText(tc.code), func(t *testing.T) {
			c := &fakeClient{resp: &rest.Response{StatusCode: tc.code, Body: "secret-body"}}
			err := newMailer(c, "NexusEcommerce", "from@example.com").Send(t.Context(), testEmail)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Send() error = %v, wantErr %v", err, tc.wantErr)
			}
			if got := errors.Is(err, notification.ErrUndeliverable); got != tc.undeliverable {
				t.Errorf("Send() undeliverable = %v, want %v", got, tc.undeliverable)
			}
			if err != nil && strings.Contains(err.Error(), "secret-body") {
				t.Errorf("Send() error leaks response body: %v", err)
			}
		})
	}
}

func TestSendTransportError(t *testing.T) {
	c := &fakeClient{err: errors.New("connection reset")}
	err := newMailer(c, "NexusEcommerce", "from@example.com").Send(t.Context(), testEmail)
	if err == nil {
		t.Fatal("Send() error = nil, want error")
	}
	if errors.Is(err, notification.ErrUndeliverable) {
		t.Errorf("Send() error = %v, want retryable", err)
	}
}

func TestSendBuildsMessage(t *testing.T) {
	c := &fakeClient{resp: &rest.Response{StatusCode: 202}}
	if err := newMailer(c, "NexusEcommerce", "from@example.com").Send(t.Context(), testEmail); err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	m := c.got
	if m.From.Name != "NexusEcommerce" || m.From.Address != "from@example.com" {
		t.Errorf("From = %+v, want NexusEcommerce <from@example.com>", m.From)
	}
	if m.Subject != "Hi" {
		t.Errorf("Subject = %q, want %q", m.Subject, "Hi")
	}
	to := m.Personalizations[0].To[0]
	if to.Name != "Jane Doe" || to.Address != "jane@example.com" {
		t.Errorf("To = %+v, want Jane Doe <jane@example.com>", to)
	}
}

// mustRecipient builds a valid test recipient and panics on an invalid one.
func mustRecipient(firstName, lastName, address string) notification.Recipient {
	r, err := notification.NewRecipient(firstName, lastName, address)
	if err != nil {
		panic(err)
	}
	return r
}
