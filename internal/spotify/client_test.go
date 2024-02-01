package spotify

import (
	"context"
	"fmt"
	"net/http"
	"testing"
	"time"

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

func TestClient_LoadAuth(t *testing.T) {
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

	after := time.Now().Add(-time.Hour)
	items, err := client.GetRecentlyPlayed(ctx, token, after)
	assert.NoError(err)
	//assert.Len(items, 3)
	//assert.True(len(items) > 0)
	fmt.Printf("first item: %+v\n", items[0])

}
