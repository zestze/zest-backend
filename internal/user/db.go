package user

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"time"

	"github.com/zestze/zest-backend/internal/zlog"
	"github.com/zestze/zest-backend/internal/zql"
	"github.com/zestze/zest-backend/internal/ztrace"
)

var DB_FILE_NAME = "internal/user/store.db"

type Store struct {
	io.Closer
	db *sql.DB
}

func NewStore(dbName string) (Store, error) {
	db, err := zql.Sqlite3(dbName)
	if err != nil {
		return Store{}, err
	}
	return Store{
		db: db,
	}, nil
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
		WHERE username=?`, username).
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

	result, err := s.db.ExecContext(ctx,
		`INSERT INTO users
		(username, password, salt, created_at)
		VALUES 
		(?, ?, ?, ?)
	`, username, password, salt, time.Now().Unix())
	if err != nil {
		logger.Error("error persisting user", "error", err)
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil {
		logger.Error("error getting id after persisting user", "error", err)
		return 0, err
	}
	return id, nil
}

func (s Store) Reset(ctx context.Context) error {
	logger := zlog.Logger(ctx)

	_, err := s.db.Exec(`
	DROP TABLE IF EXISTS users;
	CREATE TABLE IF NOT EXISTS users (
		id         INTEGER PRIMARY KEY AUTOINCREMENT,
		username   TEXT UNIQUE,
		password   TEXT UNIQUE,
		salt       TEXT UNIQUE,
		created_at INTEGER
	);`)
	if err != nil {
		logger.Error("error running reset sql", "error", err)
		return err
	}
	return nil
}

func (s Store) Close() error {
	return s.db.Close()
}
