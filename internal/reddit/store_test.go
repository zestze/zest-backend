package reddit

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zestze/zest-backend/internal/httptest"
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

	ctx := context.Background()
	assert.NoError(store.Reset(ctx))

	client, err := NewClientWithSecrets(httptest.MockRTWithFile(t, "test_response.json"),
		"../../secrets/config.json")
	assert.NoError(err)

	posts, err := client.Fetch(ctx, false)
	assert.NoError(err)
	assert.Len(posts, 6)

	userID := 1
	ids, err := store.PersistPosts(ctx, posts, userID)
	assert.NoError(err)
	assert.Len(ids, 6)

	persistedPosts, err := store.GetAllPosts(ctx, userID)
	assert.NoError(err)
	assert.Len(persistedPosts, 6)

	subreddit := "ExperiencedDevs"
	filteredPosts, err := store.GetPostsFor(ctx, subreddit, userID)
	assert.NoError(err)
	assert.Len(filteredPosts, 2)

	subreddits, err := store.GetSubreddits(ctx, userID)
	assert.NoError(err)
	assert.Len(subreddits, 5)
}
