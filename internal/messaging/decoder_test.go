package messaging

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"

	"github.com/nexus-commerce/notification-service/internal/notification"
)

func userMsg(typ, data string) []byte {
	return []byte(`{"eventId":"e1","eventType":"` + typ + `","version":"1.0","data":` + data + `}`)
}

type tokenOpener struct{}

func (tokenOpener) Open(sealed []byte) ([]byte, error) {
	if string(sealed) != "sealed" {
		return nil, errors.New("wrong sealed value")
	}
	return []byte("plain"), nil
}

func (tokenOpener) KeyID() string { return "k1" }

func assertToken(t *testing.T, tok notification.ConfirmationToken) {
	t.Helper()
	expiresAt := time.Date(2026, 11, 9, 10, 0, 0, 0, time.UTC)
	if !tok.ExpiresAt().Equal(expiresAt) {
		t.Errorf("Token.ExpiresAt() = %v, want %v", tok.ExpiresAt(), expiresAt)
	}
	if got, err := tok.Reveal(tokenOpener{}, expiresAt.Add(-time.Minute)); err != nil || got.Value() != "plain" {
		t.Errorf("Token.Reveal() = %q, %v; want %q, nil", got.Value(), err, "plain")
	}
}

func TestDecodeRegistrationRequested(t *testing.T) {
	v := userMsg(TypeRegistrationRequested, `{"userId":"u1","email":"jane@example.com","firstName":"Jane","lastName":"Doe","encryptedToken":"c2VhbGVk","keyId":"k1","expiresAt":"2026-11-09T10:00:00Z"}`)
	got, err := DecodeRegistrationRequested(v)
	if err != nil {
		t.Fatal(err)
	}
	if want := mustRecipient(t, "Jane", "Doe", "jane@example.com"); got.Recipient != want {
		t.Errorf("Recipient = %+v, want %+v", got.Recipient, want)
	}
	assertToken(t, got.Token)
}

