package spotify

import (
	"context"
	"database/sql"
	"errors"
	jsoniter "github.com/json-iterator/go"
	"github.com/zestze/zest-backend/internal/zql"
	"log/slog"
	"slices"
	"time"

	"github.com/zestze/zest-backend/internal/zlog"
	"github.com/zestze/zest-backend/internal/ztrace"
)

var ErrNoArtist = errors.New("no artist provided for song")

// StoreV1 is the "V1" implementation of the spotify store.
// it is a denormalized table that stores the user's recently played songs.
type StoreV1 struct {
	db *sql.DB
	TokenStore
}

func NewStoreV1(db *sql.DB) StoreV1 {
	return StoreV1{
		db:         db,
		TokenStore: NewTokenStore(db),
	}
}

func (s StoreV1) GetAll(ctx context.Context, userID int) ([]PlayHistoryObject, error) {
	ctx, span := ztrace.Start(ctx, "SQL spotify.Get")
	defer span.End()
	logger := zlog.Logger(ctx)

	rows, err := s.db.QueryContext(ctx, `
SELECT played_at, track_blob, context_blob
FROM spotify_songs
WHERE user_id=$1`, userID)
	if err != nil {
		logger.Error("error querying for rows", "error", err)
		return nil, err
	}

	songs := make([]PlayHistoryObject, 0)
	for rows.Next() {
		var song PlayHistoryObject
		var trackBlob, contextBlob string
		if err := rows.Scan(&song.PlayedAt, &trackBlob, &contextBlob); err != nil {
			return nil, err
		}

		// can scan most of the info from the blobs!
		if err = jsoniter.UnmarshalFromString(contextBlob, &song.Context); err != nil {
			return nil, err
		}
		if err = jsoniter.UnmarshalFromString(trackBlob, &song.Track); err != nil {
			return nil, err
		}
		songs = append(songs, song)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return songs, nil
}

func (s StoreV1) PersistRecentlyPlayed(
	ctx context.Context, songs []PlayHistoryObject, userID int,
) ([]string, error) {
	ctx, span := ztrace.Start(ctx, "SQL spotify.Persist")
	defer span.End()
	logger := zlog.Logger(ctx)

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
			return nil, zql.Rollback(tx, ErrNoArtist)
		}
		// assume 0 is 'primary', in future should have denormalized table
		artist := song.Track.Artists[0]
		contextBlob, err := song.ContextBlob()
		if err != nil {
			logger.Error("error encoding context blob")
			return nil, zql.Rollback(tx, err)
		}
		trackBlob, err := song.TrackBlob()
		if err != nil {
			logger.Error("error encoding track blob")
			return nil, zql.Rollback(tx, err)
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
			return nil, zql.Rollback(tx, err)
		}
		persisted = append(persisted, trackID)
	}

	return persisted, tx.Commit()
}

func (s StoreV1) GetRecentlyPlayed(
	ctx context.Context, userID int, start, end time.Time,
) ([]NameWithTime, error) {
	ctx, span := ztrace.Start(ctx, "SQL spotify.Get")
	defer span.End()
	logger := zlog.Logger(ctx)

	rows, err := s.db.QueryContext(ctx,
		`SELECT played_at, track_name
		FROM spotify_songs
		WHERE user_id=$1 AND $2 <= played_at and played_at <= $3`,
		userID, start, end)

	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.Error("encountered internal error when scanning spotify songs", "error", err)
	}
	defer rows.Close()

	songs := make([]NameWithTime, 0)
	for rows.Next() {
		var song NameWithTime
		if err := rows.Scan(&song.Time, &song.Name); err != nil {
			return nil, err
		}
		songs = append(songs, song)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return songs, nil
}

type artistBlob struct {
	Name string `json:"name"`
}

type TrackBlob struct {
	Artists []artistBlob `json:"artists"`
}

// GetRecentlyPlayedByArtist returns a map of artist names to the number of times
// they appear in the user's recently played songs.
func (s StoreV1) GetRecentlyPlayedByArtist(
	ctx context.Context, userID int, start, end time.Time,
) ([]NameWithListens, error) {
	ctx, span := ztrace.Start(ctx, "SQL spotify.Get")
	defer span.End()
	logger := zlog.Logger(ctx)

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
		var track TrackBlob
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

	return toSlice(artistCounts), nil
}

// should be a better way of doing this...
func toSlice(m map[string]int) []NameWithListens {
	s := make([]NameWithListens, 0, len(m))
	for k, v := range m {
		s = append(s, NameWithListens{
			Name:    k,
			Listens: v,
		})
	}
	slices.SortFunc(s, func(a, b NameWithListens) int {
		return b.Listens - a.Listens
	})
	return s
}

// TODO(zeke): really need to setup multiple tables for this kind of relationship...
func (s StoreV1) GetRecentlyPlayedForArtist(
	ctx context.Context, userID int, artist string, start, end time.Time,
) ([]NameWithListens, error) {
	ctx, span := ztrace.Start(ctx, "SQL spotify.Get")
	defer span.End()
	logger := zlog.Logger(ctx)

	rows, err := s.db.QueryContext(ctx,
		`SELECT track_name, artist_name, track_blob
		FROM spotify_songs
		WHERE user_id=$1 AND $2 <= played_at and played_at <= $3`,
		userID, artist, start, end)
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		logger.Error("encountered internal error when scanning spotify songs", "error", err)
	}
	defer rows.Close()

	songCounts := make(map[string]int)
	for rows.Next() {
		var trackName, artistName, trackBlob string
		if err := rows.Scan(&trackName, &artistName, &trackBlob); err != nil {
			return nil, err
		}
		if artistName == artist {
			songCounts[trackName] += 1
			continue
		}
		var track TrackBlob
		if err = jsoniter.UnmarshalFromString(trackBlob, &track); err != nil {
			return nil, err
		}
		if slices.ContainsFunc(track.Artists[1:], func(a artistBlob) bool {
			return a.Name == artist
		}) {
			songCounts[trackName] += 1
		}
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return toSlice(songCounts), nil
}

func (s StoreV1) Reset(ctx context.Context) error {
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
