package main

import (
	"context"
	"log/slog"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	cors "github.com/rs/cors/wrapper/gin"
	"github.com/zestze/zest-backend/internal/metacritic"
	"github.com/zestze/zest-backend/internal/reddit"
	"go.opentelemetry.io/otel"

	stdout "go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/propagation"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
)

var tracer = otel.Tracer("zest-api")

func main() {
	slog.Info("going to start on :8080")

	tp, err := newTracer()
	if err != nil {
		slog.Error("error setting up tracer", "error", err)
		return
	}
	defer func() {
		if err := tp.Shutdown(context.Background()); err != nil {
			slog.Error("error shutting down tracer provider", "error", err)
		}
	}()

	router := gin.Default()
	router.Use(cors.Default())

	{
		v1 := router.Group("v1")
		metacritic.Register(v1)
		reddit.Register(v1)
	}

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	err = router.Run(":8080")
	if err != nil {
		slog.Error("critical error, shutting down", "error", err)
	}
}

func newTracer() (*sdktrace.TracerProvider, error) {
	exporter, err := stdout.New(stdout.WithPrettyPrint())
	if err != nil {
		return nil, err
	}
	tp := sdktrace.NewTracerProvider(
		sdktrace.WithSampler(sdktrace.AlwaysSample()),
		sdktrace.WithBatcher(exporter),
	)
	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(propagation.TraceContext{},
		propagation.Baggage{}))
	return tp, nil
}

func scrape(medium metacritic.Medium, startYear int, numPages int) {
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
			time.Sleep(5 * time.Second)
		}
	}
}
