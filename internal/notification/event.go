package notification

import (
	"fmt"
	"slices"
	"time"
)

type RegistrationRequested struct {
	Recipient Recipient
	Token     ConfirmationToken
}

// Confirmation returns the registration confirmation the event warrants.
// It fails with ErrUndeliverable when the token cannot be revealed.
func (e RegistrationRequested) Confirmation(o Opener, now time.Time) (RegistrationConfirmation, error) {
	tok, err := revealForDelivery(e.Token, o, now)
	if err != nil {
		return RegistrationConfirmation{}, err
	}
	return RegistrationConfirmation{To: e.Recipient, Token: tok, ExpiresAt: e.Token.ExpiresAt()}, nil
}

type Registered struct {
	Recipient Recipient
}

func (e Registered) Welcome() Welcome {
	return Welcome{To: e.Recipient}
}

type DuplicateRegistrationAttempted struct {
	Recipient Recipient
}

func (e DuplicateRegistrationAttempted) Notice() DuplicateRegistrationNotice {
	return DuplicateRegistrationNotice{To: e.Recipient}
}

type EmailChangeRequested struct {
	Recipient Recipient
	Token     ConfirmationToken
}

// Confirmation returns the email change confirmation the event warrants.
// It fails with ErrUndeliverable when the token cannot be revealed.
func (e EmailChangeRequested) Confirmation(o Opener, now time.Time) (EmailChangeConfirmation, error) {
	tok, err := revealForDelivery(e.Token, o, now)
	if err != nil {
		return EmailChangeConfirmation{}, err
	}
	return EmailChangeConfirmation{To: e.Recipient, Token: tok, ExpiresAt: e.Token.ExpiresAt()}, nil
}

type EmailChangeToTakenAddressAttempted struct {
	Recipient Recipient
}

func (e EmailChangeToTakenAddressAttempted) Notice() AddressTakenNotice {
	return AddressTakenNotice{To: e.Recipient}
}

type EmailChanged struct {
	Recipient  Recipient
	NewAddress string
}

func (e EmailChanged) Notice() EmailChangedNotice {
	return EmailChangedNotice{To: e.Recipient, NewAddress: e.NewAddress}
}

type OrderConfirmed struct {
	Recipient         Recipient
	OrderID           int64
	OrderDate         time.Time
	PaidByCard        bool
	Items             []OrderItem
	Total             float64
	ShippingAddress   string
	EstimatedDelivery time.Time
}

func (e OrderConfirmed) Confirmation() OrderConfirmation {
	return OrderConfirmation{
		To:                e.Recipient,
		OrderID:           e.OrderID,
		OrderDate:         e.OrderDate,
		PaidByCard:        e.PaidByCard,
		Items:             slices.Clone(e.Items),
		Total:             e.Total,
		ShippingAddress:   e.ShippingAddress,
		EstimatedDelivery: e.EstimatedDelivery,
	}
}

type OrderItem struct {
	Quantity   int32
	Price      float64
	SKU        string
	Name       string
	ImageURL   string
	TotalPrice float64
}

// revealForDelivery reveals a confirmation token. A token that cannot be
// revealed makes the confirmation undeliverable: retrying will not help.
func revealForDelivery(t ConfirmationToken, o Opener, now time.Time) (RevealedToken, error) {
	tok, err := t.Reveal(o, now)
	if err != nil {
		return RevealedToken{}, fmt.Errorf("%w: %w", ErrUndeliverable, err)
	}
	return tok, nil
}
