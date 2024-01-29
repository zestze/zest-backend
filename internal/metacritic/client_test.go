package metacritic

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zestze/zest-backend/internal/httptest"
)

func TestExtract(t *testing.T) {
	f, err := os.Open("test_index.html")
	assert.NoError(t, err)

	defer f.Close()

	cards, err := Extract(context.Background(), f)
	assert.NoError(t, err)

	assert.Len(t, cards, 24)
}

func TestFetchPosts(t *testing.T) {
	// mock client
	client := NewClient(httptest.MockRTWithFile(t, "test_index.html"))

	posts, err := client.FetchPosts(context.Background(), Options{
		Medium:  TV,
		MinYear: 2021,
		MaxYear: 2023,
	})
	assert.NoError(t, err)

	assert.Len(t, posts, 24)
}

func TestFetchPosts_Actual(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test due to running in short mode")
	}

	client := NewClient(http.DefaultTransport)
	posts, err := client.FetchPosts(context.Background(), Options{
		Medium:  TV,
		MinYear: 2021,
		MaxYear: 2023,
	})
	assert.NoError(t, err)

	assert.True(t, len(posts) > 0)
}
