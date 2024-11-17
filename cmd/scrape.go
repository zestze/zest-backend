package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/zestze/zest-backend/internal/metacritic"
	"github.com/zestze/zest-backend/internal/reddit"
	"github.com/zestze/zest-backend/internal/zql"
)

func scrapeReddit(ctx context.Context, persistToFile, reset bool) {
	db, err := zql.Postgres()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	svc, err := reddit.New(db, http.DefaultTransport)
	if err != nil {
		panic(err)
	}
	if reset {
		if err := svc.Store.Reset(ctx); err != nil {
			panic(err)
		}
	}

	savedPosts, err := svc.Client.Fetch(ctx, reset)
	if err != nil {
		panic(err)
	}

	slog.Info("successfully fetched posts", slog.Int("num_post", len(savedPosts)))

	if persistToFile {
		f, err := os.Create("temp_posts.json")
		if err != nil {
			panic(err)
		}
		defer f.Close()

		if err := json.NewEncoder(f).Encode(savedPosts); err != nil {
			panic(err)
		}
	}

	// TODO(zeke): hardcoding userID 1 for now
	const HARDCODE_ZEKE_ID = 1
	ids, err := svc.Store.PersistPosts(ctx, savedPosts, HARDCODE_ZEKE_ID)
	if err != nil {
		panic(err)
	}

	slog.Info("successfully persisted posts", slog.Int("num_persisted", len(ids)))
}

func scrapeMetacritic(medium metacritic.Medium, startYear int, numPages int) {
	ctx := context.Background()
	db, err := zql.Postgres()
	if err != nil {
		panic(err)
	}
	defer db.Close()
	svc := metacritic.New(db, http.DefaultTransport)
	for year := startYear; year <= time.Now().Year(); year++ {
		for i := 1; i <= numPages; i++ {

			logger := slog.With("medium", medium, "year", year, "page", i)
			logger.Info("going to fetch posts from metacritic")
			posts, err := svc.Client.FetchPosts(ctx, metacritic.Options{
				Medium:  medium,
				MinYear: year,
				MaxYear: year,
				Page:    i,
			})
			if err != nil {
				panic(err)
			}

			logger.Info("going to persist posts to sqlite")
			ids, err := svc.Store.PersistPosts(ctx, posts)
			if err != nil {
				panic(err)
			}

			logger.Info("persisted " + strconv.Itoa(len(ids)) + " items")

			// ensure we don't get blacklisted
			logger.Info("going to sleep")
			time.Sleep(1 * time.Second)
		}
	}
}
