package usecase

import (
	"context"

	"github.com/nexus-commerce/notification-service/internal/notification"
)

type SendWelcome struct {
	notifier Notifier
}

func NewSendWelcome(n Notifier) *SendWelcome {
	return &SendWelcome{notifier: n}
}

func (s *SendWelcome) Execute(ctx context.Context, ev notification.Registered) error {
	return s.notifier.Notify(ctx, ev.Welcome())
}
