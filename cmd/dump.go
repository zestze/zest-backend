// these commands are primarily for moving data around in my personal setup.
// not super important for any of the true logic
package main

import (
	"context"
	"time"

	"github.com/zestze/zest-backend/internal/metacritic"
	"github.com/zestze/zest-backend/internal/reddit"
	"github.com/zestze/zest-backend/internal/spotify"
	"github.com/zestze/zest-backend/internal/user"
	"github.com/zestze/zest-backend/internal/zql"
)

func TransferSpotifyToken(ctx context.Context) {
	// for running on baremetal
	targetDB, err := zql.PostgresWithOptions(zql.WithHost("localhost"))
	if err != nil {
		panic(err)
	}
	defer targetDB.Close()

	sourceDB, err := zql.Sqlite3("internal/spotify/spotify_test.db")
	if err != nil {
		panic(err)
	}
	defer sourceDB.Close()

	targetStore := spotify.NewStoreV1(targetDB)
	sourceStore := spotify.NewStoreV1(sourceDB)
	userID := 1 // hard-coded for me!
	token, err := sourceStore.GetToken(ctx, userID)
	if err != nil {
		panic(err)
	}
	err = targetStore.PersistToken(ctx, token, userID)
	if err != nil {
		panic(err)
	}
}

func TransferSpotifySongs(ctx context.Context) {
	db, err := zql.Postgres()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	storeV1 := spotify.NewStoreV1(db)
	storeV2 := spotify.NewStoreV2(db)
	userID := 1 // hard-coded for me!
	songs, err := storeV1.GetAll(ctx, userID)
	if err != nil {
		panic(err)
	}

	_, err = storeV2.PersistRecentlyPlayed(ctx, songs, userID)
	if err != nil {
		panic(err)
	}

}

func Transfer(ctx context.Context, directory, redditFile, metacriticFile, userFile string) {
	// first load the users!
	targetDB, err := zql.Postgres()
	if err != nil {
		panic(err)
	}
	defer targetDB.Close()

	users := getUsers(ctx, directory+"/"+userFile)
	userStore := user.NewStore(targetDB)
	for _, u := range users {
		_, err = userStore.PersistUser(ctx, u.username, u.password, u.salt)
		if err != nil {
			panic(err)
		}
	}

	// now, load the reddit posts!
	redditPosts := getRedditPosts(ctx, directory+"/"+redditFile)
	redditStore := reddit.NewStore(targetDB)
	_, err = redditStore.PersistPosts(ctx, redditPosts, 1)
	if err != nil {
		panic(err)
	}

	// finally, load the metacritic posts!
	metacriticPosts := getMetacriticPosts(ctx, directory+"/"+metacriticFile)
	metacriticStore := metacritic.NewStore(targetDB)
	_, err = metacriticStore.PersistPosts(ctx, metacriticPosts)
	if err != nil {
		panic(err)
	}
}

func getMetacriticPosts(ctx context.Context, filename string) []metacritic.Post {
	db, err := zql.Sqlite3(filename)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx,
		`SELECT title, href, score, description, release_date, medium, requested_at
		FROM posts
		ORDER BY id asc`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	posts := make([]metacritic.Post, 0)
	for rows.Next() {
		var (
			p                   metacritic.Post
			released, requested int64
		)
		if err = rows.Scan(&p.Title, &p.Href, &p.Score, &p.Description,
			&released, &p.Medium, &requested); err != nil {
			panic(err)
		}
		p.ReleaseDate = time.Unix(released, 0).UTC()
		p.RequestedAt = time.Unix(requested, 0).UTC()
		posts = append(posts, p)
	}
	return posts
}

func getRedditPosts(ctx context.Context, filename string) []reddit.Post {
	db, err := zql.Sqlite3(filename)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx,
		`SELECT permalink, subreddit, num_comments, upvote_ratio, ups, score, total_awards_received, suggested_sort, title, name, created_utc
		FROM saved_posts
		ORDER BY id ASC`)
	if err != nil {
		panic(err)
	}

	posts := make([]reddit.Post, 0)
	for rows.Next() {
		var p reddit.Post
		if err = rows.Scan(&p.Permalink, &p.Subreddit, &p.NumComments, &p.UpvoteRatio,
			&p.Ups, &p.Score, &p.TotalAwardsReceived, &p.SuggestedSort, &p.Title,
			&p.Name, &p.CreatedUTC); err != nil {
			panic(err)
		}
		posts = append(posts, p)
	}
	return posts
}

type User struct {
	id        int
	username  string
	password  string
	salt      int
	createdAt time.Time
}

func getUsers(ctx context.Context, filename string) []User {
	db, err := zql.Sqlite3(filename)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	rows, err := db.QueryContext(ctx,
		`SELECT id, username, password, salt, created_at
		FROM users`)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	users := make([]User, 0)
	for rows.Next() {
		var u User
		var seconds int64
		if err := rows.Scan(&u.id, &u.username, &u.password,
			&u.salt, &seconds); err != nil {
			panic(err)
		}
		u.createdAt = time.Unix(seconds, 0).UTC()
		users = append(users, u)
	}

	return users
}
