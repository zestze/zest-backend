package spotify

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"
)

const defaultSecretsPath = "secrets/spotify_config.json"

var ErrTokenExpired = errors.New("access token expired")

type Client struct {
	*http.Client
	secrets Secrets
}

func NewClient(roundTripper http.RoundTripper) (Client, error) {
	return NewClientWithSecrets(roundTripper, defaultSecretsPath)
}

func NewClientWithSecrets(roundTripper http.RoundTripper, secretsPath string) (Client, error) {
	secrets, err := loadSecrets(secretsPath)
	if err != nil {
		return Client{}, err
	}
	return Client{
		Client: &http.Client{
			Transport: roundTripper,
			Timeout:   60 * time.Second,
		},
		secrets: secrets,
	}, nil
}

// see: https://developer.spotify.com/documentation/web-api/tutorials/code-flow
func (c Client) GetAccessToken(ctx context.Context) (AccessToken, error) {
	form := url.Values{}
	form.Add("grant_type", "authorization_code")
	form.Add("code", c.secrets.AuthCode)
	form.Add("redirect_uri", c.secrets.RedirectURI) // not used, but required to match exactly

	return c.doRequestToken(ctx, form)
}

// see: https://developer.spotify.com/documentation/web-api/tutorials/refreshing-tokens
func (c Client) RefreshAccess(ctx context.Context, token AccessToken) (AccessToken, error) {
	form := url.Values{}
	form.Add("grant_type", "refresh_token")
	form.Add("refresh_token", token.Refresh)

	refreshed, err := c.doRequestToken(ctx, form)
	if err != nil {
		return AccessToken{}, err
	}

	// weird case where not all fields are updated.
	return token.Merge(refreshed), nil
}

func (c Client) doRequestToken(ctx context.Context, form url.Values) (AccessToken, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		"https://accounts.spotify.com/api/token", strings.NewReader(form.Encode()))
	if err != nil {
		return AccessToken{}, err
	}
	req.Header.Add("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Add("Authorization", c.secrets.BasicAuth())

	resp, err := c.Do(req)
	if err != nil {
		return AccessToken{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bs, err := io.ReadAll(resp.Body)
		if err != nil {
			return AccessToken{}, err
		}
		return AccessToken{}, fmt.Errorf("error from spotify auth, status: [%v], body: [%v]",
			resp.StatusCode, string(bs))
	}

	var token AccessToken
	if err = jsoniter.NewDecoder(resp.Body).Decode(&token); err != nil {
		return AccessToken{}, err
	}

	seconds := time.Duration(token.ExpiresIn) * time.Second
	token.ExpiresAt = time.Now().Add(seconds).UTC()
	return token, nil

}

// see: https://developer.spotify.com/documentation/web-api/reference/get-recently-played
func (c Client) GetRecentlyPlayed(
	ctx context.Context, token AccessToken, after time.Time,
) ([]PlayHistoryObject, error) {
	if token.Expired() {
		return nil, ErrTokenExpired
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet,
		"https://api.spotify.com/v1/me/player/recently-played", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Add("Authorization", "Bearer "+token.Access)

	q := req.URL.Query()
	q.Add("limit", "50")
	q.Add("after", strconv.FormatInt(after.UnixMilli(), 10))
	req.URL.RawQuery = q.Encode()

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bs, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("error from spotify api, status: [%v], body: [%v]",
			resp.StatusCode, string(bs))
	}

	var apiResponse ApiResponse
	if err := jsoniter.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, err
	}

	return apiResponse.Items, nil
}
