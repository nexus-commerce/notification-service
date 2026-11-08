package messaging

import (
	"errors"
	"testing"
	"time"

	"github.com/nexus-commerce/notification-service/internal/event"
)

func userMsg(typ, data string) []byte {
	return []byte(`{"eventId":"e1","eventType":"` + typ + `","version":"1.0","data":` + data + `}`)
}

func TestDecodeRegistrationRequested(t *testing.T) {
	v := userMsg(TypeRegistrationRequested, `{"userId":"u1","email":"jane@example.com","firstName":"Jane","lastName":"Doe","encryptedToken":"c2VhbGVk","keyId":"k1","expiresAt":"2026-11-09T10:00:00Z"}`)
	got, err := DecodeRegistrationRequested(v)
	if err != nil {
		t.Fatal(err)
	}
	if got.UserID != "u1" || got.Email != "jane@example.com" || got.FirstName != "Jane" || got.LastName != "Doe" ||
		string(got.EncryptedToken) != "sealed" || got.KeyID != "k1" ||
		!got.ExpiresAt.Equal(time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)) {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestDecodeRegistered(t *testing.T) {
	got, err := DecodeRegistered(userMsg(TypeRegistered, `{"userId":"u1","email":"a@b.c","firstName":"Jane","lastName":"Doe"}`))
	want := event.Registered{UserID: "u1", Email: "a@b.c", FirstName: "Jane", LastName: "Doe"}
	if err != nil || got != want {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestDecodeDuplicateRegistrationAttempted(t *testing.T) {
	got, err := DecodeDuplicateRegistrationAttempted(userMsg(TypeDuplicateRegistrationAttempted, `{"userId":"u1","email":"a@b.c","firstName":"Jane","lastName":"Doe"}`))
	want := event.DuplicateRegistrationAttempted{UserID: "u1", Email: "a@b.c", FirstName: "Jane", LastName: "Doe"}
	if err != nil || got != want {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestDecodeEmailChangeRequested(t *testing.T) {
	got, err := DecodeEmailChangeRequested(userMsg(TypeEmailChangeRequested, `{"userId":"u1","newEmail":"n@b.c","firstName":"Jane","lastName":"Doe","encryptedToken":"c2VhbGVk","keyId":"k1","expiresAt":"2026-11-09T10:00:00Z"}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.UserID != "u1" || got.NewEmail != "n@b.c" || got.FirstName != "Jane" || got.LastName != "Doe" ||
		string(got.EncryptedToken) != "sealed" || got.KeyID != "k1" || got.ExpiresAt.IsZero() {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestDecodeEmailChangeToTakenAddressAttempted(t *testing.T) {
	got, err := DecodeEmailChangeToTakenAddressAttempted(userMsg(TypeEmailChangeToTakenAddressAttempted, `{"userId":"u1","newEmail":"n@b.c"}`))
	want := event.EmailChangeToTakenAddressAttempted{UserID: "u1", NewEmail: "n@b.c"}
	if err != nil || got != want {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestDecodeEmailChanged(t *testing.T) {
	got, err := DecodeEmailChanged(userMsg(TypeEmailChanged, `{"userId":"u1","firstName":"Jane","lastName":"Doe","oldEmail":"o@b.c","newEmail":"n@b.c"}`))
	want := event.EmailChanged{UserID: "u1", OldEmail: "o@b.c", NewEmail: "n@b.c", FirstName: "Jane", LastName: "Doe"}
	if err != nil || got != want {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestDecodeEmailChangedWithoutName(t *testing.T) {
	got, err := DecodeEmailChanged(userMsg(TypeEmailChanged, `{"userId":"u1","oldEmail":"o@b.c","newEmail":"n@b.c"}`))
	if err != nil {
		t.Fatal(err)
	}
	if got.FirstName != "" || got.LastName != "" || got.OldEmail != "o@b.c" {
		t.Errorf("unexpected: %+v", got)
	}
}

func TestDecodeOrderConfirmed(t *testing.T) {
	v := []byte(`{"event_id":"e1","event_type":"orders.confirmed","timestamp":"2026-11-08T10:00:00Z","version":"1.0","data":{
		"customer_first_name":"Jane","customer_last_name":"Doe","customer_email":"jane@example.com",
		"order_id":42,"order_date":"2026-11-08T10:00:00Z","payment_method":"card","payment_intent_id":"pi_1","user_id":7,
		"items":[{"quantity":2,"price":9.5,"sku":"SKU1","name":"Mug","image_url":"https://img/mug.png","item_total_price":19}],
		"amount":19,"shipping_address":"1 Main St","estimated_delivery":"2026-11-12T10:00:00Z"}}`)
	got, err := DecodeOrderConfirmed(v)
	if err != nil {
		t.Fatal(err)
	}
	if got.CustomerEmail != "jane@example.com" || got.OrderID != 42 || got.UserID != 7 || got.Amount != 19 ||
		got.PaymentIntentID == nil || *got.PaymentIntentID != "pi_1" {
		t.Errorf("unexpected: %+v", got)
	}
	if len(got.Items) != 1 || got.Items[0].ImageURL != "https://img/mug.png" || got.Items[0].Quantity != 2 || got.Items[0].ItemTotalPrice != 19 {
		t.Errorf("unexpected items: %+v", got.Items)
	}
}

type decoderCase struct {
	typ    string
	decode func([]byte) error
	// noRecipient is valid data apart from the missing recipient.
	noRecipient string
}

func decoderCases() map[string]decoderCase {
	tok := `"encryptedToken":"c2VhbGVk","keyId":"k","expiresAt":"2026-11-09T10:00:00Z"`
	return map[string]decoderCase{
		"RegistrationRequested": {TypeRegistrationRequested, func(b []byte) error { _, err := DecodeRegistrationRequested(b); return err }, `{"userId":"u",` + tok + `}`},
		"Registered":            {TypeRegistered, func(b []byte) error { _, err := DecodeRegistered(b); return err }, `{"userId":"u"}`},
		"Duplicate":             {TypeDuplicateRegistrationAttempted, func(b []byte) error { _, err := DecodeDuplicateRegistrationAttempted(b); return err }, `{"userId":"u"}`},
		"EmailChangeRequested":  {TypeEmailChangeRequested, func(b []byte) error { _, err := DecodeEmailChangeRequested(b); return err }, `{"userId":"u",` + tok + `}`},
		"TakenAddress":          {TypeEmailChangeToTakenAddressAttempted, func(b []byte) error { _, err := DecodeEmailChangeToTakenAddressAttempted(b); return err }, `{"userId":"u"}`},
		"EmailChanged":          {TypeEmailChanged, func(b []byte) error { _, err := DecodeEmailChanged(b); return err }, `{"userId":"u","newEmail":"n@b.c"}`},
	}
}

func TestDecodersRejectMalformed(t *testing.T) {
	all := decoderCases()
	all["OrderConfirmed"] = decoderCase{typ: TypeOrderConfirmed, decode: func(b []byte) error { _, err := DecodeOrderConfirmed(b); return err }}
	for name, d := range all {
		cases := map[string][]byte{
			"not json":        []byte(`not json`),
			"other type":      []byte(`{"eventType":"other","event_type":"other","data":{}}`),
			"data not object": []byte(`{"eventType":"` + d.typ + `","event_type":"` + d.typ + `","data":"x"}`),
		}
		for cn, v := range cases {
			t.Run(name+"/"+cn, func(t *testing.T) {
				if err := d.decode(v); !errors.Is(err, ErrMalformedMessage) {
					t.Errorf("err = %v, want ErrMalformedMessage", err)
				}
			})
		}
	}
}

func TestDecodersRejectMissingRecipient(t *testing.T) {
	for name, d := range decoderCases() {
		t.Run(name, func(t *testing.T) {
			if err := d.decode(userMsg(d.typ, d.noRecipient)); !errors.Is(err, ErrMalformedMessage) {
				t.Errorf("err = %v, want ErrMalformedMessage", err)
			}
		})
	}
	t.Run("OrderConfirmed", func(t *testing.T) {
		v := []byte(`{"event_type":"orders.confirmed","data":{"order_id":1}}`)
		if _, err := DecodeOrderConfirmed(v); !errors.Is(err, ErrMalformedMessage) {
			t.Errorf("err = %v, want ErrMalformedMessage", err)
		}
	})
}

func TestDecodeConfirmationsRejectMissingExpiry(t *testing.T) {
	_, err := DecodeRegistrationRequested(userMsg(TypeRegistrationRequested, `{"userId":"u","email":"a@b.c","encryptedToken":"c2VhbGVk","keyId":"k"}`))
	if !errors.Is(err, ErrMalformedMessage) {
		t.Errorf("registration: %v", err)
	}
	_, err = DecodeEmailChangeRequested(userMsg(TypeEmailChangeRequested, `{"userId":"u","newEmail":"a@b.c","encryptedToken":"c2VhbGVk","keyId":"k"}`))
	if !errors.Is(err, ErrMalformedMessage) {
		t.Errorf("email change: %v", err)
	}
}

func TestDecodeConfirmationsRejectMissingToken(t *testing.T) {
	_, err := DecodeRegistrationRequested(userMsg(TypeRegistrationRequested, `{"userId":"u","email":"a@b.c","expiresAt":"2026-11-09T10:00:00Z"}`))
	if !errors.Is(err, ErrMalformedMessage) {
		t.Errorf("registration: %v", err)
	}
	_, err = DecodeEmailChangeRequested(userMsg(TypeEmailChangeRequested, `{"userId":"u","newEmail":"a@b.c","expiresAt":"2026-11-09T10:00:00Z"}`))
	if !errors.Is(err, ErrMalformedMessage) {
		t.Errorf("email change: %v", err)
	}
}
