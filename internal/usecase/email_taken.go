package usecase

import (
	"context"

	"github.com/nexus-commerce/notification-service/internal/notification"
)

type SendEmailTakenNotice struct {
	notifier Notifier
}

func NewSendEmailTakenNotice(n Notifier) *SendEmailTakenNotice {
	return &SendEmailTakenNotice{notifier: n}
}

func (s *SendEmailTakenNotice) Execute(ctx context.Context, ev notification.EmailChangeToTakenAddressAttempted) error {
	return s.notifier.Notify(ctx, ev.Notice())
}
