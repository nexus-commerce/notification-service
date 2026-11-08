package usecase

import (
	"context"

	"github.com/nexus-commerce/notification-service/internal/event"
)

type SendEmailChangedNotice struct {
	renderer Renderer
	mailer   Mailer
}

func NewSendEmailChangedNotice(r Renderer, m Mailer) *SendEmailChangedNotice {
	return &SendEmailChangedNotice{renderer: r, mailer: m}
}

func (s *SendEmailChangedNotice) Execute(ctx context.Context, ev event.EmailChanged) error {
	data := struct {
		FirstName string
		NewEmail  string
	}{
		FirstName: ev.FirstName,
		NewEmail:  ev.NewEmail,
	}
	return send(ctx, s.renderer, s.mailer, PageEmailChanged, data, Email{
		ToName:    fullName(ev.FirstName, ev.LastName),
		ToAddress: ev.OldEmail,
		Subject:   "Your email address was changed",
		PlainText: "Your account email address was changed.",
	})
}
