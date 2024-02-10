package reddit

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/zestze/zest-backend/internal/httptest"
)

func TestFetchPosts(t *testing.T) {
	client := NewClient(WithRoundTripper(mockRT(t, "mock_api_response.json")))
	posts, err := client.Fetch(context.Background(), false)
	assert.NoError(t, err)
	assert.True(t, len(posts) > 0)
}

func TestFetchPosts_Actual(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test due to running in short mode")
	}

	secrets, err := loadSecrets("../../secrets/reddit_config.json")
	assert.NoError(t, err)
	client := NewClient(WithSecrets(secrets))
	assert.NoError(t, err)

	posts, err := client.Fetch(context.Background(), false)
	assert.NoError(t, err)
	assert.True(t, len(posts) > 0)
}

// special mock bc reddit client authenticates _and_ fetches data
func mockRT(t *testing.T, responseFile string) httptest.RoundTripFunc {
	return httptest.RoundTripFunc(
		func(req *http.Request) (*http.Response, error) {
			t.Helper()
			// oauth.reddit.com indicates we are already authenticated
			if strings.Contains(req.URL.Host, "oauth") {
				bs, err := os.ReadFile(responseFile)
				assert.NoError(t, err)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBuffer(bs)),
				}, nil
			}
			// otherwise we need to authenticate
			secrets := Secrets{
				ClientId:     "id",
				ClientSecret: "secret",
				Username:     "user",
				Password:     "password",
			}
			bs, err := jsoniter.Marshal(secrets)
			assert.NoError(t, err)
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(bytes.NewBuffer(bs)),
			}, nil
		})
}
