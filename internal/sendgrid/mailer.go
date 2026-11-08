package sendgrid

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sendgrid/rest"
	sg "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/nexus-commerce/notification-service/internal/email"
	"github.com/nexus-commerce/notification-service/internal/notification"
)

// client is the part of the SendGrid client that Mailer uses. New wires in
// the real client; tests substitute a fake to control the response.
type client interface {
	SendWithContext(ctx context.Context, m *mail.SGMailV3) (*rest.Response, error)
}

type Mailer struct {
	client      client
	fromName    string
	fromAddress string
}

func New(apiKey, fromName, fromAddress string) *Mailer {
	return newMailer(sg.NewSendClient(apiKey), fromName, fromAddress)
}

func newMailer(c client, fromName, fromAddress string) *Mailer {
	return &Mailer{client: c, fromName: fromName, fromAddress: fromAddress}
}

func (m *Mailer) Send(ctx context.Context, e email.Email) error {
	msg := mail.NewSingleEmail(
		mail.NewEmail(m.fromName, m.fromAddress),
		e.Subject,
		mail.NewEmail(e.To.Name(), e.To.Address()),
		e.PlainText,
		e.HTML,
	)
	resp, err := m.client.SendWithContext(ctx, msg)
	if err != nil {
		return fmt.Errorf("sendgrid send: %w", err)
	}
	switch code := resp.StatusCode; {
	case code >= 200 && code < 300:
		return nil
	case retryableStatus(code):
		return fmt.Errorf("sendgrid status %d", code)
	default:
		return fmt.Errorf("%w: sendgrid status %d", notification.ErrUndeliverable, code)
	}
}

// retryableStatus reports whether a request SendGrid rejected with code may
// succeed on retry: rate limiting, credential problems an operator can fix,
// and server errors.
func retryableStatus(code int) bool {
	switch code {
	case http.StatusTooManyRequests, http.StatusUnauthorized, http.StatusForbidden:
		return true
	}
	return code >= 500
}
