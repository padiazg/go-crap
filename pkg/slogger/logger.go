package slogger

import (
	"context"
	"log/slog"
	"os"

	"github.com/padiazg/go-crap/pkg/logger"
)

var _ logger.Logger = (*Logger)(nil)

// Logger is the application logger backed by log/slog.
type Logger struct {
	slog  *slog.Logger
	level logger.Level
}

// New creates a new Logger from Config.
func New(cfg *logger.Config) Logger {
	lvl := logger.ParseLevel(cfg.Level)
	opts := &slog.HandlerOptions{Level: slogLevel(lvl)}

	var handler slog.Handler
	if cfg.Format == "json" {
		handler = slog.NewJSONHandler(os.Stdout, opts)
	} else {
		handler = slog.NewTextHandler(os.Stdout, opts)
	}
	return Logger{
		slog:  slog.New(handler),
		level: lvl,
	}
}

func slogLevel(l logger.Level) slog.Level {
	switch l {
	case logger.DebugLevel:
		return slog.LevelDebug
	case logger.InfoLevel:
		return slog.LevelInfo
	case logger.WarnLevel:
		return slog.LevelWarn
	case logger.ErrorLevel:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Debug logs at debug level.
func (l Logger) Debug(msg string, args ...any) {
	if l.level <= logger.DebugLevel {
		l.slog.DebugContext(context.Background(), msg, args...)
	}
}

// Info logs at info level.
func (l Logger) Info(msg string, args ...any) {
	if l.level <= logger.InfoLevel {
		l.slog.InfoContext(context.Background(), msg, args...)
	}
}

// Warn logs at warn level.
func (l Logger) Warn(msg string, args ...any) {
	if l.level <= logger.WarnLevel {
		l.slog.WarnContext(context.Background(), msg, args...)
	}
}

// Error logs at error level.
func (l Logger) Error(msg string, args ...any) {
	if l.level <= logger.ErrorLevel {
		l.slog.ErrorContext(context.Background(), msg, args...)
	}
}

// Fatal logs at error level then exits.
func (l Logger) Fatal(msg string, args ...any) {
	l.slog.ErrorContext(context.Background(), msg, args...)
	os.Exit(1)
}

// Slog returns the underlying *slog.Logger.
func (l Logger) Slog() *slog.Logger {
	return l.slog
}

// SlogLevel converts a Level to slog.Level.
func SlogLevel(l logger.Level) slog.Level {
	return slogLevel(l)
}

// Level returns the current log level.
func (l Logger) Level() logger.Level {
	return l.level
}
