package metacritic

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zestze/zest-backend/internal/httptest"
	"github.com/zestze/zest-backend/internal/user"
	"github.com/zestze/zest-backend/internal/zql"
)

func TestDB(t *testing.T) {
	assert := assert.New(t)
	f, err := os.CreateTemp("", "metacritic.*.db")
	assert.NoError(err)
	defer os.Remove(f.Name())

	db, err := zql.Sqlite3(f.Name())
	assert.NoError(err)
	defer db.Close()
	store := NewStore(db)

	ctx := context.Background()
	// TODO(zeke): need a better way of applying migrations in unit tests.
	assert.NoError(store.Reset(ctx))

	client := NewClient(httptest.MockRTWithFile(t, "test_index.html"))

	options := Options{
		Medium:  TV,
		MinYear: 2021,
		MaxYear: 2023,
	}
	posts, err := client.FetchPosts(ctx, options)
	assert.Len(posts, 24)
	assert.NoError(err)

	ids, err := store.PersistPosts(ctx, posts)
	assert.NoError(err)
	assert.Len(ids, len(posts))

	persistedPosts, err := store.GetPosts(ctx, options)
	assert.NoError(err)
	assert.Len(persistedPosts, len(posts))

	userStore := user.NewStore(db)
	userID, err := userStore.PersistUser(ctx, "zeke", "reyna", 1)
	assert.NoError(err)
	err = store.SavePostsForUser(ctx, ids[:4], user.ID(userID), SAVED)
	assert.NoError(err)

	// TODO(zeke): now test fetching saved posts for the user!
}
