package messaging

const (
	HeaderEventType = "event-type"

	TopicUsers           = "users"
	TopicOrdersConfirmed = "orders.confirmed"

	TypeRegistrationRequested              = "user.registration_requested"
	TypeRegistered                         = "user.registered"
	TypeDuplicateRegistrationAttempted     = "user.duplicate_registration_attempted"
	TypeEmailChangeRequested               = "user.email_change_requested"
	TypeEmailChangeToTakenAddressAttempted = "user.email_change_to_taken_address_attempted"
	TypeEmailChanged                       = "user.email_changed"
	TypeOrderConfirmed                     = "orders.confirmed"
)

// Record is an event as read from a topic: its type, routing data and raw payload.
type Record struct {
	Type  string
	Topic string
	Key   string
	Value []byte
}
