package ztrace

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

type Options struct {
	ServiceName   string
	OTLPEndppoint string
	Enabled       bool
}

// TODO(zeke): want to deprecate this, but might be nice for working with gin
func Start(ctx context.Context, name string) (context.Context, tracer.Span) {
	// gin middleware oddity (this is just a hack for now)
	// the span is stored in the request's context exclusively
	if c, ok := ctx.(*gin.Context); ok {
		ctx = c.Request.Context()
	}
	span, spanCtx := tracer.StartSpanFromContext(ctx, name)
	return spanCtx, span
}

type Shutdowner interface {
	Shutdown(ctx context.Context) error
}

// TODO(zeke): is this still necessary?
// often the parent context will already have been canceled
// so set our own internal timeout for shutting down the tracer provider
func Shutdown(ctx context.Context, s Shutdowner, timeout time.Duration) {
	if ctx.Err() != nil {
		// if parent context is already cancelled, fallback to background
		// this is super likely, since we're shutting down.
		ctx = context.Background()
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	if err := s.Shutdown(ctx); err != nil {
		slog.Error("error shutting down tracing", "error", err)
	}
}
