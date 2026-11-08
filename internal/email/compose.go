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
	templateConfirmEmail          = "confirm-email"
	templateEmailTaken            = "email-taken"
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
	confirmEmailData struct {
		Greeting  string
		Link      string
		NewEmail  string
		ExpiresAt time.Time
	}
	emailTakenData struct {
		Email string
	}
	emailChangedData struct {
		Greeting string
		NewEmail string
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
			subject:   "Welcome to MyEcom",
			plainText: "Welcome to MyEcom! Your account is ready.",
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
		link := n.links.ConfirmEmailURL(m.Token.Value())
		return composed{
			template:  templateConfirmEmail,
			subject:   "Confirm your new email address",
			plainText: "Confirm your new email address: " + link,
			data: confirmEmailData{
				Greeting:  m.To.Greeting(),
				Link:      link,
				NewEmail:  m.To.Address(),
				ExpiresAt: m.ExpiresAt,
			},
		}, nil

	case notification.EmailTakenNotice:
		return composed{
			template:  templateEmailTaken,
			subject:   "Someone tried to use your email address",
			plainText: "Someone tried to change their account email to your address.",
			data:      emailTakenData{Email: m.To.Address()},
		}, nil

	case notification.EmailChangedNotice:
		return composed{
			template:  templateEmailChanged,
			subject:   "Your email address was changed",
			plainText: "Your account email address was changed.",
			data:      emailChangedData{Greeting: m.To.Greeting(), NewEmail: m.NewEmail},
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
