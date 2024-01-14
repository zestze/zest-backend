package main

import (
	"context"
	"log/slog"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	cors "github.com/rs/cors/wrapper/gin"
	"github.com/zestze/zest-backend/internal/metacritic"
	"github.com/zestze/zest-backend/internal/reddit"
	"github.com/zestze/zest-backend/internal/requestid"
	"github.com/zestze/zest-backend/internal/ztrace"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"

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
	OtlpEndpoint string `short:"e" env:"OTLP_ENDPOINT" default:"tempo:4318" help:"otlp endpoint for trace exporters"`
	ServiceName  string `short:"n" env:"SERVICE_NAME" default:"zest"`
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
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			slog.Error("error shutting down tracer provider", "error", err)
		}
	}()

	router := gin.Default()
	router.Use(cors.Default())
	router.Use(requestid.New())
	router.Use(otelgin.Middleware(r.ServiceName,
		otelgin.WithSpanNameFormatter(ztrace.SpanName)))

	v1 := router.Group("v1")
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

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	err = router.Run(addr)
	if err != nil {
		slog.Error("critical error, shutting down", "error", err)
	}
	return nil
}

func main() {
	ctx := kong.Parse(&cli)
	ctx.FatalIfErrorf(ctx.Run())
}
