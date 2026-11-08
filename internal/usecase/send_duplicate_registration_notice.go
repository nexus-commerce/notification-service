package usecase

import (
	"context"

	"github.com/nexus-commerce/notification-service/internal/event"
)

type SendDuplicateRegistrationNotice struct {
	renderer Renderer
	mailer   Mailer
	links    Links
}

func NewSendDuplicateRegistrationNotice(r Renderer, m Mailer, l Links) *SendDuplicateRegistrationNotice {
	return &SendDuplicateRegistrationNotice{renderer: r, mailer: m, links: l}
}

func (s *SendDuplicateRegistrationNotice) Execute(ctx context.Context, ev event.DuplicateRegistrationAttempted) error {
	data := struct {
		FirstName string
		LoginURL  string
	}{
		FirstName: ev.FirstName,
		LoginURL:  s.links.Login(),
	}
	return send(ctx, s.renderer, s.mailer, PageDuplicateRegistration, data, Email{
		ToName:    fullName(ev.FirstName, ev.LastName),
		ToAddress: ev.Email,
		Subject:   "Someone tried to register with your email",
		PlainText: "Someone tried to register with your email address; you already have an account.",
	})
}
