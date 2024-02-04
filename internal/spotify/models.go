package spotify

import (
	"encoding/base64"
	"os"
	"time"

	jsoniter "github.com/json-iterator/go"
)

type AccessToken struct {
	Access    string `json:"access_token"`
	Type      string `json:"token_type"`
	Scope     string `json:"scope"`
	ExpiresIn int    `json:"expires_in"`
	Refresh   string `json:"refresh_token"`
	// not set by spotify API
	ExpiresAt time.Time `json:"expires_at"`
}

func merge[T comparable](v, fallback T) T {
	var zero T
	if v == zero {
		return fallback
	}
	return v
}

func (old AccessToken) Merge(refreshed AccessToken) AccessToken {
	return AccessToken{
		Access:    merge(refreshed.Access, old.Access),
		Type:      merge(refreshed.Type, old.Type),
		Scope:     merge(refreshed.Scope, old.Scope),
		ExpiresIn: merge(refreshed.ExpiresIn, old.ExpiresIn),
		Refresh:   merge(refreshed.Refresh, old.Refresh),
		ExpiresAt: merge(refreshed.ExpiresAt, old.ExpiresAt),
	}
}

// Expired checks if the access token has expired, with a little buffer room
func (at AccessToken) Expired() bool {
	return time.Now().Add(time.Minute).After(at.ExpiresAt)
}

type ApiResponse struct {
	Href    string `json:"href"`
	Limit   int    `json:"limit"`
	Next    string `json:"next"`
	Cursors struct {
		After  string `json:"after"`
		Before string `json:"before"`
	} `json:"cursors"`
	Total int                 `json:"total"`
	Items []PlayHistoryObject `json:"items"`
}

type ExternalURLs struct {
	Spotify string `json:"spotify"`
}

type ContextObject struct {
	Type         string       `json:"type"`
	Href         string       `json:"href"`
	ExternalURLs ExternalURLs `json:"external_urls"`
	URI          string       `json:"uri"`
}

type Identifier struct {
	Href         string       `json:"href"`
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	URI          string       `json:"uri"`
	ExternalURLs ExternalURLs `json:"external_urls"`
}

// TrackObject is not exhaustive!
// just the core fields
type TrackObject struct {
	Identifier
	Album struct {
		Identifier
		Type string `json:"album_type"`
	} `json:"album"`
	Artists []struct {
		Identifier
		Genres     []string `json:"genres"`
		Popularity int      `json:"popularity"`
	} `json:"artists"`
	DurationMS int  `json:"duration_ms"`
	Explicit   bool `json:"explicit"`
	Popularity int  `json:"popularity"`
}

type PlayHistoryObject struct {
	PlayedAt time.Time     `json:"played_at"`
	Context  ContextObject `json:"context"`
	Track    TrackObject   `json:"track"`
}

type Secrets struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	AuthCode     string `json:"auth_code"`
	RedirectURI  string `json:"redirect_uri"`
}

func (s Secrets) BasicAuth() string {
	bs := []byte(s.ClientID + ":" + s.ClientSecret)
	encoded := base64.StdEncoding.EncodeToString(bs)
	return "Basic " + encoded
}

func loadSecrets(fname string) (Secrets, error) {
	f, err := os.Open(fname)
	if err != nil {
		return Secrets{}, err
	}
	defer f.Close()

	var secrets Secrets
	if err = jsoniter.NewDecoder(f).Decode(&secrets); err != nil {
		return Secrets{}, err
	}
	return secrets, nil
}
