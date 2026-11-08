package usecase

import (
	"context"

	"github.com/nexus-commerce/notification-service/internal/notification"
)

type SendEmailChangedNotice struct {
	notifier Notifier
}

func NewSendEmailChangedNotice(n Notifier) *SendEmailChangedNotice {
	return &SendEmailChangedNotice{notifier: n}
}

func (s *SendEmailChangedNotice) Execute(ctx context.Context, ev notification.EmailChanged) error {
	return s.notifier.Notify(ctx, ev.Notice())
}
