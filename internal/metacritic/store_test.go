package metacritic

import (
	"context"
	"database/sql"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDB(t *testing.T) {
	ctx := context.Background()
	// TODO(zeke): fix this!!
	db, err := sql.Open("TODO", "TODO")
	assert.NoError(t, err)
	store := NewStore(db)
	assert.NoError(t, err)

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