func TestDecodeRegistered(t *testing.T) {
	got, err := DecodeRegistered(userMsg(TypeRegistered, `{"userId":"u1","email":"a@b.c","firstName":"Jane","lastName":"Doe"}`))
	want := notification.Registered{Recipient: mustRecipient(t, "Jane", "Doe", "a@b.c")}
	if err != nil || got != want {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestDecodeDuplicateRegistrationAttempted(t *testing.T) {
	got, err := DecodeDuplicateRegistrationAttempted(userMsg(TypeDuplicateRegistrationAttempted, `{"userId":"u1","email":"a@b.c","firstName":"Jane","lastName":"Doe"}`))
	want := notification.DuplicateRegistrationAttempted{Recipient: mustRecipient(t, "Jane", "Doe", "a@b.c")}
	if err != nil || got != want {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestDecodeEmailChangeRequested(t *testing.T) {
	got, err := DecodeEmailChangeRequested(userMsg(TypeEmailChangeRequested, `{"userId":"u1","newEmail":"n@b.c","firstName":"Jane","lastName":"Doe","encryptedToken":"c2VhbGVk","keyId":"k1","expiresAt":"2026-11-09T10:00:00Z"}`))
	if err != nil {
		t.Fatal(err)
	}
	if want := mustRecipient(t, "Jane", "Doe", "n@b.c"); got.Recipient != want {
		t.Errorf("Recipient = %+v, want %+v", got.Recipient, want)
	}
	assertToken(t, got.Token)
}

func TestDecodeEmailChangeToTakenAddressAttempted(t *testing.T) {
	got, err := DecodeEmailChangeToTakenAddressAttempted(userMsg(TypeEmailChangeToTakenAddressAttempted, `{"userId":"u1","newEmail":"n@b.c"}`))
	want := notification.EmailChangeToTakenAddressAttempted{Recipient: mustRecipient(t, "", "", "n@b.c")}
	if err != nil || got != want {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestDecodeEmailChanged(t *testing.T) {
	got, err := DecodeEmailChanged(userMsg(TypeEmailChanged, `{"userId":"u1","firstName":"Jane","lastName":"Doe","oldEmail":"o@b.c","newEmail":"n@b.c"}`))
	want := notification.EmailChanged{Recipient: mustRecipient(t, "Jane", "Doe", "o@b.c"), NewAddress: "n@b.c"}
	if err != nil || got != want {
		t.Errorf("got %+v, %v", got, err)
	}
}

func TestDecodeEmailChangedWithoutName(t *testing.T) {
	got, err := DecodeEmailChanged(userMsg(TypeEmailChanged, `{"userId":"u1","oldEmail":"o@b.c","newEmail":"n@b.c"}`))
	if err != nil {
		t.Fatal(err)
	}
	if want := mustRecipient(t, "", "", "o@b.c"); got.Recipient != want {
		t.Errorf("Recipient = %+v, want %+v", got.Recipient, want)
	}
}

func TestDecodeOrderConfirmed(t *testing.T) {
	v := []byte(`{"event_id":"e1","event_type":"orders.confirmed","timestamp":"2026-11-08T10:00:00Z","version":"1.0","data":{
		"customer_first_name":"Jane","customer_last_name":"Doe","customer_email":"jane@example.com",
		"order_id":42,"order_date":"2026-11-08T10:00:00Z","payment_method":"PAYMENT_METHOD_CARD","payment_intent_id":"pi_1","user_id":7,
		"items":[{"quantity":2,"price":9.5,"sku":"SKU1","name":"Mug","image_url":"https://img/mug.png","item_total_price":19}],
		"amount":19,"shipping_address":"1 Main St","estimated_delivery":"2026-11-12T10:00:00Z"}}`)
	got, err := DecodeOrderConfirmed(v)
	if err != nil {
		t.Fatal(err)
	}
	want := notification.OrderConfirmed{
		Recipient:  mustRecipient(t, "Jane", "Doe", "jane@example.com"),
		OrderID:    42,
		OrderDate:  time.Date(2026, 11, 8, 10, 0, 0, 0, time.UTC),
		PaidByCard: true,
		Items: []notification.OrderItem{
			{Quantity: 2, Price: 9.5, SKU: "SKU1", Name: "Mug", ImageURL: "https://img/mug.png", TotalPrice: 19},
		},
		Total:             19,
		ShippingAddress:   "1 Main St",
		EstimatedDelivery: time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC),
	}
	if diff := cmp.Diff(want, got, cmp.AllowUnexported(notification.Recipient{})); diff != "" {
		t.Errorf("DecodeOrderConfirmed() mismatch (-want +got):\n%s", diff)
	}
}

func TestDecodeOrderConfirmedPaymentMethod(t *testing.T) {
	tests := map[string]struct {
		method      string
		wantPaid    bool
		wantInvalid bool
	}{
		"card":        {method: "PAYMENT_METHOD_CARD", wantPaid: true},
		"on delivery": {method: "PAYMENT_METHOD_ON_DELIVERY", wantPaid: false},
		"unspecified": {method: "PAYMENT_METHOD_UNSPECIFIED", wantInvalid: true},
		"missing":     {method: "", wantInvalid: true},
		"unknown":     {method: "PAYMENT_METHOD_CRYPTO", wantInvalid: true},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			v := []byte(`{"event_type":"orders.confirmed","data":{"customer_email":"jane@example.com","order_id":1,"payment_method":"` + tt.method + `"}}`)
			got, err := DecodeOrderConfirmed(v)
			if tt.wantInvalid {
				if !errors.Is(err, ErrMalformedMessage) {
					t.Errorf("err = %v, want ErrMalformedMessage", err)
				}
				return
			}
			if err != nil {
				t.Fatalf("DecodeOrderConfirmed() error: %v", err)
			}
			if got.PaidByCard != tt.wantPaid {
				t.Errorf("PaidByCard = %v, want %v", got.PaidByCard, tt.wantPaid)
			}
		})
	}
}

type decoderCase struct {
	typ    string
	decode func([]byte) error

	noRecipient string
}

// discard adapts a typed decoder to one that reports only the error.
func discard[T any](decode func([]byte) (T, error)) func([]byte) error {
	return func(b []byte) error {
		_, err := decode(b)
		return err
	}
}

func decoderCases() map[string]decoderCase {
	tok := `"encryptedToken":"c2VhbGVk","keyId":"k","expiresAt":"2026-11-09T10:00:00Z"`
	return map[string]decoderCase{
		"RegistrationRequested": {TypeRegistrationRequested, discard(DecodeRegistrationRequested), `{"userId":"u",` + tok + `}`},
		"Registered":            {TypeRegistered, discard(DecodeRegistered), `{"userId":"u"}`},
		"Duplicate":             {TypeDuplicateRegistrationAttempted, discard(DecodeDuplicateRegistrationAttempted), `{"userId":"u"}`},
		"EmailChangeRequested":  {TypeEmailChangeRequested, discard(DecodeEmailChangeRequested), `{"userId":"u",` + tok + `}`},
		"TakenAddress":          {TypeEmailChangeToTakenAddressAttempted, discard(DecodeEmailChangeToTakenAddressAttempted), `{"userId":"u"}`},
		"EmailChanged":          {TypeEmailChanged, discard(DecodeEmailChanged), `{"userId":"u","newEmail":"n@b.c"}`},
	}
}

func TestDecodersRejectMalformed(t *testing.T) {
	all := decoderCases()
	all["OrderConfirmed"] = decoderCase{typ: TypeOrderConfirmed, decode: discard(DecodeOrderConfirmed)}
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

func mustRecipient(t *testing.T, firstName, lastName, address string) notification.Recipient {
	t.Helper()
	r, err := notification.NewRecipient(firstName, lastName, address)
	if err != nil {
		t.Fatalf("NewRecipient(%q, %q, %q) error: %v", firstName, lastName, address, err)
	}
	return r
}

func TestDecodersReportMissingAddressField(t *testing.T) {
	tests := map[string]struct {
		decode func() error
		field  string
	}{
		"blank email": {
			func() error {
				_, err := DecodeRegistered(userMsg(TypeRegistered, `{"userId":"u","email":"  "}`))
				return err
			},
			"email",
		},
		"missing oldEmail": {
			func() error {
				_, err := DecodeEmailChanged(userMsg(TypeEmailChanged, `{"userId":"u","newEmail":"n@b.c"}`))
				return err
			},
			"oldEmail",
		},
		"missing customer_email": {
			func() error {
				_, err := DecodeOrderConfirmed([]byte(`{"event_type":"orders.confirmed","data":{"order_id":1,"payment_method":"PAYMENT_METHOD_CARD"}}`))
				return err
			},
			"customer_email",
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			err := tt.decode()
			if !errors.Is(err, ErrMalformedMessage) || !errors.Is(err, notification.ErrMissingAddress) {
				t.Fatalf("err = %v, want ErrMalformedMessage and ErrMissingAddress", err)
			}
			if !strings.Contains(err.Error(), tt.field) {
				t.Errorf("err = %q, want it to name %q", err, tt.field)
			}
		})
	}
}
