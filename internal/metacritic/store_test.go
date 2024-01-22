package metacritic

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zestze/zest-backend/internal/zql"
)

func TestDB(t *testing.T) {
	assert := assert.New(t)
	f, err := os.CreateTemp("", "reddit.*.db")
	assert.NoError(err)
	defer os.Remove(f.Name())

	db, err := zql.Sqlite3(f.Name())
	assert.NoError(err)
	defer db.Close()
	store := NewStore(db)
	assert.NoError(err)

	ctx := context.Background()
	store.Reset(ctx)

	client := NewClient(mockRoundTrip(t))

	options := Options{
		Medium:  TV,
		MinYear: 2021,
		MaxYear: 2023,
	}
	posts, err := client.FetchPosts(ctx, options)

	assert.NoError(err)

	ids, err := store.PersistPosts(ctx, posts)
	assert.NoError(err)
	assert.Len(ids, len(posts))

	persistedPosts, err := store.GetPosts(ctx, options)
	assert.NoError(err)
	assert.Len(persistedPosts, len(posts))
}
