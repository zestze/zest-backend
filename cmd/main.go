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
	"go.opentelemetry.io/otel"

	"github.com/alecthomas/kong"
)

// TODO(zeke): move tracing to separate pkg!
var tracer = otel.Tracer("zest-api")

var cli struct {
	Port         int    `short:"p" env:"PORT" default:"8080" help:"port to run server on"`
	ForceScrape  bool   `short:"f" help:"force a scrape operation"`
	OtlpEndpoint string `short:"e" env:"OTLP_ENDPOINT" default:"tempo:4318" help:"otlp endpoint for trace exporters"`
	ServiceName  string `short:"s" env:"SERVICE_NAME" default:"zest"`
}

func main() {
	kong.Parse(&cli)

	if cli.ForceScrape {
		// TODO(zeke): make this more configurable!
		//scrapeReddit(context.Background(), false)
		for _, m := range metacritic.AvailableMediums {
			scrapeMetacritic(m, 1995, 5)
		}
		return
	}

	addr := ":" + strconv.Itoa(cli.Port)
	slog.Info("going to start on " + addr)

	ctx := context.Background()
	tp, err := ztrace.New(ctx, ztrace.Options{
		ServiceName:   cli.ServiceName,
		OTLPEndppoint: cli.OtlpEndpoint,
	})
	if err != nil {
		slog.Error("error setting up tracer", "error", err)
		return
	}
	defer func() {
		if err := tp.Shutdown(ctx); err != nil {
			slog.Error("error shutting down tracer provider", "error", err)
		}
	}()

	router := gin.Default()
	router.Use(cors.Default())
	router.Use(requestid.New())
	router.Use(otelgin.Middleware(cli.ServiceName,
		otelgin.WithSpanNameFormatter(func(r *http.Request) string {
			return "HTTP " + r.Method + " " + r.URL.Path
		})))

	{
		v1 := router.Group("v1")
		metacritic.Register(v1)
		reddit.Register(v1)
	}

	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "OK"})
	})

	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	err = router.Run(addr)
	if err != nil {
		slog.Error("critical error, shutting down", "error", err)
	}
}
