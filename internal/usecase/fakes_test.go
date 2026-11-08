package usecase_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/nexus-commerce/notification-service/internal/usecase"
)

type fakeRenderer struct {
	page string
	data any
	err  error
}

func (f *fakeRenderer) Render(page string, data any) (string, error) {
	f.page = page
	f.data = data
	if f.err != nil {
		return "", f.err
	}
	return "<html>" + page, nil
}

type fakeMailer struct {
	sent []usecase.Email
	err  error
}

func (f *fakeMailer) Send(_ context.Context, e usecase.Email) error {
	if f.err != nil {
		return f.err
	}
	f.sent = append(f.sent, e)
	return nil
}

type fakeOpener struct {
	keyID string
	plain map[string]string
}

func (f fakeOpener) Open(sealed []byte) ([]byte, error) {
	p, ok := f.plain[string(sealed)]
	if !ok {
		return nil, errors.New("cannot open sealed token")
	}
	return []byte(p), nil
}

func (f fakeOpener) KeyID() string {
	return f.keyID
}

func fixedNow() time.Time {
	return time.Date(2026, 11, 8, 12, 0, 0, 0, time.UTC)
}

const (
	testKeyID  = "key-1"
	testSealed = "sealed-1"
	testToken  = "tok en/1"
)

func newOpener() fakeOpener {
	return fakeOpener{keyID: testKeyID, plain: map[string]string{testSealed: testToken}}
}

func newLinks(t *testing.T) usecase.Links {
	t.Helper()
	l, err := usecase.NewLinks("http://front.test")
	if err != nil {
		t.Fatalf("NewLinks() error: %v", err)
	}
	return l
}
