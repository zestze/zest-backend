package user

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/zestze/zest-backend/internal/zql"
)

func TestStore(t *testing.T) {
	assert := assert.New(t)
	f, err := os.CreateTemp("", "user.*.db")
	assert.NoError(err)
	defer os.Remove(f.Name())

	db, err := zql.Sqlite3(f.Name())
	assert.NoError(err)
	defer db.Close()

	ctx := context.Background()
	store := NewStore(db)
	store.Reset(ctx)

	id, err := store.PersistUser(context.Background(), "zeke", "reyna", 1)
	assert.NoError(err)
	assert.Equal(int64(1), id)

	user, err := store.GetUser(context.Background(), "zeke")
	assert.NoError(err)
	assert.Equal("zeke", user.Username)
	assert.Equal(1, user.ID)

}
