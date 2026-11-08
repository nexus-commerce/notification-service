package usecase

import (
	"context"
	"time"

	"github.com/nexus-commerce/notification-service/internal/notification"
)

type SendEmailChangeConfirmation struct {
	notifier Notifier
	opener   notification.Opener
	now      func() time.Time
}

func NewSendEmailChangeConfirmation(n Notifier, o notification.Opener, now func() time.Time) *SendEmailChangeConfirmation {
	return &SendEmailChangeConfirmation{notifier: n, opener: o, now: now}
}

func (s *SendEmailChangeConfirmation) Execute(ctx context.Context, ev notification.EmailChangeRequested) error {
	msg, err := ev.Confirmation(s.opener, s.now())
	if err != nil {
		return err
	}
	return s.notifier.Notify(ctx, msg)
}
