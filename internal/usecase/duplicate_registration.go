package usecase

import (
	"context"

	"github.com/nexus-commerce/notification-service/internal/notification"
)

type SendDuplicateRegistrationNotice struct {
	notifier Notifier
}

func NewSendDuplicateRegistrationNotice(n Notifier) *SendDuplicateRegistrationNotice {
	return &SendDuplicateRegistrationNotice{notifier: n}
}

func (s *SendDuplicateRegistrationNotice) Execute(ctx context.Context, ev notification.DuplicateRegistrationAttempted) error {
	return s.notifier.Notify(ctx, ev.Notice())
}
