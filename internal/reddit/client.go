package reddit

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"os"
	"strings"
)

func Authorize(ctx context.Context, client *http.Client, secrets Secrets) (AuthResponse, error) {
	postForm := url.Values{}
	postForm.Add("grant_type", "password")
	postForm.Add("username", secrets.Username)
	postForm.Add("password", secrets.Password)

	req, err := http.NewRequestWithContext(ctx,
		http.MethodPost,
		"https://www.reddit.com/api/v1/access_token",
		strings.NewReader(postForm.Encode()),
	)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("Authorize(): error constructing request: %w", err)
	}

	req.SetBasicAuth(secrets.ClientId, secrets.ClientSecret)
	req.Header.Add("User-Agent", "simpleRedditClient/0.1 by ZestyZeke")

	resp, err := client.Do(req)
	if err != nil {
		return AuthResponse{}, fmt.Errorf("Authorize(): error making request: %w", err)
	} else if resp.StatusCode != http.StatusOK {
		return AuthResponse{}, fmt.Errorf("Authorize(): status code is not 200: %v", resp.StatusCode)
	}
	defer resp.Body.Close()

	var authResponse AuthResponse
	if err := json.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		return AuthResponse{}, fmt.Errorf("Authorize(): error decoding response: %w", err)
	}
	return authResponse, nil
}

func GetSavedPosts(ctx context.Context, client *http.Client, secrets Secrets, authData AuthResponse, lastReceived string) (ApiResponse, error) {
	fileToRequest := "/user/" + secrets.Username + "/saved"

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

	resp, err := client.Do(req)
	if err != nil {
		return ApiResponse{}, fmt.Errorf("GetSavedPosts(): error making request: %w", err)
	} else if resp.StatusCode != http.StatusOK {
		return ApiResponse{}, fmt.Errorf("GetSavedPosts(): status code is not 200: %v", resp.StatusCode)
	}
	defer resp.Body.Close()

	var apiResponse ApiResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return ApiResponse{}, fmt.Errorf("GetSavedPosts(): error decoding: %w", err)
	}

	return apiResponse, nil
}

func LoadSecrets() (Secrets, error) {
	f, err := os.Open("secrets/config.json")
	if err != nil {
		return Secrets{}, fmt.Errorf("LoadSecrets(): error opening file: %w", err)
	}
	defer f.Close()

	var secrets Secrets
	if err = json.NewDecoder(f).Decode(&secrets); err != nil {
		return Secrets{}, fmt.Errorf("LoadSecrets(): error decoding file: %w", err)
	}
	return secrets, nil
}

func PullData(ctx context.Context) ([]Child, error) {
	secrets, err := LoadSecrets()
	if err != nil {
		return nil, err
	}

	client := &http.Client{}
	authData, err := Authorize(ctx, client, secrets)
	if err != nil {
		return nil, fmt.Errorf("PullData(): error during Auth: %w", err)
	}

	slog.Info("successfully authenticated")

	savedPosts := make([]Child, 0)
	lastSeenPost := ""
	seen := map[string]bool{}

	for !seen[lastSeenPost] {
		seen[lastSeenPost] = true

		slog.Info("going to pull", slog.String("lastSeenPost", lastSeenPost))
		apiResponse, err := GetSavedPosts(ctx, client, secrets, authData, lastSeenPost)
		if err != nil {
			return nil, fmt.Errorf("PullData(): error during Get: %w", err)
		}

		for _, child := range apiResponse.Data.Children {
			savedPosts = append(savedPosts, child.Data)
		}

		lastSeenPost = apiResponse.Data.After
	}

	slog.Info("done fetching")

	return savedPosts, nil
}
