package reddit

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zestze/zest-backend/internal/httptest"
)

func TestFetchPosts(t *testing.T) {
	client, err := NewClientWithSecrets(httptest.MockRTWithFile(t, "test_response.json"),
		"../../secrets/reddit_config.json")
	assert.NoError(t, err)

	posts, err := client.Fetch(context.Background(), false)
	assert.NoError(t, err)
	assert.True(t, len(posts) > 0)
}

func TestFetchPosts_Actual(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test due to running in short mode")
	}

	client, err := NewClientWithSecrets(httptest.MockRTWithFile(t, "test_response.json"),
		"../../secrets/reddit_config.json")
	assert.NoError(t, err)

	posts, err := client.Fetch(context.Background(), false)
	assert.NoError(t, err)
	assert.True(t, len(posts) > 0)
}
