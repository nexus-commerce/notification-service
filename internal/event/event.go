package event

import "time"

type RegistrationRequested struct {
	UserID         string
	Email          string
	FirstName      string
	LastName       string
	EncryptedToken []byte
	KeyID          string
	ExpiresAt      time.Time
}

type Registered struct {
	UserID    string
	Email     string
	FirstName string
	LastName  string
}

type DuplicateRegistrationAttempted struct {
	UserID    string
	Email     string
	FirstName string
	LastName  string
}

type EmailChangeRequested struct {
	UserID         string
	NewEmail       string
	FirstName      string
	LastName       string
	EncryptedToken []byte
	KeyID          string
	ExpiresAt      time.Time
}

type EmailChangeToTakenAddressAttempted struct {
	UserID   string
	NewEmail string
}

type EmailChanged struct {
	UserID    string
	OldEmail  string
	NewEmail  string
	FirstName string
	LastName  string
}

type OrderConfirmed struct {
	CustomerFirstName string
	CustomerLastName  string
	CustomerEmail     string
	OrderID           int64
	OrderDate         time.Time
	PaymentMethod     string
	PaymentIntentID   *string
	UserID            int64
	Items             []*OrderItem
	Amount            float64
	ShippingAddress   string
	EstimatedDelivery time.Time
}

type OrderItem struct {
	Quantity       int32
	Price          float64
	Sku            string
	Name           string
	ImageURL       string
	ItemTotalPrice float64
}
