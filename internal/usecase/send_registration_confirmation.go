package usecase

import (
	"context"
	"time"

	"github.com/nexus-commerce/notification-service/internal/event"
)

type SendRegistrationConfirmation struct {
	renderer Renderer
	mailer   Mailer
	opener   Opener
	links    Links
	now      func() time.Time
}

func NewSendRegistrationConfirmation(r Renderer, m Mailer, o Opener, l Links, now func() time.Time) *SendRegistrationConfirmation {
	return &SendRegistrationConfirmation{
		renderer: r,
		mailer:   m,
		opener:   o,
		links:    l,
		now:      now,
	}
}

func (s *SendRegistrationConfirmation) Execute(ctx context.Context, ev event.RegistrationRequested) error {
	token, err := openToken(s.opener, s.now, ev.KeyID, ev.EncryptedToken, ev.ExpiresAt)
	if err != nil {
		return err
	}
	link := s.links.ConfirmRegistration(token)
	data := struct {
		FirstName string
		Link      string
		ExpiresAt time.Time
	}{
		FirstName: ev.FirstName,
		Link:      link,
		ExpiresAt: ev.ExpiresAt,
	}
	return send(ctx, s.renderer, s.mailer, PageConfirmRegistration, data, Email{
		ToName:    fullName(ev.FirstName, ev.LastName),
		ToAddress: ev.Email,
		Subject:   "Confirm your registration",
		PlainText: "Confirm your registration: " + link,
	})
}
