package spotify

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"github.com/zestze/zest-backend/internal/zlog"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type AccessToken struct {
	Access    string `json:"access_token"`
	Type      string `json:"token_type"`
	Scope     string `json:"scope"`
	ExpiresIn int    `json:"expires_in"`
	Refresh   string `json:"refresh_token"`
	// not set by spotify API
	ExpiresAt time.Time `json:"expires_at"`
}

func merge[T comparable](v, fallback T) T {
	var zero T
	if v == zero {
		return fallback
	}
	return v
}

func (old AccessToken) Merge(refreshed AccessToken) AccessToken {
	return AccessToken{
		Access:    merge(refreshed.Access, old.Access),
		Type:      merge(refreshed.Type, old.Type),
		Scope:     merge(refreshed.Scope, old.Scope),
		ExpiresIn: merge(refreshed.ExpiresIn, old.ExpiresIn),
		Refresh:   merge(refreshed.Refresh, old.Refresh),
		ExpiresAt: merge(refreshed.ExpiresAt, old.ExpiresAt),
	}
}

// Expired checks if the access token has expired, with a little buffer room
func (at AccessToken) Expired() bool {
	return time.Now().Add(time.Minute).After(at.ExpiresAt)
}

type TokenStore struct {
	db *sql.DB
}

func NewTokenStore(db *sql.DB) TokenStore {
	return TokenStore{
		db: db,
	}
}

var spanOpts = []tracer.StartSpanOption{
	tracer.SpanType("db"),
	tracer.ResourceName("sql"),
}

func (s TokenStore) PersistToken(ctx context.Context, token AccessToken, userID int) error {
	span, ctx := tracer.StartSpanFromContext(ctx, "spotify.Persist", spanOpts...)
	defer span.Finish()
	logger := zlog.Logger(ctx)

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

func (s TokenStore) GetToken(ctx context.Context, userID int) (AccessToken, error) {
	span, ctx := tracer.StartSpanFromContext(ctx, "spotify.Get", spanOpts...)
	defer span.Finish()
	logger := zlog.Logger(ctx)

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
