package reddit

import (
	"context"
	"os"
	"testing"

	jsoniter "github.com/json-iterator/go"
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

	ctx := context.Background()
	assert.NoError(store.Reset(ctx))

	posts := mockFetchPosts(t, "mock_api_response.json")

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

func mockFetchPosts(t *testing.T, fname string) []Post {
	t.Helper()
	f, err := os.Open(fname)
	assert.NoError(t, err)
	defer f.Close()

	var apiResponse ApiResponse
	assert.NoError(t, jsoniter.NewDecoder(f).Decode(&apiResponse))

	return apiResponse.Posts()
}
