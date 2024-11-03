package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strconv"
	"time"

	"github.com/joho/godotenv"
	sloggin "github.com/samber/slog-gin"
	gintrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/gin-gonic/gin"
	httptrace "gopkg.in/DataDog/dd-trace-go.v1/contrib/net/http"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	cors "github.com/rs/cors/wrapper/gin"
	"github.com/zestze/zest-backend/internal/metacritic"
	"github.com/zestze/zest-backend/internal/publisher"
	"github.com/zestze/zest-backend/internal/reddit"
	"github.com/zestze/zest-backend/internal/spotify"
	"github.com/zestze/zest-backend/internal/user"
	"github.com/zestze/zest-backend/internal/zql"
	"golang.org/x/sync/errgroup"

	"github.com/alecthomas/kong"
)

func main() {
	err := godotenv.Load()
	if errors.Is(err, fs.ErrNotExist) {
		slog.Info("dotenv not found, ignoring and continuing")
	} else if err != nil {
		slog.Error("error loading dotenv file", "error", err)
		return
	}

	ctx := kong.Parse(&cli)
	ctx.FatalIfErrorf(ctx.Run())
}

var cli struct {
	Server   ServerCmd   `cmd:"" help:"run server"`
	Scrape   ScrapeCmd   `cmd:"" help:"scrape the internet"`
	Dump     DumpCmd     `cmd:"" help:"dump from sqlite to postgres"`
	Backfill BackfillCmd `cmd:"" help:"hit the server"`
}

type ServerCmd struct {
	Port          int           `short:"p" env:"PORT" default:"8080" help:"port to run server on"`
	OtlpEndpoint  string        `short:"e" env:"OTLP_ENDPOINT" default:"tempo:4317" help:"otlp endpoint for trace exporters"`
	ServiceName   string        `short:"n" env:"SERVICE_NAME" default:"zest"`
	SessionLength time.Duration `env:"SESSION_LENGTH" default:"15m" help:"maximum length of user session"`
	EnableTracing bool          `short:"t" env:"ENABLE_TRACING" help:"set to start tracing"`
	DogStatsdURL  string        `env:"DOGSTATSD_URL" default:""`
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

	router := gin.New()
	router.Use(
		sloggin.NewWithFilters(slog.Default(),
			sloggin.IgnorePath("/metrics"),
			sloggin.IgnorePath("/health")),
		gin.Recovery(),
		cors.Default(),
	)

	// TODO(zeke): rename otlp endpoint to something ddog specific
	if r.OtlpEndpoint != "" {
		logger.Info("setting up tracer")
		tracer.Start(
			tracer.WithEnv("prod"),
			tracer.WithService(r.ServiceName),
			tracer.WithServiceVersion("temp-sha"), // TODO(zeke): store git commit!
		)
		defer tracer.Stop()
		router.Use(gintrace.Middleware(r.ServiceName))
	}

	logger.Info("setting up db connection")
	db, err := zql.Postgres()
	if err != nil {
		logger.Error("error initializing db", "error", err)
		return err
	}
	defer db.Close()

	logger.Info("setting up services")
	session := user.NewSession(r.SessionLength)
	uService := user.New(session, db)
	uService.Register(router)

	rt := httptrace.WrapRoundTripper(http.DefaultTransport)
	{
		v1 := router.Group("v1")
		auth := user.Auth(session)

		mService := metacritic.New(db, rt)
		mService.Register(v1, auth)

		rService, err := reddit.New(db, rt)
		if err != nil {
			logger.Error("error setting up reddit service", "error", err)
			return err
		}
		rService.Register(v1, auth)

		publisher, err := r.publisher(ctx)
		if err != nil {
			logger.Error("error making publisher", "error", err)
			return err
		}
		sService, err := spotify.New(ctx, db, publisher, rt)
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

func (r *ServerCmd) publisher(ctx context.Context) (spotify.Publisher, error) {
	if r.DogStatsdURL != "" {
		return publisher.New(ctx, r.DogStatsdURL)
	} else {
		return fakePublisher{}, nil
	}

}

type DumpCmd struct {
	BaseDir        string `help:"base directory for sqlite files" default:"./sqlite-temp"`
	RedditFile     string `help:"name of reddit sqlite file" default:"reddit.db"`
	MetacriticFile string `help:"name of metacritic sqlite file" default:"metacritic.db"`
	UserFile       string `help:"name of user sqlite file" default:"user.db"`
}

func (r *DumpCmd) Run() error {
	//TransferSpotifyToken(context.Background())
	//Transfer(context.Background(), r.BaseDir, r.RedditFile, r.MetacriticFile, r.UserFile)
	TransferSpotifySongs(context.Background())
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

type fakePublisher struct{}

func (fakePublisher) Publish(ctx context.Context, message any) error {
	return nil
}
