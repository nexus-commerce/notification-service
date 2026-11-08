package sendgrid

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"

	"github.com/sendgrid/rest"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/nexus-commerce/notification-service/internal/usecase"
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

var email = usecase.Email{
	ToName: "Jane Doe", ToAddress: "jane@example.com",
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
		{http.StatusBadRequest, true, true},
	}
	for _, tc := range tests {
		t.Run(http.StatusText(tc.code), func(t *testing.T) {
			c := &fakeClient{resp: &rest.Response{StatusCode: tc.code, Body: "secret-body"}}
			err := newMailer(c, "MyEcom", "from@example.com").Send(context.Background(), email)
			if (err != nil) != tc.wantErr {
				t.Fatalf("Send() error = %v, wantErr %v", err, tc.wantErr)
			}
			if got := errors.Is(err, usecase.ErrUndeliverable); got != tc.undeliverable {
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
	err := newMailer(c, "MyEcom", "from@example.com").Send(context.Background(), email)
	if err == nil {
		t.Fatal("Send() error = nil, want error")
	}
	if errors.Is(err, usecase.ErrUndeliverable) {
		t.Errorf("Send() error = %v, want retryable", err)
	}
}

func TestSendBuildsMessage(t *testing.T) {
	c := &fakeClient{resp: &rest.Response{StatusCode: 202}}
	if err := newMailer(c, "MyEcom", "from@example.com").Send(context.Background(), email); err != nil {
		t.Fatalf("Send() error: %v", err)
	}
	m := c.got
	if m.From.Name != "MyEcom" || m.From.Address != "from@example.com" {
		t.Errorf("From = %+v, want MyEcom <from@example.com>", m.From)
	}
	if m.Subject != "Hi" {
		t.Errorf("Subject = %q, want %q", m.Subject, "Hi")
	}
	to := m.Personalizations[0].To[0]
	if to.Name != "Jane Doe" || to.Address != "jane@example.com" {
		t.Errorf("To = %+v, want Jane Doe <jane@example.com>", to)
	}
}
