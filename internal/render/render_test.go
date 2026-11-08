package render

import (
	"strings"
	"testing"
)

const testContact = "help@shop.example"

func TestRenderPageWithLayout(t *testing.T) {
	r, err := New(testContact)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	got, err := r.Render("welcome", struct{ Greeting, LoginURL string }{"Hi Jane,", "https://shop.example/login"})
	if err != nil {
		t.Fatalf("Render() error: %v", err)
	}

	for _, want := range []string{"<html", "Hi Jane,", "https://shop.example/login", "mailto:" + testContact} {
		if !strings.Contains(got, want) {
			t.Errorf("Render() = %q, want it to contain %q", got, want)
		}
	}
}

func TestRenderMissingFieldFails(t *testing.T) {
	r, err := New(testContact)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}

	if _, err := r.Render("welcome", struct{ Greeting string }{"Hi Jane,"}); err == nil {
		t.Error("Render(data without LoginURL) error = nil, want error")
	}
}

func TestRenderUnknownPage(t *testing.T) {
	r, err := New(testContact)
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	if _, err := r.Render("nope", nil); err == nil {
		t.Error("Render(unknown) error = nil, want error")
	}
}
