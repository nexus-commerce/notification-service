package usecase

import (
	"context"

	"github.com/nexus-commerce/notification-service/internal/notification"
)

type SendAddressTakenNotice struct {
	notifier Notifier
}

func NewSendAddressTakenNotice(n Notifier) *SendAddressTakenNotice {
	return &SendAddressTakenNotice{notifier: n}
}

func (s *SendAddressTakenNotice) Execute(ctx context.Context, ev notification.EmailChangeToTakenAddressAttempted) error {
	return s.notifier.Notify(ctx, ev.Notice())
}
