package logger

import (
	"context"
	"log/slog"
	"os"
	"time"
)

// Level represents a log level.
type Level int

const (
	DebugLevel Level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

func (l Level) String() string {
	switch l {
	case DebugLevel:
		return "debug"
	case InfoLevel:
		return "info"
	case WarnLevel:
		return "warn"
	case ErrorLevel:
		return "error"
	case FatalLevel:
		return "fatal"
	default:
		return "info"
	}
}

// Config holds logger configuration.
type Config struct {
	Level  string // "debug", "info", "warn", "error"
	Format string // "text", "json"
}

// Logger is the application logger backed by log/slog.
type Logger struct {
	slog  *slog.Logger
	level Level
}

// New creates a new Logger from Config.
func New(cfg *Config) Logger {
	lvl := parseLevel(cfg.Level)
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

func parseLevel(level string) Level {
	switch level {
	case "debug":
		return DebugLevel
	case "warn":
		return WarnLevel
	case "error":
		return ErrorLevel
	default:
		return InfoLevel
	}
}

func slogLevel(l Level) slog.Level {
	switch l {
	case DebugLevel:
		return slog.LevelDebug
	case InfoLevel:
		return slog.LevelInfo
	case WarnLevel:
		return slog.LevelWarn
	case ErrorLevel:
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Debug logs at debug level.
func (l Logger) Debug(msg string, args ...any) {
	if l.level <= DebugLevel {
		l.slog.DebugContext(context.Background(), msg, args...)
	}
}

// Info logs at info level.
func (l Logger) Info(msg string, args ...any) {
	if l.level <= InfoLevel {
		l.slog.InfoContext(context.Background(), msg, args...)
	}
}

// Warn logs at warn level.
func (l Logger) Warn(msg string, args ...any) {
	if l.level <= WarnLevel {
		l.slog.WarnContext(context.Background(), msg, args...)
	}
}

// Error logs at error level.
func (l Logger) Error(msg string, args ...any) {
	if l.level <= ErrorLevel {
		l.slog.ErrorContext(context.Background(), msg, args...)
	}
}

// Fatal logs at error level then exits.
func (l Logger) Fatal(msg string, args ...any) {
	l.slog.ErrorContext(context.Background(), msg, args...)
	os.Exit(1)
}

// Package-level convenience functions.
var defaultLogger = New(&Config{Level: "info", Format: "text"})

// Debug logs at debug level using the default logger.
func Debug(msg string, args ...any) {
	defaultLogger.Debug(msg, args...)
}

// Info logs at info level using the default logger.
func Info(msg string, args ...any) {
	defaultLogger.Info(msg, args...)
}

// Warn logs at warn level using the default logger.
func Warn(msg string, args ...any) {
	defaultLogger.Warn(msg, args...)
}

// Error logs at error level using the default logger.
func Error(msg string, args ...any) {
	defaultLogger.Error(msg, args...)
}

// Fatal logs at error level then exits using the default logger.
func Fatal(msg string, args ...any) {
	defaultLogger.Fatal(msg, args...)
}

// Time logs duration with key at info level using the default logger.
func Time(key string, start time.Time) {
	Info(key, "elapsed", time.Since(start).String())
}

// Slog returns the underlying *slog.Logger.
func (l Logger) Slog() *slog.Logger {
	return l.slog
}

// SlogLevel converts a Level to slog.Level.
func SlogLevel(l Level) slog.Level {
	return slogLevel(l)
}

// Level returns the current log level.
func (l Logger) Level() Level {
	return l.level
}
