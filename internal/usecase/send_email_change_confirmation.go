package usecase

import (
	"context"
	"time"

	"github.com/nexus-commerce/notification-service/internal/event"
)

type SendEmailChangeConfirmation struct {
	renderer Renderer
	mailer   Mailer
	opener   Opener
	links    Links
	now      func() time.Time
}

func NewSendEmailChangeConfirmation(r Renderer, m Mailer, o Opener, l Links, now func() time.Time) *SendEmailChangeConfirmation {
	return &SendEmailChangeConfirmation{
		renderer: r,
		mailer:   m,
		opener:   o,
		links:    l,
		now:      now,
	}
}

func (s *SendEmailChangeConfirmation) Execute(ctx context.Context, ev event.EmailChangeRequested) error {
	token, err := openToken(s.opener, s.now, ev.KeyID, ev.EncryptedToken, ev.ExpiresAt)
	if err != nil {
		return err
	}
	data := struct {
		FirstName string
		Link      string
		NewEmail  string
		ExpiresAt time.Time
	}{
		FirstName: ev.FirstName,
		Link:      s.links.ConfirmEmail(token),
		NewEmail:  ev.NewEmail,
		ExpiresAt: ev.ExpiresAt,
	}
	return send(ctx, s.renderer, s.mailer, PageConfirmEmail, data, Email{
		ToName:    fullName(ev.FirstName, ev.LastName),
		ToAddress: ev.NewEmail,
		Subject:   "Confirm your new email address",
	})
}
