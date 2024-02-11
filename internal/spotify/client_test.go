package spotify

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/zestze/zest-backend/internal/zql"
)

func TestClient_MakeAuth(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test due to running in short mode")
	}
	assert := assert.New(t)
	client, err := NewClientWithSecrets(http.DefaultTransport, "../../secrets/spotify_config.json")
	assert.NoError(err)

	ctx := context.Background()

	token, err := client.GetAccessToken(ctx)
	assert.NoError(err)

	after := time.Now().Add(-time.Hour)
	_, err = client.GetRecentlyPlayed(ctx, token, after)
	assert.NoError(err)

	// TODO(zeke): make this temporary!
	db, err := zql.Sqlite3("spotify_test.db")
	assert.NoError(err)
	defer db.Close()

	store := NewStore(db)

	assert.NoError(store.Reset(ctx))

	hardcodedID := 1
	assert.NoError(store.PersistToken(ctx, token, hardcodedID))

	token2, err := store.GetToken(ctx, hardcodedID)
	assert.NoError(err)

	assert.Equal(token.Access, token2.Access)
}

func TestClient_FetchSongs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping integration test due to running in short mode")
	}
	db, err := zql.Sqlite3("spotify_test.db")
	assert := assert.New(t)
	assert.NoError(err)
	defer db.Close()

	store := NewStore(db)

	ctx := context.Background()
	hardcodedID := 1
	token, err := store.GetToken(ctx, hardcodedID)
	assert.NoError(err)

	fmt.Printf("original token: %+v\n", token)

	// now, use token for recently played request
	client, err := NewClientWithSecrets(http.DefaultTransport, "../../secrets/spotify_config.json")
	assert.NoError(err)

	if token.Expired() {
		token, err = client.RefreshAccess(ctx, token)
		assert.NoError(err)
		fmt.Printf("updated token: %+v\n", token)
		err = store.PersistToken(ctx, token, hardcodedID)
		assert.NoError(err)
	}

	after := time.Date(2024, 2, 10, 17, 0, 0, 0, time.UTC)
	items, err := client.GetRecentlyPlayed(ctx, token, after)
	assert.NoError(err)

	// write items
	f, err := os.Create("test_response.json")
	assert.NoError(err)
	defer f.Close()
	assert.True(len(items) > 0)
	assert.NoError(jsoniter.NewEncoder(f).Encode(items))
}
