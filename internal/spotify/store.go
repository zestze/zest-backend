package spotify

import (
	"context"
	"database/sql"
	"errors"
	jsoniter "github.com/json-iterator/go"
	"log/slog"
	"time"

	"github.com/zestze/zest-backend/internal/zlog"
	"github.com/zestze/zest-backend/internal/ztrace"
)

var ErrNoArtist = errors.New("no artist provided for song")

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

func (s Store) PersistRecentlyPlayed(
	ctx context.Context, songs []PlayHistoryObject, userID int,
) ([]string, error) {
	logger := zlog.Logger(ctx)
	ctx, span := ztrace.Start(ctx, "SQL spotify.Persist")
	defer span.End()

	// TODO(zeke): at scale, might want to have "artist" and "album" tables
	// since i'm recreating a lot of data over and over. even "track" could work!
	stmt, err := s.db.PrepareContext(ctx,
		`INSERT INTO spotify_songs
		(user_id, played_at, track_id,
		track_name, track_blob, album_id, album_name,
		artist_id, artist_name, context_blob)
		VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		ON CONFLICT
			DO NOTHING
		RETURNING track_id`)
	if err != nil {
		logger.Error("error preparing statement", "error", err)
		return nil, err
	}
	defer stmt.Close()

	tx, err := s.db.Begin()
	if err != nil {
		logger.Error("error beginning transaction", "error", err)
		return nil, err
	}

	persisted := make([]string, 0, len(songs))
	for _, song := range songs {
		logger := logger.With(slog.String("track", song.Track.Name))
		logger.Info("processing track")
		if len(song.Track.Artists) == 0 {
			logger.Error("track doesn't have an artist")
			tx.Rollback()
			return nil, ErrNoArtist
		}
		// assume 0 is 'primary', in future should have denormalized table
		artist := song.Track.Artists[0]
		contextBlob, err := song.ContextBlob()
		if err != nil {
			logger.Error("error encoding context blob")
			tx.Rollback()
			return nil, err
		}
		trackBlob, err := song.TrackBlob()
		if err != nil {
			logger.Error("error encoding track blob")
			tx.Rollback()
			return nil, err
		}

		var trackID string
		err = tx.Stmt(stmt).
			QueryRowContext(ctx,
				userID, song.PlayedAt,
				song.Track.ID, song.Track.Name,
				trackBlob,
				song.Track.Album.ID, song.Track.Album.Name,
				artist.ID, artist.Name,
				contextBlob).Scan(&trackID)

		if err != nil && errors.Is(err, sql.ErrNoRows) {
			continue
		} else if err != nil {
			logger.Error("error persisting song", "error", err)
			tx.Rollback()
			return nil, err
		}
		persisted = append(persisted, trackID)
	}

	return persisted, tx.Commit()
}

type SongLite struct {
	PlayedAt   time.Time `json:"played_at"`
	TrackName  string    `json:"track_name"`
	AlbumName  string    `json:"album_name"`
	ArtistName string    `json:"artist_name"`
}

func (s Store) GetRecentlyPlayed(
	ctx context.Context, userID int, start, end time.Time,
) ([]SongLite, error) {
	logger := zlog.Logger(ctx)
	ctx, span := ztrace.Start(ctx, "SQL spotify.Get")
	defer span.End()

	rows, err := s.db.QueryContext(ctx,
		`SELECT played_at, track_name, album_name, artist_name
		FROM spotify_songs
		WHERE user_id=$1 AND $2 <= played_at and played_at <= $3`,
		userID, start, end)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.Error("encountered internal error when scanning spotify songs", "error", err)
	}
	defer rows.Close()

	songs := make([]SongLite, 0)
	for rows.Next() {
		var song SongLite
		if err := rows.Scan(&song.PlayedAt, &song.TrackName,
			&song.AlbumName, &song.ArtistName); err != nil {
			return nil, err
		}
		songs = append(songs, song)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return songs, nil
}

// GetRecentlyPlayedByArtist returns a map of artist names to the number of times
// they appear in the user's recently played songs.
func (s Store) GetRecentlyPlayedByArtist(
	ctx context.Context, userID int, start, end time.Time,
) (map[string]int, error) {
	logger := zlog.Logger(ctx)
	ctx, span := ztrace.Start(ctx, "SQL spotify.Get")
	defer span.End()

	rows, err := s.db.QueryContext(ctx,
		`SELECT artist_name, track_blob
		FROM spotify_songs
		WHERE user_id=$1 AND $2 <= played_at and played_at <= $3`,
		userID, start, end)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.Error("encountered internal error when scanning spotify songs", "error", err)
	}
	defer rows.Close()

	artistCounts := make(map[string]int)
	for rows.Next() {
		var artistName, trackBlob string
		if err := rows.Scan(&artistName, &trackBlob); err != nil {
			return nil, err
		}
		artistCounts[artistName] += 1
		var track struct {
			Artists []struct {
				Name string `json:"name"`
			} `json:"artists"`
		}
		if err = jsoniter.UnmarshalFromString(trackBlob, &track); err != nil {
			return nil, err
		}
		for i := 1; i < len(track.Artists); i++ {
			artistCounts[track.Artists[i].Name] += 1
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return artistCounts, nil
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
		);
		DROP TABLE IF EXISTS spotify_songs;
		CREATE TABLE IF NOT EXISTS spotify_songs (
			user_id      INTEGER,
			played_at    INTEGER,
			track_id     TEXT,	
			track_name   TEXT,
			track_blob   TEXT,
			album_id     TEXT,
			album_name   TEXT,
			artist_id    TEXT,
			artist_name  TEXT,
			context_blob TEXT,
			created_at   INTEGER
		);`); err != nil {
		logger.Error("error running reset sql", "error", err)
		return err
	}
	return nil
}
