package metacritic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB(t *testing.T) {
	DB_FILE_NAME = "test.db"
	ctx := context.Background()
	store, err := NewStore(DB_FILE_NAME)
	assert.NoError(t, err)
	defer store.Close()

	store.Reset(ctx)

	client := NewClient(mockRoundTrip(t))

	options := Options{
		Medium:  TV,
		MinYear: 2021,
		MaxYear: 2023,
	}
	posts, err := client.FetchPosts(ctx, options)

	assert.NoError(t, err)

	ids, err := store.PersistPosts(ctx, posts)
	assert.NoError(t, err)
	assert.Len(t, ids, len(posts))

	persistedPosts, err := store.GetPosts(ctx, options)
	assert.NoError(t, err)
	assert.Len(t, persistedPosts, len(posts))
}
