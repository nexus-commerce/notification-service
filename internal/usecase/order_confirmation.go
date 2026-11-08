package usecase

import (
	"context"

	"github.com/nexus-commerce/notification-service/internal/notification"
)

type SendOrderConfirmation struct {
	notifier Notifier
}

func NewSendOrderConfirmation(n Notifier) *SendOrderConfirmation {
	return &SendOrderConfirmation{notifier: n}
}

func (s *SendOrderConfirmation) Execute(ctx context.Context, ev notification.OrderConfirmed) error {
	return s.notifier.Notify(ctx, ev.Confirmation())
}
