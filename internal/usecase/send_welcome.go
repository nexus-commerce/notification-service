package usecase

import (
	"context"

	"github.com/nexus-commerce/notification-service/internal/event"
)

type SendWelcome struct {
	renderer Renderer
	mailer   Mailer
	links    Links
}

func NewSendWelcome(r Renderer, m Mailer, l Links) *SendWelcome {
	return &SendWelcome{renderer: r, mailer: m, links: l}
}

func (s *SendWelcome) Execute(ctx context.Context, ev event.Registered) error {
	data := struct {
		FirstName string
		LoginURL  string
	}{
		FirstName: ev.FirstName,
		LoginURL:  s.links.Login(),
	}
	return send(ctx, s.renderer, s.mailer, PageWelcome, data, Email{
		ToName:    fullName(ev.FirstName, ev.LastName),
		ToAddress: ev.Email,
		Subject:   "Welcome to MyEcom",
		PlainText: "Welcome to MyEcom! Your account is ready.",
	})
}
