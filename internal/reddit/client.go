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
	"github.com/samber/lo"

	"github.com/zestze/zest-backend/internal/zlog"
)

var Client = &http.Client{
	Timeout: 60 * time.Second,
}

func Fetch(ctx context.Context, grabAll bool) ([]Post, error) {
	logger := zlog.Logger(ctx)
	secrets, err := loadSecrets()
	if err != nil {
		return nil, err
	}

	authData, err := authorize(ctx, Client, secrets)
	if err != nil {
		return nil, fmt.Errorf("Fetch(): error during Auth: %w", err)
	}
	logger.Info("successfully authenticated")

	logger.Info("going to pull")
	apiResponse, err := getSavedPosts(ctx, Client, secrets, authData, "")
	if err != nil {
		return nil, fmt.Errorf("Fetch(): error during Get: %w", err)
	}

	toPosts := func() []Post {
		return lo.Map(apiResponse.Data.Children, func(child struct{ Data Post }, _ int) Post {
			return child.Data
		})
	}

	savedPosts := toPosts()

	seen := map[string]bool{}
	lastSeenPost := apiResponse.Data.After

	for grabAll && !seen[lastSeenPost] {
		seen[lastSeenPost] = true

		logger.Info("going to pull", slog.String("lastSeenPost", lastSeenPost))
		apiResponse, err := getSavedPosts(ctx, Client, secrets, authData, lastSeenPost)
		if err != nil {
			return nil, fmt.Errorf("Fetch(): error during Get: %w", err)
		}

		savedPosts = append(savedPosts, toPosts()...)

		lastSeenPost = apiResponse.Data.After
	}

	logger.Info("done fetching")

	return savedPosts, nil
}

func authorize(ctx context.Context, client *http.Client, secrets Secrets) (AuthResponse, error) {
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
	if err := jsoniter.NewDecoder(resp.Body).Decode(&authResponse); err != nil {
		return AuthResponse{}, fmt.Errorf("Authorize(): error decoding response: %w", err)
	}
	return authResponse, nil
}

func getSavedPosts(ctx context.Context, client *http.Client, secrets Secrets, authData AuthResponse, lastReceived string) (ApiResponse, error) {
	fileToRequest := "/user/" + secrets.Username + "/saved?raw_json=1"

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
	if err := jsoniter.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return ApiResponse{}, fmt.Errorf("GetSavedPosts(): error decoding: %w", err)
	}

	return apiResponse, nil
}

func loadSecrets() (Secrets, error) {
	f, err := os.Open("secrets/config.json")
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
