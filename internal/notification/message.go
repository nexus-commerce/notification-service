package notification

import "time"

type Message interface {
	Recipient() Recipient
	isMessage()
}

type Welcome struct {
	To Recipient
}

func (m Welcome) Recipient() Recipient { return m.To }

func (Welcome) isMessage() {}

type RegistrationConfirmation struct {
	To        Recipient
	Token     RevealedToken
	ExpiresAt time.Time
}

func (m RegistrationConfirmation) Recipient() Recipient { return m.To }

func (RegistrationConfirmation) isMessage() {}

type DuplicateRegistrationNotice struct {
	To Recipient
}

func (m DuplicateRegistrationNotice) Recipient() Recipient { return m.To }

func (DuplicateRegistrationNotice) isMessage() {}

type EmailChangeConfirmation struct {
	To        Recipient
	Token     RevealedToken
	ExpiresAt time.Time
}

func (m EmailChangeConfirmation) Recipient() Recipient { return m.To }

func (EmailChangeConfirmation) isMessage() {}

type EmailTakenNotice struct {
	To Recipient
}

func (m EmailTakenNotice) Recipient() Recipient { return m.To }

func (EmailTakenNotice) isMessage() {}

type EmailChangedNotice struct {
	To       Recipient
	NewEmail string
}

func (m EmailChangedNotice) Recipient() Recipient { return m.To }

func (EmailChangedNotice) isMessage() {}

type OrderConfirmation struct {
	To                Recipient
	OrderID           int64
	OrderDate         time.Time
	PaidByCard        bool
	Items             []OrderItem
	Total             float64
	ShippingAddress   string
	EstimatedDelivery time.Time
}

func (m OrderConfirmation) Recipient() Recipient { return m.To }

func (OrderConfirmation) isMessage() {}
