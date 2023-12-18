package internal

import (
	"context"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB(t *testing.T) {
	DB_FILE_NAME = "test.db"
	reset()

	http.DefaultClient.Transport = mockRoundTrip(t)

	ctx := context.Background()
	options := Options{
		Medium:  TV,
		MinYear: 2021,
		MaxYear: 2023,
	}
	posts, err := FetchPosts(ctx, options)

	assert.NoError(t, err)

	ids, err := PersistPosts(ctx, posts)
	assert.NoError(t, err)
	assert.Len(t, ids, len(posts))

	persistedPosts, err := GetPosts(ctx, options)
	assert.NoError(t, err)
	assert.Len(t, persistedPosts, len(posts))

	http.DefaultClient.Transport = http.DefaultTransport
}
