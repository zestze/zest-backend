package spotify

import (
	"context"
	"database/sql"
	"fmt"
	"github.com/zestze/zest-backend/internal/zlog"
	"github.com/zestze/zest-backend/internal/zql"
	"github.com/zestze/zest-backend/internal/ztrace"
	"log/slog"
	"time"
)

type StoreV2 struct {
	db *sql.DB
	TokenStore
}

func NewStoreV2(db *sql.DB) StoreV2 {
	return StoreV2{
		db:         db,
		TokenStore: NewTokenStore(db),
	}
}

func (s StoreV2) PersistRecentlyPlayed(
	ctx context.Context, songs []PlayHistoryObject, userID int,
) ([]string, error) {
	ctx, span := ztrace.Start(ctx, "SQL spotify.Persist")
	defer span.End()
	logger := zlog.Logger(ctx)

	// for each item, create all necessary rows
	tx, err := s.db.Begin()
	if err != nil {
		logger.Error("error beginning transaction", "error", err)
		return nil, zql.Rollback(tx, err)
	}

	persisted := make([]string, 0)
	for _, song := range songs {
		logger := logger.With(slog.String("track", song.Track.Name))
		trackID, err := persistSong(ctx, tx, song, userID)
		if err != nil {
			logger.Error("error persisting song", "error", err)
			return nil, zql.Rollback(tx, err)
		}
		persisted = append(persisted, trackID)
	}

	return persisted, tx.Commit()
}

type NameWithTime struct {
	Name string    `json:"name"`
	Time time.Time `json:"time"`
}

