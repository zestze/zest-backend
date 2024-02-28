package zlog

import (
	"context"
	"log/slog"
)

// Logger makes a new logger with a request id set as an attribute.
func Logger(ctx context.Context) *slog.Logger {
	rid, ok := requestID(ctx)
	if !ok {
		return slog.Default()
	}
	return slog.With(slog.String("request_id", rid))
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
