package email

import (
	"context"
	"fmt"

	"github.com/nexus-commerce/notification-service/internal/notification"
)

type Renderer interface {
	Render(template string, data any) (string, error)
}

type Email struct {
	To        notification.Recipient
	Subject   string
	PlainText string
	HTML      string
}

type Mailer interface {
	Send(ctx context.Context, e Email) error
}

type Notifier struct {
	renderer Renderer
	mailer   Mailer
	links    Links
}

func NewNotifier(r Renderer, m Mailer, l Links) *Notifier {
	return &Notifier{renderer: r, mailer: m, links: l}
}

func (n *Notifier) Notify(ctx context.Context, msg notification.Message) error {
	c, err := n.compose(msg)
	if err != nil {
		return fmt.Errorf("%w: %w", notification.ErrUndeliverable, err)
	}
	html, err := n.renderer.Render(c.template, c.data)
	if err != nil {
		return fmt.Errorf("%w: render: %w", notification.ErrUndeliverable, err)
	}
	return n.mailer.Send(ctx, Email{
		To:        msg.Recipient(),
		Subject:   c.subject,
		PlainText: c.plainText,
		HTML:      html,
	})
}
