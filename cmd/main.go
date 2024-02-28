package main

import (
	"context"
	"errors"
	sloggin "github.com/samber/slog-gin"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	cors "github.com/rs/cors/wrapper/gin"
	"github.com/zestze/zest-backend/internal/metacritic"
	"github.com/zestze/zest-backend/internal/reddit"
	"github.com/zestze/zest-backend/internal/spotify"
	"github.com/zestze/zest-backend/internal/user"
	"github.com/zestze/zest-backend/internal/zql"
	"github.com/zestze/zest-backend/internal/ztrace"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"golang.org/x/sync/errgroup"

	"github.com/alecthomas/kong"
)

func main() {
	ctx := kong.Parse(&cli)
	ctx.FatalIfErrorf(ctx.Run())
}

var cli struct {
	Server ServerCmd `cmd:"" help:"run server"`
	Scrape ScrapeCmd `cmd:"" help:"scrape the internet"`
	Dump   DumpCmd   `cmd:"" help:"dump from sqlite to postgres"`
}

type ServerCmd struct {
	Port          int           `short:"p" env:"PORT" default:"8080" help:"port to run server on"`
	OtlpEndpoint  string        `short:"e" env:"OTLP_ENDPOINT" default:"tempo:4317" help:"otlp endpoint for trace exporters"`
	ServiceName   string        `short:"n" env:"SERVICE_NAME" default:"zest"`
	SessionLength time.Duration `env:"SESSION_LENGTH" default:"15m" help:"maximum length of user session"`
	EnableTracing bool          `short:"t" env:"ENABLE_TRACING" help:"set to start tracing"`
}

func (r *ServerCmd) Group() slog.Attr {
	return slog.Group("server", slog.Int("port", r.Port),
		slog.String("otlp_endpoint", r.OtlpEndpoint),
		slog.String("service_name", r.ServiceName),
		slog.String("session_length", r.SessionLength.String()),
		slog.Bool("enable_tracing", r.EnableTracing))
}

func (r *ServerCmd) Run() error {
	ctx := context.Background()
	addr := ":" + strconv.Itoa(r.Port)
	if !gin.IsDebugging() {
		jsonLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil)).
			With("container", os.Getenv("HOSTNAME"))
		slog.SetDefault(jsonLogger)
	}
	logger := slog.Default().With(r.Group())
	logger.Info("starting server on " + addr)

	logger.Info("setting up tracer")
	tp, err := ztrace.New(ctx, ztrace.Options{
		ServiceName:   r.ServiceName,
		OTLPEndppoint: r.OtlpEndpoint,
		Enabled:       r.EnableTracing,
	})
	if err != nil {
		logger.Error("error setting up tracer", "error", err)
		return err
	}
	defer ztrace.Shutdown(ctx, tp, 2*time.Second)

	router := gin.New()
	router.Use(
		sloggin.NewWithFilters(slog.Default(),
			sloggin.IgnorePath("/metrics"),
			sloggin.IgnorePath("/health")),
		gin.Recovery(),
		cors.Default(),
		otelgin.Middleware(
			r.ServiceName,
			otelgin.WithSpanNameFormatter(ztrace.SpanName),
		),
	)

	logger.Info("setting up db connection")
	db, err := zql.WithMigrations()
	if err != nil {
		logger.Error("error initializing db", "error", err)
		return err
	}
	defer db.Close()

	logger.Info("setting up services")
	session := user.NewSession(r.SessionLength)
	uService := user.New(session, db)
	uService.Register(router)

	{
		v1 := router.Group("v1")
		auth := user.Auth(session)

		mService := metacritic.New(db)
		mService.Register(v1, auth)

		rService, err := reddit.New(db)
		if err != nil {
			logger.Error("error setting up reddit service", "error", err)
			return err
		}
		rService.Register(v1, auth)

		sService, err := spotify.New(db)
		if err != nil {
			logger.Error("error setting up spotify service", "error", err)
			return err
		}
		sService.Register(v1, auth)
	}

	{
		router.GET("/health", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"status": "OK"})
		})

		router.GET("/metrics", gin.WrapH(promhttp.Handler()))
	}

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	// logic for graceful shutdown
	ctx, stop := signal.NotifyContext(ctx, os.Interrupt)
	defer stop()

	logger.Info("running server")
	g, ctx := errgroup.WithContext(ctx)
	g.Go(func() error {
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			return err
		}
		return nil
	})
	g.Go(func() error {
		<-ctx.Done()
		stop()
		return srv.Shutdown(ctx)
	})

	if err = g.Wait(); err != nil {
		logger.Error("error from server", "error", err)
	}
	logger.Info("gracefully shutting down")

	return nil
}

type DumpCmd struct {
	BaseDir        string `help:"base directory for sqlite files" default:"./sqlite-temp"`
	RedditFile     string `help:"name of reddit sqlite file" default:"reddit.db"`
	MetacriticFile string `help:"name of metacritic sqlite file" default:"metacritic.db"`
	UserFile       string `help:"name of user sqlite file" default:"user.db"`
}

func (r *DumpCmd) Run() error {
	//TransferSpotifyToken(context.Background())
	Transfer(context.Background(), r.BaseDir, r.RedditFile, r.MetacriticFile, r.UserFile)
	return nil
}

type ScrapeCmd struct {
	Target string `arg:"" enum:"reddit,metacritic" help:"where to scrape from"`
	Reset  bool   `help:"if the db should be reset"`
}

func (r *ScrapeCmd) Run() error {
	ctx := context.Background()
	if r.Target == "reddit" {
		scrapeReddit(ctx, true, false)
	} else if r.Target == "metacritic" {
		for _, m := range metacritic.AvailableMediums {
			scrapeMetacritic(m, 1995, 5)
		}
	}
	return nil
}
