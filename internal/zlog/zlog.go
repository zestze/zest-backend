package zlog

import (
	"context"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
)

const (
	// TraceIDKey, SpanIDKey, and RequestIDKey are constants for log attributes
	// the values are chosen to match `sloggin`
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
	spanCtx := trace.SpanContextFromContext(ctx)
	if spanCtx.IsValid() {
		// do things
		traceID, spanID := spanCtx.TraceID().String(), spanCtx.SpanID().String()
		logger = logger.With(slog.String(TraceIDKey, traceID),
			slog.String(SpanIDKey, spanID))
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
