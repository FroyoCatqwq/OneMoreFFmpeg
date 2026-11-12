// Package logging configures slog with JSON or tinted text output.
package logging

import (
	"context"
	"io"
	"log/slog"

	"github.com/lmittmann/tint"

	"github.com/FroyoCatqwq/OneMoreFFmpeg/internal/config"
	apperrors "github.com/FroyoCatqwq/OneMoreFFmpeg/internal/errors"
)

type contextKey string

const loggerKey contextKey = "onemoreffmpeg.logger"

func New(w io.Writer, cfg config.LogConfig) (*slog.Logger, error) {
	level, err := parseLevel(cfg.Level)
	if err != nil {
		return nil, err
	}
	if cfg.JSON {
		return slog.New(slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})), nil
	}
	return slog.New(tint.NewTextHandler(w, &tint.Options{
		Level:   level,
		NoColor: !colorEnabled(w, cfg.Color),
	})), nil
}

func parseLevel(level string) (slog.Level, error) {
	switch level {
	case "error":
		return slog.LevelError, nil
	case "warn":
		return slog.LevelWarn, nil
	case "info", "":
		return slog.LevelInfo, nil
	case "debug":
		return slog.LevelDebug, nil
	case "trace":
		return slog.LevelDebug - 4, nil
	default:
		return slog.LevelInfo, apperrors.New(apperrors.E7005, "invalid log level")
	}
}

func colorEnabled(w io.Writer, mode string) bool {
	switch mode {
	case "always":
		return true
	case "never", "":
		return false
	default:
		return supportsColor(w)
	}
}

func WithLogger(ctx context.Context, logger *slog.Logger) context.Context {
	return context.WithValue(ctx, loggerKey, logger)
}

func FromContext(ctx context.Context) *slog.Logger {
	if logger, ok := ctx.Value(loggerKey).(*slog.Logger); ok && logger != nil {
		return logger
	}
	return slog.Default()
}
