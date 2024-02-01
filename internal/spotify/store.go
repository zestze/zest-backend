package spotify

import (
	"context"
	"database/sql"
	"errors"

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

func (s Store) PersistToken(ctx context.Context, token AccessToken, userID int) error {
	logger := zlog.Logger(ctx)
	ctx, span := ztrace.Start(ctx, "SQL spotify.Persist")
	defer span.End()

	if _, err := s.db.ExecContext(ctx,
		`INSERT INTO spotify_tokens
		(user_id, access_token, token_type, scope, expires_at, refresh_token)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (user_id) 
			DO UPDATE SET
			access_token=excluded.access_token,
			refresh_token=excluded.refresh_token,
			expires_at=excluded.expires_at`,
		userID, token.Access, token.Type, token.Scope,
		token.ExpiresAt, token.Refresh); err != nil {
		logger.Error("error persisting spotify tokens", "error", err)
		return err
	}
	return nil
}

func (s Store) GetToken(ctx context.Context, userID int) (AccessToken, error) {
	logger := zlog.Logger(ctx)
	ctx, span := ztrace.Start(ctx, "SQL spotify.Get")
	defer span.End()

	var token AccessToken
	err := s.db.QueryRowContext(ctx,
		`SELECT access_token, token_type, scope, expires_at, refresh_token
		FROM spotify_tokens
		WHERE user_id=$1`, userID).
		Scan(&token.Access, &token.Type, &token.Scope,
			&token.ExpiresAt, &token.Refresh)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.Error("encountered internal error when scanning spotify auth", "error", err)
	}
	return token, err
}

func (s Store) Reset(ctx context.Context) error {
	logger := zlog.Logger(ctx)

	if _, err := s.db.Exec(`
		DROP TABLE IF EXISTS spotify_tokens;
		CREATE TABLE IF NOT EXISTS spotify_tokens (
			user_id       INTEGER PRIMARY KEY,
			access_token  TEXT,
			token_type    TEXT,
			scope         TEXT,
			expires_at    TEXT,
			refresh_token TEXT
		);`); err != nil {
		logger.Error("error running reset sql", "error", err)
		return err
	}
	return nil
}
