package user

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"

	"github.com/zestze/zest-backend/internal/zlog"
	"github.com/zestze/zest-backend/internal/ztrace"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) Store {
	return Store{
		db: db,
	}
}

type User struct {
	ID       int
	Username string
	Password string
}

// can also get user by ID!
func (s Store) GetUser(ctx context.Context, username string) (User, error) {
	logger := zlog.Logger(ctx)
	ctx, span := ztrace.Start(ctx, "SQL user.Get")
	defer span.End()

	user := User{
		Username: username,
	}
	err := s.db.QueryRowContext(ctx,
		`SELECT id, password
		FROM users
		WHERE username=$1`, username).
		Scan(&user.ID, &user.Password)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.Error("encountered internal error when scanning for password", "error", err)
	}
	return user, err
}

func (s Store) PersistUser(ctx context.Context, username, password string, salt int) (int64, error) {
	logger := zlog.Logger(ctx).With(slog.String("username", username))
	ctx, span := ztrace.Start(ctx, "SQL user.Persist")
	defer span.End()

	var id int64
	if err := s.db.QueryRowContext(ctx,
		`INSERT INTO users
		(username, password, salt) 
		VALUES ($1, $2, $3)
		ON CONFLICT DO NOTHING
		RETURNING id`,
		username, password, salt).Scan(&id); err != nil {
		logger.Error("error persisting user", "error", err)
		return 0, err
	}
	return id, nil
}

func (s Store) Reset(ctx context.Context) error {
	logger := zlog.Logger(ctx)

	if _, err := s.db.Exec(`
	DROP TABLE IF EXISTS users;
	CREATE TABLE IF NOT EXISTS users (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		username   TEXT UNIQUE,
		password   TEXT UNIQUE,
		salt       INTEGER UNIQUE,
		created_at INTEGER
	);`); err != nil {
		logger.Error("error running reset sql", "error", err)
		return err
	}
	return nil
}
