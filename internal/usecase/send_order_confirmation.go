package usecase

import (
	"context"
	"strconv"

	"github.com/nexus-commerce/notification-service/internal/event"
)

type SendOrderConfirmation struct {
	renderer Renderer
	mailer   Mailer
}

func NewSendOrderConfirmation(r Renderer, m Mailer) *SendOrderConfirmation {
	return &SendOrderConfirmation{renderer: r, mailer: m}
}

func (s *SendOrderConfirmation) Execute(ctx context.Context, ev event.OrderConfirmed) error {
	subject := "Your Order Confirmation - #" + strconv.FormatInt(ev.OrderID, 10)
	return send(ctx, s.renderer, s.mailer, PageOrderConfirmed, ev, Email{
		ToName:    fullName(ev.CustomerFirstName, ev.CustomerLastName),
		ToAddress: ev.CustomerEmail,
		Subject:   subject,
		PlainText: "Thank you for your order. " + subject,
	})
}
