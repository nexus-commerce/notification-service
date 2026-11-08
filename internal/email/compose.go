package email

import (
	"fmt"
	"strconv"
	"time"

	"github.com/nexus-commerce/notification-service/internal/notification"
)

const (
	templateConfirmRegistration   = "confirm-registration"
	templateWelcome               = "welcome"
	templateDuplicateRegistration = "duplicate-registration"
	templateConfirmEmailChange    = "confirm-email-change"
	templateAddressTaken          = "address-taken"
	templateEmailChanged          = "email-changed"
	templateOrderConfirmed        = "order-confirmed"
)

type composed struct {
	template  string
	subject   string
	plainText string
	data      any
}

type (
	confirmRegistrationData struct {
		Greeting  string
		Link      string
		ExpiresAt time.Time
	}
	welcomeData struct {
		Greeting string
		LoginURL string
	}
	duplicateRegistrationData struct {
		Greeting string
		LoginURL string
	}
	confirmEmailChangeData struct {
		Greeting   string
		Link       string
		NewAddress string
		ExpiresAt  time.Time
	}
	addressTakenData struct {
		Address string
	}
	emailChangedData struct {
		Greeting   string
		NewAddress string
	}
	orderConfirmedData struct {
		Greeting          string
		OrderID           int64
		OrderDate         time.Time
		ShippingAddress   string
		EstimatedDelivery time.Time
		PaidByCard        bool
		Items             []orderItemData
		Total             float64
	}
	orderItemData struct {
		Quantity   int32
		Price      float64
		SKU        string
		Name       string
		ImageURL   string
		TotalPrice float64
	}
)

func (n *Notifier) compose(msg notification.Message) (composed, error) {
	switch m := msg.(type) {
	case notification.RegistrationConfirmation:
		link := n.links.ConfirmRegistrationURL(m.Token.Value())
		return composed{
			template:  templateConfirmRegistration,
			subject:   "Confirm your registration",
			plainText: "Confirm your registration: " + link,
			data:      confirmRegistrationData{Greeting: m.To.Greeting(), Link: link, ExpiresAt: m.ExpiresAt},
		}, nil

	case notification.Welcome:
		return composed{
			template:  templateWelcome,
			subject:   "Welcome to NexusEcommerce",
			plainText: "Welcome to NexusEcommerce! Your account is ready.",
			data:      welcomeData{Greeting: m.To.Greeting(), LoginURL: n.links.LoginURL()},
		}, nil

	case notification.DuplicateRegistrationNotice:
		return composed{
			template:  templateDuplicateRegistration,
			subject:   "Someone tried to register with your email",
			plainText: "Someone tried to register with your email address; you already have an account.",
			data:      duplicateRegistrationData{Greeting: m.To.Greeting(), LoginURL: n.links.LoginURL()},
		}, nil

	case notification.EmailChangeConfirmation:
		link := n.links.EmailChangeConfirmationURL(m.Token.Value())
		return composed{
			template:  templateConfirmEmailChange,
			subject:   "Confirm your new email address",
			plainText: "Confirm your new email address: " + link,
			data: confirmEmailChangeData{
				Greeting:   m.To.Greeting(),
				Link:       link,
				NewAddress: m.To.Address(),
				ExpiresAt:  m.ExpiresAt,
			},
		}, nil

	case notification.AddressTakenNotice:
		return composed{
			template:  templateAddressTaken,
			subject:   "Someone tried to use your email address",
			plainText: "Someone tried to change their account email to your address.",
			data:      addressTakenData{Address: m.To.Address()},
		}, nil

	case notification.EmailChangedNotice:
		return composed{
			template:  templateEmailChanged,
			subject:   "Your email address was changed",
			plainText: "Your account email address was changed.",
			data:      emailChangedData{Greeting: m.To.Greeting(), NewAddress: m.NewAddress},
		}, nil

	case notification.OrderConfirmation:
		subject := "Your Order Confirmation - #" + strconv.FormatInt(m.OrderID, 10)
		return composed{
			template:  templateOrderConfirmed,
			subject:   subject,
			plainText: "Thank you for your order. " + subject,
			data: orderConfirmedData{
				Greeting:          m.To.Greeting(),
				OrderID:           m.OrderID,
				OrderDate:         m.OrderDate,
				ShippingAddress:   m.ShippingAddress,
				EstimatedDelivery: m.EstimatedDelivery,
				PaidByCard:        m.PaidByCard,
				Items:             orderItems(m.Items),
				Total:             m.Total,
			},
		}, nil

	default:
		return composed{}, fmt.Errorf("no email for message %T", msg)
	}
}

func orderItems(items []notification.OrderItem) []orderItemData {
	out := make([]orderItemData, len(items))
	for i, it := range items {
		out[i] = orderItemData{
			Quantity:   it.Quantity,
			Price:      it.Price,
			SKU:        it.SKU,
			Name:       it.Name,
			ImageURL:   it.ImageURL,
			TotalPrice: it.TotalPrice,
		}
	}
	return out
}
