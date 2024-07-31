package metacritic

import (
	"context"
	"errors"

	"github.com/zestze/zest-backend/internal/user"
	"github.com/zestze/zest-backend/internal/zql"

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
	ctx, span := ztrace.Start(ctx, "SQL metacritic.Persist")
	defer span.End()
	logger := zlog.Logger(ctx)

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
			return nil, zql.Rollback(tx, err)
		}
		ids = append(ids, id)
	}

	return ids, tx.Commit()
}

func (s Store) GetPosts(ctx context.Context, opts Options) ([]Post, error) {
	ctx, span := ztrace.Start(ctx, "SQL metacritic.Get")
	defer span.End()
	logger := zlog.Logger(ctx)

	lowerBound, upperBound := opts.RangeAsDate()
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, title, href, score, description, released, created_at
		FROM metacritic_posts 
		WHERE medium = $1 and $2 <= released and released <= $3
		ORDER BY released DESC`,
		opts.Medium, lowerBound, upperBound)
	if err != nil {
		logger.Error("error querying for rows", "error", err)
		return nil, err
	}
	defer rows.Close()

	posts := make([]Post, 0)
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.ID, &post.Title, &post.Href, &post.Score, &post.Description,
			&post.ReleaseDate, &post.RequestedAt); err != nil {
			return nil, err
		}
		post.Medium = opts.Medium

		posts = append(posts, post)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func (s Store) SavePostsForUser(ctx context.Context, ids []int64, userID user.ID, action Action) error {
	ctx, span := ztrace.Start(ctx, "SQL metacritic.Persist")
	defer span.End()
	logger := zlog.Logger(ctx)

	stmt, err := s.db.PrepareContext(ctx,
		`INSERT INTO saved_metacritic_posts
	(user_id, post_id, action)
	VALUES
	($1, $2, $3)
	ON CONFLICT DO NOTHING`)
	if err != nil {
		logger.Error("error preparing statement", "error", err)
		return err
	}
	defer stmt.Close()

	tx, err := s.db.Begin()
	if err != nil {
		logger.Error("error beginning transaction", "error", err)
		return err
	}

	for _, id := range ids {
		_, err := tx.Stmt(stmt).
			ExecContext(ctx, userID, id)
		if err != nil {
			logger.Error("error exec", "error", err)
			return zql.Rollback(tx, err)
		}
	}
	return tx.Commit()
}

func (s Store) GetSavedPostsForUser(ctx context.Context, userID user.ID) ([]int64, error) {
	ctx, span := ztrace.Start(ctx, "SQL metacritic.Get")
	defer span.End()
	logger := zlog.Logger(ctx)

	rows, err := s.db.QueryContext(ctx,
		// TODO(zeke): likely want to fetch more than IDs!
		`SELECT post_id 
	FROM saved_metacritic_posts
	WHERE user_id = $1`,
		userID)
	if err != nil {
		logger.Error("error querying for rows", "error", err)
		return nil, err
	}
	var ids []int64
	for rows.Next() {
		var i int64
		if err := rows.Scan(&i); err != nil {
			return nil, err
		}
		ids = append(ids, i)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return ids, nil
}

// Reset is primarily for running Sqlite3 tests
func (s Store) Reset(ctx context.Context) error {
	logger := zlog.Logger(ctx)

	if _, err := s.db.Exec(`
		DROP TABLE IF EXISTS metacritic_posts;
		CREATE TABLE IF NOT EXISTS metacritic_posts (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			title        TEXT UNIQUE,
			href         TEXT,
			score        INTEGER,
			description  TEXT,
			released     INTEGER,
			medium       VARCHAR(10),
			created_at   INTEGER
		);`); err != nil {
		logger.Error("error running reset sql", "error", err)
		return err
	}
	return nil
}
