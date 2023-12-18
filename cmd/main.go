package main

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/zestze/metacritic/internal"
)

func main() {
	slog.Info("starting")
	router := gin.Default()
	internal.Register(router)

	err := router.Run("localhost:8080")
	if err != nil {
		slog.Error("critical error, shutting down", "error", err)
	}
}
func scrape(medium internal.Medium, startYear int, numPages int) {
	ctx := context.Background()
	for year := startYear; year <= time.Now().Year(); year++ {
		for i := 1; i <= numPages; i++ {

			logger := slog.With("medium", medium, "year", year, "page", i)
			logger.Info("going to fetch posts from metacritic")
			posts, err := internal.FetchPosts(ctx, internal.Options{
				Medium:  medium,
				MinYear: year,
				MaxYear: year,
				Page:    i,
			})
			if err != nil {
				panic(err)
			}

			logger.Info("going to persist posts to sqlite")
			ids, err := internal.PersistPosts(ctx, posts)
			if err != nil {
				panic(err)
			}

			logger.Info("persisted " + strconv.Itoa(len(ids)) + " items")

			// ensure we don't get blacklisted
			logger.Info("going to sleep")
			time.Sleep(5 * time.Second)
		}
	}
}
