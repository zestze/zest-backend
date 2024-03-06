package reddit

import (
	"context"
	"database/sql"
	"errors"
	"github.com/zestze/zest-backend/internal/zql"

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

func (s Store) PersistPosts(
	ctx context.Context, savedPosts []Post, userID int,
) ([]int64, error) {
	ctx, span := ztrace.Start(ctx, "SQL reddit.Persist")
	defer span.End()
	logger := zlog.Logger(ctx)

	stmt, err := s.db.PrepareContext(ctx,
		`INSERT INTO reddit_posts 
		(permalink, subreddit, num_comments, upvote_ratio, ups, score,
		total_awards_received, suggested_sort,
		title, name, created_utc, user_id,
		link_title, body)
		VALUES
		($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14)
		ON CONFLICT (name)
			DO UPDATE SET 
			ups=excluded.ups,
			score=excluded.score,
			upvote_ratio=excluded.upvote_ratio,
			link_title=excluded.link_title,
			body=excluded.body
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

	ids := make([]int64, 0, len(savedPosts))
	for _, post := range savedPosts {
		var id int64
		err := tx.Stmt(stmt).
			QueryRowContext(ctx,
				post.Permalink, post.Subreddit, post.NumComments,
				post.UpvoteRatio, post.Ups, post.Score,
				post.TotalAwardsReceived, post.SuggestedSort,
				post.Title, post.Name, post.CreatedUTC,
				userID,
				post.LinkTitle, post.Body).
			Scan(&id)
		if err != nil && errors.Is(err, sql.ErrNoRows) {
			continue
		} else if err != nil {
			logger.Error("error persisting post", "permalink", post.Permalink,
				"error", err)
			return nil, zql.Rollback(tx, err)
		}

		ids = append(ids, id)
	}

	return ids, tx.Commit()
}

func (s Store) GetAllPosts(ctx context.Context, userID int) ([]Post, error) {
	ctx, span := ztrace.Start(ctx, "SQL reddit.Get")
	defer span.End()
	logger := zlog.Logger(ctx)

	rows, err := s.db.QueryContext(ctx,
		`SELECT permalink, subreddit, score, title, name, 
			created_utc, link_title, body
		FROM reddit_posts 
		WHERE user_id=$1
		ORDER BY created_utc DESC
		LIMIT 100`,
		userID)
	if err != nil {
		logger.Error("error querying for rows", "error", err)
		return nil, err
	}
	defer rows.Close()

	posts := make([]Post, 0)
	for rows.Next() {
		var post Post
		if err := rows.Scan(&post.Permalink, &post.Subreddit, &post.Score,
			&post.Title, &post.Name, &post.CreatedUTC,
			&post.LinkTitle, &post.Body); err != nil {
			return nil, err
		}

		posts = append(posts, post)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func (s Store) GetSubreddits(ctx context.Context, userID int) ([]string, error) {
	ctx, span := ztrace.Start(ctx, "SQL reddit.Get")
	defer span.End()
	logger := zlog.Logger(ctx)

	rows, err := s.db.QueryContext(ctx,
		`SELECT DISTINCT(subreddit)
		FROM reddit_posts
		WHERE user_id=$1
		ORDER BY subreddit asc`,
		userID)
	if err != nil {
		logger.Error("error querying for rows", "error", err)
	}
	defer rows.Close()

	subreddits := make([]string, 0)
	for rows.Next() {
		var sub string
		if err := rows.Scan(&sub); err != nil {
			return nil, err
		}
		subreddits = append(subreddits, sub)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	return subreddits, nil
}

func (s Store) GetPostsFor(ctx context.Context, subreddit string, userID int) ([]Post, error) {
	ctx, span := ztrace.Start(ctx, "SQL reddit.Get")
	defer span.End()
	logger := zlog.Logger(ctx)

	rows, err := s.db.QueryContext(ctx,
		`SELECT permalink, score, title, name, created_utc,
			link_title, body
		FROM reddit_posts 
		WHERE subreddit=$1 AND user_id=$2
		ORDER BY created_utc DESC`,
		subreddit, userID)
	if err != nil {
		logger.Error("error querying for rows", "error", err)
		return nil, err
	}
	defer rows.Close()

	posts := make([]Post, 0)
	for rows.Next() {
		post := Post{
			Subreddit: subreddit,
		}
		if err := rows.Scan(&post.Permalink, &post.Score,
			&post.Title, &post.Name, &post.CreatedUTC,
			&post.LinkTitle, &post.Body); err != nil {
			return nil, err
		}

		posts = append(posts, post)
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}

	return posts, nil
}

func (s Store) Reset(ctx context.Context) error {
	logger := zlog.Logger(ctx)

	if _, err := s.db.Exec(`
		DROP TABLE IF EXISTS reddit_posts;
		CREATE TABLE IF NOT EXISTS reddit_posts (
			id                    INTEGER PRIMARY KEY AUTOINCREMENT,
			name                  TEXT UNIQUE,
			user_id               INTEGER,
			permalink             TEXT,
			subreddit             TEXT,
			title                 TEXT,
			num_comments          INTEGER,
			upvote_ratio          REAL,
			ups                   INTEGER,
			score                 INTEGER,
			total_awards_received INTEGER,
			suggested_sort        TEXT,
			link_title            TEXT,
			body                  TEXT,
			created_utc           REAL,
			created_at            INTEGER
		);`); err != nil {
		logger.Error("error running reset sql", "error", err)
		return err
	}
	return nil
}
