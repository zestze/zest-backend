package metacritic

import (
	"context"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtract(t *testing.T) {
	f, err := os.Open("test_index.html")
	assert.NoError(t, err)

	defer f.Close()

	cards, err := Extract(context.Background(), f)
	assert.NoError(t, err)

	assert.Len(t, cards, 24)
}

type RoundTripFunc func(*http.Request) (*http.Response, error)

func (f RoundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func mockRoundTrip(t *testing.T) RoundTripFunc {
	return RoundTripFunc(
		func(req *http.Request) (*http.Response, error) {
			t.Helper()
			f, err := os.Open("test_index.html")
			assert.NoError(t, err)

			// let client close!
			//defer f.Close()
			return &http.Response{
				StatusCode: http.StatusOK,
				Body:       f,
			}, nil
		})
}

func TestFetchPosts(t *testing.T) {
	// mock client
	Client.Transport = mockRoundTrip(t)

	posts, err := FetchPosts(context.Background(), Options{
		Medium:  TV,
		MinYear: 2021,
		MaxYear: 2023,
	})
	assert.NoError(t, err)

	assert.Len(t, posts, 24)

	Client.Transport = http.DefaultTransport
}

func TestFetchPosts_Actual(t *testing.T) {
	t.Skip("skipping bc integration test")
	posts, err := FetchPosts(context.Background(), Options{
		Medium:  TV,
		MinYear: 2021,
		MaxYear: 2023,
	})
	assert.NoError(t, err)

	assert.True(t, len(posts) > 0)
}
