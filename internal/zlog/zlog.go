package zlog

import (
	"context"
	"log/slog"
	"os"
	"strconv"

	"github.com/gin-gonic/gin"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

// env var name -> log key
var ddMap = map[string]string{
	"DD_SERVICE": "dd.service",
	"DD_ENV":     "dd.env",
	// TODO(zeke): add DD_VERSION ?
}

// Logger generates a slog.Logger that has dd-trace info injected as records
func Logger(ctx context.Context) *slog.Logger {
	logger := slog.Default()
	// TODO(zeke): it'd be nice to remove this, but the "blessed" slog package for ddtrace requires
	// 	all logs to use the context for its handler to function
	// TODO(zeke): gin middleware oddity, the span is stored in the request context
	//	store this logic somewhere central since also in ztrace
	if c, ok := ctx.(*gin.Context); ok {
		ctx = c.Request.Context()
	}
	span, ok := tracer.SpanFromContext(ctx)
	if ok {
		toSlog := func(fieldname string, id uint64) slog.Attr {
			// TODO(zeke): could this be slog.Uint64?
			return slog.String(fieldname, strconv.FormatUint(id, 10))
		}
		logger = logger.With(
			toSlog(ext.LogKeyTraceID, span.Context().TraceID()),
			toSlog(ext.LogKeySpanID, span.Context().SpanID()),
		)
		// TODO(zeke): not sure this is suuuper necessary, but adding.
		// could probably move the logic to the dd-agent.
		for envKey, logKey := range ddMap {
			v, ok := os.LookupEnv(envKey)
			if ok {
				logger = logger.With(slog.String(logKey, v))
			}
		}
	}
	return logger
}
