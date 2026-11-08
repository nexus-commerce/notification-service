package email

import (
	"fmt"
	"net/url"
	"strings"
)

type Links struct {
	base *url.URL
}

func NewLinks(frontendURL string) (Links, error) {
	base, err := url.Parse(frontendURL)
	if err != nil {
		return Links{}, fmt.Errorf("parse frontend url: %w", err)
	}
	if base.Scheme == "" || base.Host == "" {
		return Links{}, fmt.Errorf("frontend url %q must be absolute", frontendURL)
	}
	base.Path = strings.TrimRight(base.Path, "/")
	base.RawPath = ""
	base.RawQuery = ""
	base.Fragment = ""
	return Links{base: base}, nil
}

func (l Links) ConfirmRegistrationURL(token string) string {
	return l.withToken("/confirm-registration", token)
}

func (l Links) EmailChangeConfirmationURL(token string) string {
	return l.withToken("/confirm-email", token)
}

func (l Links) LoginURL() string {
	return l.at("/login").String()
}

func (l Links) withToken(path, token string) string {
	u := l.at(path)
	u.RawQuery = url.Values{"token": {token}}.Encode()
	return u.String()
}

func (l Links) at(path string) *url.URL {
	u := l.base.Clone()
	u.Path += path
	return u
}
