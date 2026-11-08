package messaging

import (
	"encoding/json/jsontext"
	"encoding/json/v2"
	"errors"
	"fmt"
	"time"

	"github.com/nexus-commerce/notification-service/internal/event"
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

func require(ok bool, what string) error {
	if !ok {
		return fmt.Errorf("%w: missing %s", ErrMalformedMessage, what)
	}
	return nil
}

type userRegistrationRequested struct {
	UserID         string    `json:"userId"`
	Email          string    `json:"email"`
	FirstName      string    `json:"firstName"`
	LastName       string    `json:"lastName"`
	EncryptedToken []byte    `json:"encryptedToken"`
	KeyID          string    `json:"keyId"`
	ExpiresAt      time.Time `json:"expiresAt"`
}

func DecodeRegistrationRequested(value []byte) (event.RegistrationRequested, error) {
	d, err := decodeData[userRegistrationRequested](value, TypeRegistrationRequested)
	if err != nil {
		return event.RegistrationRequested{}, err
	}
	if err := errors.Join(
		require(d.Email != "", "email"),
		require(len(d.EncryptedToken) > 0, "encryptedToken"),
		require(!d.ExpiresAt.IsZero(), "expiresAt"),
	); err != nil {
		return event.RegistrationRequested{}, err
	}
	return event.RegistrationRequested{
		UserID:         d.UserID,
		Email:          d.Email,
		FirstName:      d.FirstName,
		LastName:       d.LastName,
		EncryptedToken: d.EncryptedToken,
		KeyID:          d.KeyID,
		ExpiresAt:      d.ExpiresAt,
	}, nil
}

type userRegistered struct {
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func DecodeRegistered(value []byte) (event.Registered, error) {
	d, err := decodeData[userRegistered](value, TypeRegistered)
	if err != nil {
		return event.Registered{}, err
	}
	if err := require(d.Email != "", "email"); err != nil {
		return event.Registered{}, err
	}
	return event.Registered{UserID: d.UserID, Email: d.Email, FirstName: d.FirstName, LastName: d.LastName}, nil
}

type userDuplicateRegistrationAttempted struct {
	UserID    string `json:"userId"`
	Email     string `json:"email"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
}

func DecodeDuplicateRegistrationAttempted(value []byte) (event.DuplicateRegistrationAttempted, error) {
	d, err := decodeData[userDuplicateRegistrationAttempted](value, TypeDuplicateRegistrationAttempted)
	if err != nil {
		return event.DuplicateRegistrationAttempted{}, err
	}
	if err := require(d.Email != "", "email"); err != nil {
		return event.DuplicateRegistrationAttempted{}, err
	}
	return event.DuplicateRegistrationAttempted{UserID: d.UserID, Email: d.Email, FirstName: d.FirstName, LastName: d.LastName}, nil
}

type userEmailChangeRequested struct {
	UserID         string    `json:"userId"`
	NewEmail       string    `json:"newEmail"`
	FirstName      string    `json:"firstName"`
	LastName       string    `json:"lastName"`
	EncryptedToken []byte    `json:"encryptedToken"`
	KeyID          string    `json:"keyId"`
	ExpiresAt      time.Time `json:"expiresAt"`
}

func DecodeEmailChangeRequested(value []byte) (event.EmailChangeRequested, error) {
	d, err := decodeData[userEmailChangeRequested](value, TypeEmailChangeRequested)
	if err != nil {
		return event.EmailChangeRequested{}, err
	}
	if err := errors.Join(
		require(d.NewEmail != "", "newEmail"),
		require(len(d.EncryptedToken) > 0, "encryptedToken"),
		require(!d.ExpiresAt.IsZero(), "expiresAt"),
	); err != nil {
		return event.EmailChangeRequested{}, err
	}
	return event.EmailChangeRequested{
		UserID:         d.UserID,
		NewEmail:       d.NewEmail,
		FirstName:      d.FirstName,
		LastName:       d.LastName,
		EncryptedToken: d.EncryptedToken,
		KeyID:          d.KeyID,
		ExpiresAt:      d.ExpiresAt,
	}, nil
}

type userEmailChangeToTakenAddressAttempted struct {
	UserID   string `json:"userId"`
	NewEmail string `json:"newEmail"`
}

func DecodeEmailChangeToTakenAddressAttempted(value []byte) (event.EmailChangeToTakenAddressAttempted, error) {
	d, err := decodeData[userEmailChangeToTakenAddressAttempted](value, TypeEmailChangeToTakenAddressAttempted)
	if err != nil {
		return event.EmailChangeToTakenAddressAttempted{}, err
	}
	if err := require(d.NewEmail != "", "newEmail"); err != nil {
		return event.EmailChangeToTakenAddressAttempted{}, err
	}
	return event.EmailChangeToTakenAddressAttempted{UserID: d.UserID, NewEmail: d.NewEmail}, nil
}

type userEmailChanged struct {
	UserID    string `json:"userId"`
	FirstName string `json:"firstName"`
	LastName  string `json:"lastName"`
	OldEmail  string `json:"oldEmail"`
	NewEmail  string `json:"newEmail"`
}

func DecodeEmailChanged(value []byte) (event.EmailChanged, error) {
	d, err := decodeData[userEmailChanged](value, TypeEmailChanged)
	if err != nil {
		return event.EmailChanged{}, err
	}
	if err := require(d.OldEmail != "", "oldEmail"); err != nil {
		return event.EmailChanged{}, err
	}
	return event.EmailChanged{
		UserID:    d.UserID,
		OldEmail:  d.OldEmail,
		NewEmail:  d.NewEmail,
		FirstName: d.FirstName,
		LastName:  d.LastName,
	}, nil
}

type orderConfirmed struct {
	CustomerFirstName string       `json:"customer_first_name"`
	CustomerLastName  string       `json:"customer_last_name"`
	CustomerEmail     string       `json:"customer_email"`
	OrderID           int64        `json:"order_id"`
	OrderDate         time.Time    `json:"order_date"`
	PaymentMethod     string       `json:"payment_method"`
	PaymentIntentID   *string      `json:"payment_intent_id"`
	UserID            int64        `json:"user_id"`
	Items             []*orderItem `json:"items"`
	Amount            float64      `json:"amount"`
	ShippingAddress   string       `json:"shipping_address"`
	EstimatedDelivery time.Time    `json:"estimated_delivery"`
}

type orderItem struct {
	Quantity       int32   `json:"quantity"`
	Price          float64 `json:"price"`
	Sku            string  `json:"sku"`
	Name           string  `json:"name"`
	ImageURL       string  `json:"image_url"`
	ItemTotalPrice float64 `json:"item_total_price"`
}

func DecodeOrderConfirmed(value []byte) (event.OrderConfirmed, error) {
	d, err := decodeOrderData[orderConfirmed](value, TypeOrderConfirmed)
	if err != nil {
		return event.OrderConfirmed{}, err
	}
	if err := require(d.CustomerEmail != "", "customer_email"); err != nil {
		return event.OrderConfirmed{}, err
	}
	items := make([]*event.OrderItem, 0, len(d.Items))
	for _, it := range d.Items {
		if it == nil {
			continue
		}
		items = append(items, &event.OrderItem{
			Quantity:       it.Quantity,
			Price:          it.Price,
			Sku:            it.Sku,
			Name:           it.Name,
			ImageURL:       it.ImageURL,
			ItemTotalPrice: it.ItemTotalPrice,
		})
	}
	return event.OrderConfirmed{
		CustomerFirstName: d.CustomerFirstName,
		CustomerLastName:  d.CustomerLastName,
		CustomerEmail:     d.CustomerEmail,
		OrderID:           d.OrderID,
		OrderDate:         d.OrderDate,
		PaymentMethod:     d.PaymentMethod,
		PaymentIntentID:   d.PaymentIntentID,
		UserID:            d.UserID,
		Items:             items,
		Amount:            d.Amount,
		ShippingAddress:   d.ShippingAddress,
		EstimatedDelivery: d.EstimatedDelivery,
	}, nil
}
