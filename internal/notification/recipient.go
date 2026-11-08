package notification

import (
	"errors"
	"strings"
)

var ErrMissingAddress = errors.New("recipient address is required")

type Recipient struct {
	firstName string
	lastName  string
	address   string
}

func NewRecipient(firstName, lastName, address string) (Recipient, error) {
	address = strings.TrimSpace(address)
	if address == "" {
		return Recipient{}, ErrMissingAddress
	}
	return Recipient{
		firstName: strings.TrimSpace(firstName),
		lastName:  strings.TrimSpace(lastName),
		address:   address,
	}, nil
}

func (r Recipient) Address() string { return r.address }

func (r Recipient) Name() string {
	return strings.TrimSpace(r.firstName + " " + r.lastName)
}

func (r Recipient) Greeting() string {
	if r.firstName == "" {
		return "Hello,"
	}
	return "Hi " + r.firstName + ","
}
