package reddit

import (
	"context"
	"database/sql"
	"log/slog"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
)

var DB_FILE_NAME = "internal/reddit/store.db"

func PersistPosts(ctx context.Context, savedPosts []Child) ([]int64, error) {
	db, err := openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	stmt, err := db.PrepareContext(ctx,
		`INSERT OR IGNORE INTO saved_posts
		(permalink, subreddit, num_comments, upvote_ratio, ups, score,
		total_awards_received, suggested_sort)
		VALUES
		(?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		slog.Error("error preparing statement", "error", err)
		return nil, err
	}
	defer stmt.Close()

	tx, err := db.Begin()
	if err != nil {
		slog.Error("error beginning transaction", "error", err)
		return nil, err
	}

	ids := make([]int64, len(savedPosts))
	for i, post := range savedPosts {
		result, err := tx.Stmt(stmt).
			ExecContext(ctx,
				post.Permalink, post.Subreddit, post.NumComments,
				post.UpvoteRatio, post.Ups, post.Score,
				post.TotalAwardsReceived, post.SuggestedSort)
		if err != nil {
			slog.Error("error persisting post", "permalink", post.Permalink,
				"error", err)
			tx.Rollback()
			return nil, err
		}

		id, err := result.LastInsertId()
		if err != nil {
			slog.Error("error fetching id", "error", err)
			tx.Rollback()
			return nil, err
		}

		ids[i] = id
	}

	return ids, nil
}

func GetAllPosts(ctx context.Context) ([]Child, error) {
	db, err := openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx,
		`SELECT permalink, subreddit
		FROM saved_posts
		ORDER BY ups DESC`)
	if err != nil {
		slog.Error("error querying for rows", "error", err)
		return nil, err
	}
	defer rows.Close()

	posts := make([]Child, 0)
	for rows.Next() {
		var post Child
		if err := rows.Scan(&post.Permalink, &post.Subreddit); err != nil {
			return nil, err
		}

		posts = append(posts, post)
	}

	return posts, nil
}

func GetSubreddits(ctx context.Context) ([]string, error) {
	db, err := openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx,
		`SELECT DISTINCT(subreddit)
		FROM saved_posts
		ORDER BY subreddit asc`)
	if err != nil {
		slog.Error("error querying for rows", "error", err)
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
	return subreddits, nil
}

func GetPostsFor(ctx context.Context, subreddit string) ([]Child, error) {
	db, err := openDB()
	if err != nil {
		return nil, err
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx,
		`SELECT permalink 
		FROM saved_posts
		WHERE subreddit=?
		ORDER BY ups DESC`,
		subreddit)
	if err != nil {
		slog.Error("error querying for rows", "error", err)
		return nil, err
	}
	defer rows.Close()

	posts := make([]Child, 0)
	for rows.Next() {
		post := Child{
			Subreddit: subreddit,
		}
		if err := rows.Scan(&post.Permalink); err != nil {
			return nil, err
		}

		posts = append(posts, post)
	}

	return posts, nil
}

func openDB() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", DB_FILE_NAME)
	if err != nil {
		slog.Error("error opening db", "error", err)
		return nil, err
	}
	return db, nil
}

func reset() error {
	db, err := openDB()
	if err != nil {
		slog.Error("error resetting table", "error", err)
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
		DROP TABLE IF EXISTS saved_posts;
		CREATE TABLE IF NOT EXISTS saved_posts (
			id					  INTEGER PRIMARY KEY AUTOINCREMENT,
			permalink 			  TEXT UNIQUE,
			subreddit 			  TEXT,
			num_comments 		  INTEGER,
			upvote_ratio 		  REAL,
			ups 				  INTEGER,
			score 				  INTEGER,
			total_awards_received INTEGER,
		);`)
	if err != nil {
		slog.Error("error running reset sql", "error", err)
	}
	return nil
}
