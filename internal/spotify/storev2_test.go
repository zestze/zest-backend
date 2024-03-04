package spotify

import (
	"context"
	"github.com/stretchr/testify/assert"
	"github.com/zestze/zest-backend/internal/zql"
	"testing"
)

// TestTransferSpotifySongs is a temp function for moving the data from one store to another!
func TestTransferSpotifySongs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	db, err := zql.PostgresWithConfig(zql.WithHost("localhost"))
	assert.NoError(t, err)
	defer db.Close()

	storeV1 := NewStoreV1(db)
	storeV2 := NewStoreV2(db)

	ctx := context.Background()
	const hardcodedID = 1
	songs, err := storeV1.GetAll(ctx, hardcodedID)
	assert.NoError(t, err)

	ids, err := storeV2.PersistRecentlyPlayed(ctx, songs, hardcodedID)
	assert.NoError(t, err)
	assert.True(t, len(ids) > 0)
}
