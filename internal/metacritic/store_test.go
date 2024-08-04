package metacritic

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zestze/zest-backend/internal/httptest"
	"github.com/zestze/zest-backend/internal/user"
	"github.com/zestze/zest-backend/internal/zql"
)

func TestDB(t *testing.T) {
	assert := assert.New(t)

	// TODO(zeke): create postgres tables!
	db, err := zql.PostgresWithOptions(zql.WithDatabase("integration"), zql.WithHost("localhost"))
	assert.NoError(err)
	defer db.Close()

	// TODO(zeke): run SQL in schema to migrate! or have that done beforehand...

	store := NewStore(db)
	ctx := context.Background()
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
	// TODO(zeke): why isn't this working?

	persistedPosts, err := store.GetPosts(ctx, options)
	assert.NoError(err)
	assert.Len(persistedPosts, len(posts))

	userStore := user.NewStore(db)
	userID, err := userStore.PersistUser(ctx, "zeke", "reyna", 1)
	assert.NoError(err)
	err = store.SavePostsForUser(ctx, ids[:4], user.ID(userID), SAVED)
	assert.NoError(err)

	// TODO(zeke): now test fetching saved posts for the user!
	savedPosts, err := store.GetSavedPostsForUser(ctx, user.ID(userID))
	assert.NoError(err)
	assert.Len(savedPosts, 4)
	for i, sp := range savedPosts {
		assert.Equal(SAVED, sp.Action)
		assert.Equal(posts[i].Title, sp.Title)
	}
}
