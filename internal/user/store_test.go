package user

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zestze/zest-backend/internal/zql"
)

func TestStore(t *testing.T) {
	db, err := zql.Postgres()
	assert.NoError(t, err)
	defer db.Close()

	store := NewStore(db)

	_, err = store.GetUser(context.Background(), "zeke")
	assert.NoError(t, err)

	_, err = store.PersistUser(context.Background(), "zeke", "reyna", 1)
	assert.NoError(t, err)
}
