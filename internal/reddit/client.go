package reddit

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	jsoniter "github.com/json-iterator/go"

	"github.com/zestze/zest-backend/internal/zlog"
)

const defaultSecretsPath = "secrets/reddit_config.json"

type Client struct {
	Client  *http.Client
	secrets Secrets
}

func NewClient(options ...func(*Client)) *Client {
	client := &Client{
		Client: &http.Client{
			Timeout:   60 * time.Second,
			Transport: http.DefaultTransport,
		},
	}

	for _, o := range options {
		o(client)
	}

	return client
}

func WithRoundTripper(rt http.RoundTripper) func(*Client) {
	return func(c *Client) {
		c.Client.Transport = rt
	}
}

func WithSecrets(secrets Secrets) func(*Client) {
	return func(c *Client) {
		c.secrets = secrets
	}
}

func WithTimeout(t time.Duration) func(*Client) {
	return func(c *Client) {
		c.Client.Timeout = t
	}
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

func (c Client) Fetch(ctx context.Context, grabAll bool) ([]Post, error) {
	logger := zlog.Logger(ctx)

	authData, err := c.authorize(ctx)
	if err != nil {
		return nil, fmt.Errorf("Fetch(): error during Auth: %w", err)
	}
	logger.Info("successfully authenticated")

	logger.Info("going to pull")
	apiResponse, err := c.getSavedPosts(ctx, authData, "")
	if err != nil {
		return nil, fmt.Errorf("Fetch(): error during Get: %w", err)
	}

	savedPosts := apiResponse.Posts()

	seen := map[string]bool{}
	lastSeenPost := apiResponse.Data.After

	for grabAll && !seen[lastSeenPost] {
		seen[lastSeenPost] = true

		logger.Info("going to pull", slog.String("lastSeenPost", lastSeenPost))
		apiResponse, err := c.getSavedPosts(ctx, authData, lastSeenPost)
		if err != nil {
			return nil, fmt.Errorf("Fetch(): error during Get: %w", err)
		}

		savedPosts = append(savedPosts, apiResponse.Posts()...)

		lastSeenPost = apiResponse.Data.After
	}

	logger.Info("done fetching")

	return savedPosts, nil
}

func (c Client) authorize(ctx context.Context) (AuthResponse, error) {
	postForm := url.Values{}
	postForm.Add("grant_type", "password")
	postForm.Add("username", c.secrets.Username)
	postForm.Add("password", c.secrets.Password)

	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost,
		"https://www.reddit.com/api/v1/access_token",
		strings.NewReader(postForm.Encode()),
	)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("Authorize(): error constructing request: %w", err)
	}

	req.SetBasicAuth(c.secrets.ClientId, c.secrets.ClientSecret)
	req.Header.Add("User-Agent", "simpleRedditClient/0.1 by ZestyZeke")

	resp, err := c.Client.Do(req)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("Authorize(): error making request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return AuthResponse{}, fmt.Errorf("Authorize(): status code is not 200: %v", resp.StatusCode)
	}

	var authResponse AuthResponse
	if err := jsoniter.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		return AuthResponse{}, fmt.Errorf("Authorize(): error decoding response: %w", err)
	}
	return authResponse, nil
}

func (c Client) getSavedPosts(ctx context.Context, authData AuthResponse, lastReceived string) (ApiResponse, error) {
	fileToRequest := "/user/" + c.secrets.Username + "/saved?raw_json=1"

	req, err := http.NewRequestWithContext(ctx,
		http.MethodGet,
		"https://oauth.reddit.com"+fileToRequest,
		nil)
	if err != nil {
		return ApiResponse{}, fmt.Errorf("GetSavedPosts(): error constructing request: %w", err)
	}
	req.Header.Add("User-Agent", "simpleRedditClient/0.1 by ZestyZeke")
	req.Header.Add("Authorization", authData.TokenType+" "+authData.AccessToken)

	if lastReceived != "" {
		q := req.URL.Query()
		q.Add("after", lastReceived)
		req.URL.RawQuery = q.Encode()
	}

	resp, err := c.Client.Do(req)
	if err != nil {
		return ApiResponse{}, fmt.Errorf("GetSavedPosts(): error making request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return ApiResponse{}, fmt.Errorf("GetSavedPosts(): status code is not 200: %v", resp.StatusCode)
	}

	var apiResponse ApiResponse
	if err := jsoniter.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return ApiResponse{}, fmt.Errorf("GetSavedPosts(): error decoding: %w", err)
	}

	return apiResponse, nil
}

func loadSecrets(fname string) (Secrets, error) {
	f, err := os.Open(fname)
	if err != nil {
		return Secrets{}, fmt.Errorf("LoadSecrets(): error opening file: %w", err)
	}
	defer f.Close()

	var secrets Secrets
	if err = jsoniter.NewDecoder(f).Decode(&secrets); err != nil {
		return Secrets{}, fmt.Errorf("LoadSecrets(): error decoding file: %w", err)
	}
	return secrets, nil
}
