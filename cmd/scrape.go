package main

import (
	"context"
	"encoding/json"
	"log/slog"
	"os"
	"strconv"
	"time"

	"github.com/zestze/zest-backend/internal/metacritic"
	"github.com/zestze/zest-backend/internal/reddit"
)

func scrapeReddit(ctx context.Context, persistToFile, reset bool) {
	if reset {
		if err := reddit.Reset(ctx); err != nil {
			panic(err)
		}
	}

	savedPosts, err := reddit.Fetch(ctx, reset)
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

	ids, err := reddit.PersistPosts(ctx, savedPosts)
	if err != nil {
		panic(err)
	}

	slog.Info("successfully persisted posts", slog.Int("num_persisted", len(ids)))
}

func scrapeMetacritic(medium metacritic.Medium, startYear int, numPages int) {
	ctx := context.Background()
	for year := startYear; year <= time.Now().Year(); year++ {
		for i := 1; i <= numPages; i++ {

			logger := slog.With("medium", medium, "year", year, "page", i)
			logger.Info("going to fetch posts from metacritic")
			posts, err := metacritic.FetchPosts(ctx, metacritic.Options{
				Medium:  medium,
				MinYear: year,
				MaxYear: year,
				Page:    i,
			})
			if err != nil {
				panic(err)
			}

			logger.Info("going to persist posts to sqlite")
			ids, err := metacritic.PersistPosts(ctx, posts)
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
