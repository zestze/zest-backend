package zlog

import (
	"context"
	"log/slog"

	"github.com/zestze/zest-backend/internal/requestid"
)

// Logger makes a new logger with a request id set as an attribute.
func Logger(ctx context.Context) *slog.Logger {
	rid, ok := requestid.From(ctx)
	if !ok {
		return slog.Default()
	}
	return slog.With(slog.String("request_id", rid))
}
