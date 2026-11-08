package usecase

import (
	"context"

	"github.com/nexus-commerce/notification-service/internal/notification"
)

type Notifier interface {
	Notify(ctx context.Context, m notification.Message) error
}
