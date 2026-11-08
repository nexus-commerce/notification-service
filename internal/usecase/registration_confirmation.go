package usecase

import (
	"context"
	"time"

	"github.com/nexus-commerce/notification-service/internal/notification"
)

type SendRegistrationConfirmation struct {
	notifier Notifier
	opener   notification.Opener
	now      func() time.Time
}

func NewSendRegistrationConfirmation(n Notifier, o notification.Opener, now func() time.Time) *SendRegistrationConfirmation {
	return &SendRegistrationConfirmation{notifier: n, opener: o, now: now}
}

func (s *SendRegistrationConfirmation) Execute(ctx context.Context, ev notification.RegistrationRequested) error {
	msg, err := ev.Confirmation(s.opener, s.now())
	if err != nil {
		return err
	}
	return s.notifier.Notify(ctx, msg)
}