func (s StoreV2) GetRecentlyPlayed(
	ctx context.Context, userID int, start, end time.Time,
) ([]NameWithTime, error) {
	ctx, span := ztrace.Start(ctx, "SQL spotify.Get")
	defer span.End()
	logger := zlog.Logger(ctx)

	rows, err := s.db.QueryContext(ctx, `
SELECT played_at, spotify_tracks.name
FROM spotify_played_tracks
JOIN spotify_tracks on spotify_tracks.id = spotify_played_tracks.track_id
WHERE user_id=$1 
	AND spotify_played_tracks.played_at BETWEEN $2 AND $3`,
		userID, start, end)
	if err != nil {
		logger.Error("error querying for rows", "error", err)
		return nil, err
	}

	songs := make([]NameWithTime, 0)
	for rows.Next() {
		var s NameWithTime
		if err = rows.Scan(&s.Time, &s.Name); err != nil {
			return nil, err
		}
		songs = append(songs, s)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return songs, nil
}

func (s StoreV2) GetRecentlyPlayedByArtist(
	ctx context.Context, userID int, start, end time.Time,
) ([]NameWithListens, error) {
	ctx, span := ztrace.Start(ctx, "SQL spotify.Get")
	defer span.End()
	logger := zlog.Logger(ctx)

	rows, err := s.db.QueryContext(ctx, `
SELECT spotify_artists.name as artist_name, COUNT(spotify_played_tracks.played_at) as num_listens 
FROM spotify_played_tracks
JOIN spotify_tracks on spotify_tracks.id = spotify_played_tracks.track_id
JOIN spotify_credits on spotify_credits.track_id = spotify_tracks.id
JOIN spotify_artists on spotify_artists.id = spotify_credits.artist_id
WHERE spotify_played_tracks.user_id = $1 
	AND spotify_played_tracks.played_at BETWEEN $2 AND $3
GROUP BY artist_name
ORDER BY num_listens DESC`,
		userID, start, end)
	if err != nil {
		logger.Error("error querying for rows", "error", err)
		return nil, err
	}

	artists := make([]NameWithListens, 0)
	for rows.Next() {
		var a NameWithListens
		if err = rows.Scan(&a.Name, &a.Listens); err != nil {
			return nil, err
		}
		artists = append(artists, a)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return artists, nil
}

type NameWithListens struct {
	Name    string `json:"name"`
	Listens int    `json:"listens"`
}

func (s StoreV2) GetRecentlyPlayedForArtist(
	ctx context.Context, userID int, artist string, start, end time.Time,
) ([]NameWithListens, error) {
	ctx, span := ztrace.Start(ctx, "SQL spotify.Get")
	defer span.End()
	logger := zlog.Logger(ctx)

	rows, err := s.db.QueryContext(ctx, `
SELECT spotify_played_tracks.name as track_name, COUNT(spotify_played_tracks.played_at) as num_listens 
FROM spotify_played_tracks
JOIN spotify_tracks on spotify_tracks.id = spotify_played_tracks.track_id
JOIN spotify_credits on spotify_credits.track_id = spotify_tracks.id
JOIN spotify_artists on spotify_artists.id = spotify_credits.artist_id
WHERE spotify_played_tracks.user_id = $1 
	AND spotify_artists.name = $2
	AND spotify_played_tracks.played_at BETWEEN $3 AND $4
GROUP BY track_name
ORDER BY num_listens DESC`,
		userID, artist, start, end)
	if err != nil {
		logger.Error("error querying for rows", "error", err)
		return nil, err
	}

	songs := make([]NameWithListens, 0)
	for rows.Next() {
		var s NameWithListens
		if err = rows.Scan(&s.Name, &s.Listens); err != nil {
			return nil, err
		}
		songs = append(songs, s)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return songs, nil
}

// persistSong persists a played track to our database, along with all other rows that are necessary
func persistSong(ctx context.Context, tx *sql.Tx, song PlayHistoryObject, userID int) (string, error) {

	// first, make sure album exists
	album := song.Track.Album
	_, err := tx.ExecContext(ctx, `
INSERT INTO spotify_albums
(id, name, href, uri, external_url, type)
VALUES
($1, $2, $3, $4, $5, $6)
ON CONFLICT 
	DO NOTHING`,
		album.ID, album.Name, album.Href, album.URI, album.ExternalURLs.Spotify, album.Type)
	if err != nil {
		return "", fmt.Errorf("error inserting album: %w", err)
	}

	// then, make tracks
	_, err = tx.ExecContext(ctx, `
INSERT INTO spotify_tracks 
(id, name, href, uri, external_url, album_id, duration_ms, explicit, popularity) 
VALUES
($1, $2, $3, $4, $5, $6, $7, $8, $9)
ON CONFLICT
	DO NOTHING`,
		song.Track.ID, song.Track.Name, song.Track.Href, song.Track.URI, song.Track.ExternalURLs.Spotify,
		song.Track.Album.ID, song.Track.DurationMS, song.Track.Explicit, song.Track.Popularity)
	if err != nil {
		return "", fmt.Errorf("error inserting track: %w", err)
	}

	// THEN, make artists and their credits!
	for _, artist := range song.Track.Artists {
		// TODO(zeke): verify genres works!
		// make artists
		_, err = tx.ExecContext(ctx, `
INSERT INTO spotify_artists
(id, name, href, uri, external_url, genres, popularity)
VALUES
($1, $2, $3, $4, $5, $6, $7)
ON CONFLICT
	DO NOTHING`,
			artist.ID, artist.Name, artist.Href, artist.URI, artist.ExternalURLs.Spotify,
			artist.Genres, artist.Popularity)
		if err != nil {
			return "", fmt.Errorf("error inserting artist [%v]: %w", artist.Name, err)
		}

		// make credits
		_, err = tx.ExecContext(ctx, `
INSERT INTO spotify_credits
(track_id, artist_id)
VALUES
($1, $2)
ON CONFLICT
	DO NOTHING `, song.Track.ID, artist.ID)
		if err != nil {
			return "", fmt.Errorf("error inserting credit for artist [%v]: %w", artist.Name, err)
		}
	}

	// FINALLY, make the play history
	contextBlob, err := song.ContextBlob()
	if err != nil {
		return "", fmt.Errorf("error encoding context blob: %w", err)
	}

	var trackID string
	err = tx.QueryRowContext(ctx, `
INSERT INTO spotify_played_tracks 
(user_id, played_at, track_id, context_blob)
VALUES
($1, $2, $3, $4)
ON CONFLICT
	DO NOTHING
RETURNING track_id`,
		userID, song.PlayedAt, song.Track.ID, contextBlob).
		Scan(&trackID)
	return trackID, err
}
