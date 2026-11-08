// Package sendgrid delivers emails through the SendGrid API.
package sendgrid

import (
	"context"
	"fmt"
	"net/http"

	"github.com/sendgrid/rest"
	sg "github.com/sendgrid/sendgrid-go"
	"github.com/sendgrid/sendgrid-go/helpers/mail"

	"github.com/nexus-commerce/notification-service/internal/usecase"
)

type client interface {
	SendWithContext(ctx context.Context, m *mail.SGMailV3) (*rest.Response, error)
}

// Mailer sends usecase.Email through SendGrid.
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

// Send delivers e. Transport errors, 429 and 5xx are returned as plain
// (retryable) errors; other non-2xx statuses wrap usecase.ErrUndeliverable.
// Errors never include the message content or the response body.
func (m *Mailer) Send(ctx context.Context, e usecase.Email) error {
	msg := mail.NewSingleEmail(
		mail.NewEmail(m.fromName, m.fromAddress),
		e.Subject,
		mail.NewEmail(e.ToName, e.ToAddress),
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
	case code == http.StatusTooManyRequests || code >= 500:
		return fmt.Errorf("sendgrid status %d", code)
	default:
		return fmt.Errorf("%w: sendgrid status %d", usecase.ErrUndeliverable, code)
	}
}
