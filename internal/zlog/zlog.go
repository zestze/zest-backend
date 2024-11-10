package zlog

import (
	"context"
	"log/slog"
	"strconv"

	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/ext"
	"gopkg.in/DataDog/dd-trace-go.v1/ddtrace/tracer"
)

const (
	// TraceIDKey, SpanIDKey, and RequestIDKey are constants for log attributes
	// the values are chosen to match `sloggin`
	// TODO(zeke): maybe remove TraceIDKey and SpanIDKey?
	TraceIDKey   = "trace-id"
	SpanIDKey    = "span-id"
	RequestIDKey = "id"
)

// Logger makes a new logger with a request id set as an attribute.
// uses `id` for the request id attribute. This is because
// `sloggin` middleware sets the request id as `id` in slog.
func Logger(ctx context.Context) *slog.Logger {
	// requestID logic
	rid, ok := requestID(ctx)
	if !ok {
		return slog.Default()
	}
	logger := slog.With(slog.String(RequestIDKey, rid))
	// trace / span logic
	// TODO(zeke): it'd be nice to remove this, but the "blessed" slog package for ddtrace requires
	// all logs to use the context for its handler to function
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
	}
	return logger
}

// requestID grabs the request id from the context that is presumably set
// by the sloggin pkg. For more details, look here:
// https://github.com/samber/slog-gin/blob/main/middleware.go#L17
// and here:
// https://github.com/samber/slog-gin/blob/main/middleware.go#L258-L270
func requestID(ctx context.Context) (string, bool) {
	// TODO(zeke): maybe push a change to slog-gin to make this easier?
	v := ctx.Value("slog-gin.request-id")
	if v == nil {
		return "", false
	}
	r, ok := v.(string)
	if !ok {
		return "", false
	}
	return r, true
}
