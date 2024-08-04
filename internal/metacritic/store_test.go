package metacritic

import (
	"context"
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zestze/zest-backend/internal/httptest"
	"github.com/zestze/zest-backend/internal/user"
	"github.com/zestze/zest-backend/internal/zql"
)

func TestDB(t *testing.T) {
	assert := assert.New(t)

	// TODO(zeke): create postgres tables!
	// TODO(zeke): also want an easier way to spin up _just_ the db for unit tests!
	parentDB, err := zql.PostgresWithOptions(zql.WithHost("localhost"))
	//db, err := zql.PostgresWithOptions(zql.WithDatabase("integration"), zql.WithHost("localhost"))
	assert.NoError(err)
	defer parentDB.Close()
	ctx := context.Background()

	// TODO(zeke): generate random UUID with only underscores and alphanumeric
	dbName := "test_metacritic"
	_, err = parentDB.ExecContext(ctx, fmt.Sprintf("CREATE DATABASE %v;", dbName))
	assert.NoError(err)
	defer func() {
		_, err := parentDB.ExecContext(ctx, fmt.Sprintf("DROP DATABASE IF EXISTS %v;", dbName))
		if err != nil {
			fmt.Printf("error dropping database: %v\n", err)
		}
	}()

	db, err := zql.PostgresWithOptions(zql.WithHost("localhost"), zql.WithDatabase(dbName))
	assert.NoError(err)
	defer db.Close()

	schemaSQL, err := os.ReadFile("../../schema.sql")
	assert.NoError(err)

	_, err = db.ExecContext(ctx, string(schemaSQL))
	assert.NoError(err)

	store := NewStore(db)
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
