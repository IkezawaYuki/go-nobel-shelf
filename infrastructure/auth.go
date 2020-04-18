package infrastructure

import (
	"encoding/gob"
	"golang.org/x/oauth2"
	"google.golang.org/api/plus/v1"
	"net/http"
)

const (
	defaultSessionID        = "default"
	googleProfileSessionKey = "google_profile"
	oauthTokenSessionKey    = "oauth_token"
	oauthFlowRedirectKey    = "redirect"
)

func init() {
	gob.Register(&oauth2.Token{})
	gob.Register(&Profile{})
}

func profileFromSession(r *http.Request) *Profile {
	session, err := SessionStore.Get(r, defaultSessionID)
	if err != nil {
		return nil
	}
	tok, ok := session.Values[oauthTokenSessionKey].(*oauth2.Token)
	if !ok || !tok.Valid() {
		return nil
	}
	profile, ok := session.Values[googleProfileSessionKey].(*Profile)
	if !ok {
		return nil
	}
	return profile
}

type Profile struct {
	ID          string
	DisplayName string
	ImageURL    string
}

func stripProfile(p *plus.Person) *Profile {
	return &Profile{
		ID:          p.Id,
		DisplayName: p.DisplayName,
		ImageURL:    p.Image.Url,
	}
}
