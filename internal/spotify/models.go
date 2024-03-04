package spotify

import (
	"encoding/base64"
	"os"
	"time"

	jsoniter "github.com/json-iterator/go"
)

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
//
// in postgreSQL to get the nested genres field, need to do:
// `SELECT track_blob->'artists'->0->'genres' FROM spotify_songs;`
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

func (obj PlayHistoryObject) ContextBlob() ([]byte, error) {
	return jsoniter.Marshal(obj.Context)
}

func (obj PlayHistoryObject) TrackBlob() ([]byte, error) {
	return jsoniter.Marshal(obj.Track)
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
