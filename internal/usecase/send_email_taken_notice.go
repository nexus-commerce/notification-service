package usecase

import (
	"context"

	"github.com/nexus-commerce/notification-service/internal/event"
)

type SendEmailTakenNotice struct {
	renderer Renderer
	mailer   Mailer
}

func NewSendEmailTakenNotice(r Renderer, m Mailer) *SendEmailTakenNotice {
	return &SendEmailTakenNotice{renderer: r, mailer: m}
}

func (s *SendEmailTakenNotice) Execute(ctx context.Context, ev event.EmailChangeToTakenAddressAttempted) error {
	data := struct{ Email string }{Email: ev.NewEmail}
	return send(ctx, s.renderer, s.mailer, PageEmailTaken, data, Email{
		ToAddress: ev.NewEmail,
		Subject:   "Someone tried to use your email address",
		PlainText: "Someone tried to change their account email to your address.",
	})
}
