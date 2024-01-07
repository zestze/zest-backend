package metacritic

import (
	"context"
	"log/slog"
	"time"

	"database/sql"

	"github.com/zestze/zest-backend/internal/zlog"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

var DB_FILE_NAME = "internal/metacritic/store.db"

func PersistPosts(ctx context.Context, posts []Post) ([]int64, error) {
	logger := zlog.Logger(ctx)
	db, err := openDB(logger)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	stmt, err := db.PrepareContext(ctx,
		`INSERT OR IGNORE INTO posts 
		(title, href, score, description, release_date, medium, requested_at)
		VALUES
		(?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		logger.Error("error preparing statement", "error", err)
		return nil, err
	}
	defer stmt.Close()

	tx, err := db.Begin()
	if err != nil {
		logger.Error("error beginning transaction", "error", err)
		return nil, err
	}

	ids := make([]int64, len(posts))
	for i, post := range posts {
		result, err := tx.Stmt(stmt).
			ExecContext(ctx,
				post.Title, post.Href, post.Score,
				post.Description, post.ReleaseDate.Unix(),
				post.Medium, post.RequestedAt.Unix())
		if err != nil {
			logger.Error("error persisting post", "title", post.Title,
				"error", err)
			tx.Rollback()
			return nil, err
		}

		id, err := result.LastInsertId()
		if err != nil {
			logger.Error("error fetching id", "error", err)
			tx.Rollback()
			return nil, err
		}

		ids[i] = id
	}

	return ids, tx.Commit()
}

func GetPosts(ctx context.Context, opts Options) ([]Post, error) {
	logger := zlog.Logger(ctx)
	db, err := openDB(logger)
	if err != nil {
		return nil, err
	}
	defer db.Close()

	lowerBound, upperBound := opts.RangeAsEpoch()
	rows, err := db.QueryContext(ctx,
		`SELECT title, href, score, description, release_date, requested_at
		FROM posts 
		WHERE medium=? and ? <= release_date and release_date <= ?
		ORDER BY score DESC`,
		opts.Medium, lowerBound, upperBound)
	if err != nil {
		logger.Error("error querying for rows", "error", err)
		return nil, err
	}
	defer rows.Close()

	posts := make([]Post, 0)
	for rows.Next() {
		var (
			releaseDateAsEpoch, requestedAtAsEpoch int64
			post                                   Post
		)
		if err := rows.Scan(&post.Title, &post.Href, &post.Score, &post.Description,
			&releaseDateAsEpoch, &requestedAtAsEpoch); err != nil {
			return nil, err
		}
		post.ReleaseDate = time.Unix(releaseDateAsEpoch, 0).UTC()
		post.RequestedAt = time.Unix(requestedAtAsEpoch, 0).UTC()
		post.Medium = opts.Medium

		posts = append(posts, post)
	}

	return posts, nil
}

func openDB(logger *slog.Logger) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", DB_FILE_NAME)
	if err != nil {
		logger.Error("error opening db", "error", err)
		return nil, err
	}
	return db, nil
}

func Reset(ctx context.Context) error {
	logger := zlog.Logger(ctx)
	db, err := openDB(logger)
	if err != nil {
		logger.Error("error resetting table", "error", err)
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
		DROP TABLE IF EXISTS posts;
		CREATE TABLE IF NOT EXISTS posts (
			id 			 INTEGER PRIMARY KEY AUTOINCREMENT,
			title 		 TEXT UNIQUE,
			href		 TEXT,
			score		 INTEGER,
			description  TEXT,
			release_date INTEGER,
			medium 		 VARCHAR(10),
			requested_at INTEGER
		);
	`)
	if err != nil {
		logger.Error("error running reset sql", "error", err)
		return err
	}
	return nil
}
