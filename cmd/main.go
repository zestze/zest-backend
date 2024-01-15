package main

import (
	"context"
	"errors"
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
	"github.com/zestze/zest-backend/internal/requestid"
	"github.com/zestze/zest-backend/internal/user"
	"github.com/zestze/zest-backend/internal/ztrace"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
	"golang.org/x/sync/errgroup"

	"github.com/alecthomas/kong"
)

var cli struct {
	Server ServerCmd `cmd:"" help:"run server"`
	Scrape ScrapeCmd `cmd:"" help:"scrape the internet"`
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

type ServerCmd struct {
	Port         int    `short:"p" env:"PORT" default:"8080" help:"port to run server on"`
	OtlpEndpoint string `short:"e" env:"OTLP_ENDPOINT" default:"tempo:4317" help:"otlp endpoint for trace exporters"`
	ServiceName  string `short:"n" env:"SERVICE_NAME" default:"zest"`
	WithAuth     bool   `env:"WITH_AUTH" help:"enables auth protection"`
}

func (r *ServerCmd) Run() error {
	ctx := context.Background()
	addr := ":" + strconv.Itoa(r.Port)
	slog.Info("going to start on " + addr)

	tp, err := ztrace.New(ctx, ztrace.Options{
		ServiceName:   r.ServiceName,
		OTLPEndppoint: r.OtlpEndpoint,
	})
	if err != nil {
		slog.Error("error setting up tracer", "error", err)
		return err
	}
	defer ztrace.ShutDown(ctx, tp, 2*time.Second)

	router := gin.Default()
	router.Use(cors.Default())
	router.Use(requestid.New())
	router.Use(otelgin.Middleware(r.ServiceName,
		otelgin.WithSpanNameFormatter(ztrace.SpanName)))

	uService, err := user.New()
	if err != nil {
		slog.Error("error setting up user service", "error", err)
		return err
	}
	uService.Register(router)
	defer uService.Close()

	{
		v1 := router.Group("v1")
		signingKey, err := user.NewSigningKey()
		if err != nil {
			slog.Error("error setting up signing key", "error", err)
			return err
		}
		if r.WithAuth {
			slog.Info("enabling auth protection")
			v1.Use(user.Auth(signingKey))
		}

		mService, err := metacritic.New()
		if err != nil {
			slog.Error("error setting up metacritic service", "error", err)
			return err
		}
		mService.Register(v1)
		defer mService.Close()

		rService, err := reddit.New()
		if err != nil {
			slog.Error("error setting up reddit service", "error", err)
			return err
		}
		rService.Register(v1)
		defer rService.Close()
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
		slog.Error("error from server", "error", err)
	}
	slog.Info("gracefully shutting down")

	return nil
}

func main() {
	ctx := kong.Parse(&cli)
	ctx.FatalIfErrorf(ctx.Run())
}
