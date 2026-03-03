package logging

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"sync/atomic"
)

type ctxKey string

const requestIDContextKey ctxKey = "request_id"

type Config struct {
	Level  string
	Format string
}

var (
	loggerPtr atomic.Pointer[slog.Logger]
	levelVar            = &slog.LevelVar{}
	outWriter io.Writer = os.Stdout
	writerMu  sync.Mutex
)

func init() {
	levelVar.Set(slog.LevelInfo)
	loggerPtr.Store(newLogger(Config{Level: "info", Format: "console"}, outWriter))
}

func SetOutput(w io.Writer) {
	writerMu.Lock()
	defer writerMu.Unlock()
	if w == nil {
		outWriter = os.Stdout
	} else {
		outWriter = w
	}
}

func ApplyConfig(cfg Config) {
	writerMu.Lock()
	defer writerMu.Unlock()
	loggerPtr.Store(newLogger(cfg, outWriter))
}

func L() *slog.Logger {
	if logger := loggerPtr.Load(); logger != nil {
		return logger
	}
	return slog.Default()
}

func WithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, requestIDContextKey, requestID)
}

func RequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	requestID, _ := ctx.Value(requestIDContextKey).(string)
	return requestID
}

func ForContext(ctx context.Context) *slog.Logger {
	requestID := RequestIDFromContext(ctx)
	if requestID == "" {
		return L()
	}
	return L().With(slog.String("request_id", requestID))
}

func newLogger(cfg Config, output io.Writer) *slog.Logger {
	levelVar.Set(parseLevel(cfg.Level))
	options := &slog.HandlerOptions{
		Level: levelVar,
	}

	format := strings.ToLower(strings.TrimSpace(cfg.Format))
	switch format {
	case "json":
		return slog.New(slog.NewJSONHandler(output, options))
	default:
		return slog.New(slog.NewTextHandler(output, options))
	}
}

func parseLevel(level string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(level)) {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
