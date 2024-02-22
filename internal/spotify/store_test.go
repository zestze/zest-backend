package spotify

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/stretchr/testify/assert"
	"github.com/zestze/zest-backend/internal/zql"
)

func TestDB(t *testing.T) {
	assert := assert.New(t)

	f, err := os.CreateTemp("", "spotify.*.db")
	assert.NoError(err)
	defer os.Remove(f.Name())

	db, err := zql.Sqlite3(f.Name())
	assert.NoError(err)
	defer db.Close()
	store := NewStore(db)

	ctx := context.Background()
	assert.NoError(store.Reset(ctx))

	songs := mockFetchSongs(t, "mock_api_response.json")

	userID := 1
	persistedTracks, err := store.PersistRecentlyPlayed(ctx, songs, userID)
	assert.NoError(err)
	assert.Len(persistedTracks, 5)

	start := time.Date(2024, 2, 10, 17, 0, 0, 0, time.UTC)
	end := start.Add(time.Hour)
	loaded, err := store.GetRecentlyPlayed(ctx, userID, start, end)
	assert.NoError(err)
	assert.Len(loaded, 5)

	artistMap, err := store.GetRecentlyPlayedByArtist(ctx, userID, start, end)
	assert.NoError(err)
	fmt.Println(artistMap)
	assert.Len(artistMap, 7)
}

func mockFetchSongs(t *testing.T, fname string) []PlayHistoryObject {
	t.Helper()
	f, err := os.Open(fname)
	assert.NoError(t, err)
	defer f.Close()

	var items []PlayHistoryObject
	assert.NoError(t, jsoniter.NewDecoder(f).Decode(&items))
	assert.Len(t, items, 5)
	return items
}
