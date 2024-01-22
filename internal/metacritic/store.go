package metacritic

import (
	"context"
	"errors"

	"database/sql"

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

func (s Store) PersistPosts(ctx context.Context, posts []Post) ([]int64, error) {
	logger := zlog.Logger(ctx)
	ctx, span := ztrace.Start(ctx, "SQL metacritic.Persist")
	defer span.End()

	stmt, err := s.db.PrepareContext(ctx,
		`INSERT INTO metacritic_posts 
		(title, href, score, description, released, medium, created_at)
		VALUES
		($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT DO NOTHING
		RETURNING id`)
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

	ids := make([]int64, 0, len(posts))
	for _, post := range posts {
		var id int64
		err := tx.Stmt(stmt).
			QueryRowContext(ctx,
				post.Title, post.Href, post.Score,
				post.Description, post.ReleaseDate.UTC(),
				post.Medium, post.RequestedAt.UTC()).
			Scan(&id)
		if err != nil && errors.Is(err, sql.ErrNoRows) {
			continue
		} else if err != nil {
			logger.Error("error persisting post", "title", post.Title,
				"error", err)
			tx.Rollback()
			return nil, err
		}
		ids = append(ids, id)
	}

	return ids, tx.Commit()
}

func (s Store) GetPosts(ctx context.Context, opts Options) ([]Post, error) {
	logger := zlog.Logger(ctx)
	ctx, span := ztrace.Start(ctx, "SQL metacritic.Get")
	defer span.End()

	lowerBound, upperBound := opts.RangeAsEpoch()
	rows, err := s.db.QueryContext(ctx,
		`SELECT title, href, score, description, release_date, requested_at
		FROM metacritic_posts 
		WHERE medium = $1 and $2 <= release_date and release_date <= $3
		ORDER BY score DESC`,
		opts.Medium, lowerBound, upperBound)
	if err != nil {
		logger.Error("error querying for rows", "error", err)
		return nil, err
	}
	defer rows.Close()

	posts := make([]Post, 0)
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.Title, &post.Href, &post.Score, &post.Description,
			&post.ReleaseDate, &post.RequestedAt); err != nil {
			return nil, err
		}
		post.Medium = opts.Medium

		posts = append(posts, post)
	}

	return posts, nil
}

func (s Store) Reset(ctx context.Context) error {
	logger := zlog.Logger(ctx)

	_, err := s.db.Exec(`
		DROP TABLE IF EXISTS posts;
		CREATE TABLE IF NOT EXISTS posts (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			title        TEXT UNIQUE,
			href         TEXT,
			score        INTEGER,
			description  TEXT,
			release_date INTEGER,
			medium       VARCHAR(10),
			requested_at INTEGER
		);`)
	if err != nil {
		logger.Error("error running reset sql", "error", err)
		return err
	}
	return nil
}
