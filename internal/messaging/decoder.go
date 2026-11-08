package messaging

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"time"

	"github.com/nexus-commerce/notification-service/internal/notification"
)

var ErrMalformedMessage = errors.New("malformed message")

type userEnvelope struct {
	Type string         `json:"eventType"`
	Data jsontext.Value `json:"data"`
}

type orderEnvelope struct {
	Type string         `json:"event_type"`
	Data jsontext.Value `json:"data"`
}

func decodeData[T any](value []byte, typ string) (T, error) {
	var env userEnvelope
	if err := json.Unmarshal(value, &env); err != nil {
		var zero T
		return zero, fmt.Errorf("%w: decode envelope: %w", ErrMalformedMessage, err)
	}
	return decodePayload[T](env.Type, env.Data, typ)
}

func decodeOrderData[T any](value []byte, typ string) (T, error) {
	var env orderEnvelope
	if err := json.Unmarshal(value, &env); err != nil {
		var zero T
		return zero, fmt.Errorf("%w: decode envelope: %w", ErrMalformedMessage, err)
	}
	return decodePayload[T](env.Type, env.Data, typ)
}

func decodePayload[T any](got string, raw jsontext.Value, typ string) (T, error) {
	var data T
	if got != typ {
		return data, fmt.Errorf("%w: event type %q, want %q", ErrMalformedMessage, got, typ)
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return data, fmt.Errorf("%w: decode %s data: %w", ErrMalformedMessage, typ, err)
	}
	return data, nil
}

func malformedField(field string, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %s: %w", ErrMalformedMessage, field, err)
}

func malformed(err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%w: %w", ErrMalformedMessage, err)
}

type userRegistrationRequested struct {
	Email          string    `json:"email"`
	FirstName      string    `json:"firstName"`
	LastName       string    `json:"lastName"`
	EncryptedToken []byte    `json:"encryptedToken"`
	KeyID          string    `json:"keyId"`
	ExpiresAt      time.Time `json:"expiresAt"`
}

func DecodeRegistrationRequested(value []byte) (notification.RegistrationRequested, error) {
	d, err := decodeData[userRegistrationRequested](value, TypeRegistrationRequested)
	if err != nil {
		return notification.RegistrationRequested{}, err
	}
	r, recipientErr := notification.NewRecipient(d.FirstName, d.LastName, d.Email)
	token, tokenErr := notification.NewConfirmationToken(d.EncryptedToken, d.KeyID, d.ExpiresAt)
	if err := errors.Join(malformedField("email", recipientErr), malformed(tokenErr)); err != nil {
		return notification.RegistrationRequested{}, err
	}
	return notification.RegistrationRequested{Recipient: r, Token: token}, nil
}

type userRegistered struct {
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func DecodeRegistered(value []byte) (notification.Registered, error) {
	d, err := decodeData[userRegistered](value, TypeRegistered)
	if err != nil {
		return notification.Registered{}, err
	}
	r, err := notification.NewRecipient(d.FirstName, d.LastName, d.Email)
	if err != nil {
		return notification.Registered{}, malformedField("email", err)
	}
	return notification.Registered{Recipient: r}, nil
}

type userDuplicateRegistrationAttempted struct {
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func DecodeDuplicateRegistrationAttempted(value []byte) (notification.DuplicateRegistrationAttempted, error) {
	d, err := decodeData[userDuplicateRegistrationAttempted](value, TypeDuplicateRegistrationAttempted)
	if err != nil {
		return notification.DuplicateRegistrationAttempted{}, err
	}
	r, err := notification.NewRecipient(d.FirstName, d.LastName, d.Email)
	if err != nil {
		return notification.DuplicateRegistrationAttempted{}, malformedField("email", err)
	}
	return notification.DuplicateRegistrationAttempted{Recipient: r}, nil
}

type userEmailChangeRequested struct {
	NewEmail       string    `json:"newEmail"`
	FirstName      string    `json:"firstName"`
	LastName       string    `json:"lastName"`
	EncryptedToken []byte    `json:"encryptedToken"`
	KeyID          string    `json:"keyId"`
	ExpiresAt      time.Time `json:"expiresAt"`
}

func DecodeEmailChangeRequested(value []byte) (notification.EmailChangeRequested, error) {
	d, err := decodeData[userEmailChangeRequested](value, TypeEmailChangeRequested)
	if err != nil {
		return notification.EmailChangeRequested{}, err
	}
	r, recipientErr := notification.NewRecipient(d.FirstName, d.LastName, d.NewEmail)
	token, tokenErr := notification.NewConfirmationToken(d.EncryptedToken, d.KeyID, d.ExpiresAt)
	if err := errors.Join(malformedField("newEmail", recipientErr), malformed(tokenErr)); err != nil {
		return notification.EmailChangeRequested{}, err
	}
	return notification.EmailChangeRequested{Recipient: r, Token: token}, nil
}

type userEmailChangeToTakenAddressAttempted struct {
	NewEmail string `json:"newEmail"`
}

func DecodeEmailChangeToTakenAddressAttempted(value []byte) (notification.EmailChangeToTakenAddressAttempted, error) {
	d, err := decodeData[userEmailChangeToTakenAddressAttempted](value, TypeEmailChangeToTakenAddressAttempted)
	if err != nil {
		return notification.EmailChangeToTakenAddressAttempted{}, err
	}
	r, err := notification.NewRecipient("", "", d.NewEmail)
	if err != nil {
		return notification.EmailChangeToTakenAddressAttempted{}, malformedField("newEmail", err)
	}
	return notification.EmailChangeToTakenAddressAttempted{Recipient: r}, nil
}

type userEmailChanged struct {
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	OldEmail  string `json:"oldEmail"`
	NewEmail  string `json:"newEmail"`
}

func DecodeEmailChanged(value []byte) (notification.EmailChanged, error) {
	d, err := decodeData[userEmailChanged](value, TypeEmailChanged)
	if err != nil {
		return notification.EmailChanged{}, err
	}
	r, err := notification.NewRecipient(d.FirstName, d.LastName, d.OldEmail)
	if err != nil {
		return notification.EmailChanged{}, malformedField("oldEmail", err)
	}
	return notification.EmailChanged{Recipient: r, NewEmail: d.NewEmail}, nil
}

type orderConfirmed struct {
	CustomerFirstName string       `json:"customer_first_name"`
	CustomerLastName  string       `json:"customer_last_name"`
	CustomerEmail     string       `json:"customer_email"`
	OrderID           int64        `json:"order_id"`
	OrderDate         time.Time    `json:"order_date"`
	PaymentMethod     string       `json:"payment_method"`
	Items             []*orderItem `json:"items"`
	Amount            float64      `json:"amount"`
	ShippingAddress   string       `json:"shipping_address"`
	EstimatedDelivery time.Time    `json:"estimated_delivery"`
}

type orderItem struct {
	Quantity   int32   `json:"quantity"`
	Price      float64 `json:"price"`
	SKU        string  `json:"sku"`
	Name       string  `json:"name"`
	ImageURL   string  `json:"image_url"`
	TotalPrice float64 `json:"item_total_price"`
}

func DecodeOrderConfirmed(value []byte) (notification.OrderConfirmed, error) {
	d, err := decodeOrderData[orderConfirmed](value, TypeOrderConfirmed)
	if err != nil {
		return notification.OrderConfirmed{}, err
	}
	r, recipientErr := notification.NewRecipient(d.CustomerFirstName, d.CustomerLastName, d.CustomerEmail)
	paidByCard, payErr := decodePaidByCard(d.PaymentMethod)
	if err := errors.Join(malformedField("customer_email", recipientErr), payErr); err != nil {
		return notification.OrderConfirmed{}, err
	}
	items := make([]notification.OrderItem, 0, len(d.Items))
	for _, it := range d.Items {
		if it == nil {
			continue
		}
		items = append(items, notification.OrderItem{
			Quantity:   it.Quantity,
			Price:      it.Price,
			SKU:        it.SKU,
			Name:       it.Name,
			ImageURL:   it.ImageURL,
			TotalPrice: it.TotalPrice,
		})
	}
	return notification.OrderConfirmed{
		Recipient:         r,
		OrderID:           d.OrderID,
		OrderDate:         d.OrderDate,
		PaidByCard:        paidByCard,
		Items:             items,
		Total:             d.Amount,
		ShippingAddress:   d.ShippingAddress,
		EstimatedDelivery: d.EstimatedDelivery,
	}, nil
}

const (
	paymentMethodCard       = "PAYMENT_METHOD_CARD"
	paymentMethodOnDelivery = "PAYMENT_METHOD_ON_DELIVERY"
)

func decodePaidByCard(method string) (bool, error) {
	switch method {
	case paymentMethodCard:
		return true, nil
	case paymentMethodOnDelivery:
		return false, nil
	default:
		return false, fmt.Errorf("%w: unknown payment_method %q", ErrMalformedMessage, method)
	}
}
