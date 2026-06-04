package dummylogger

import "github.com/padiazg/go-crap/pkg/logger"

var _ logger.Logger = (*Logger)(nil)

// Logger is a blackhole logger.
type Logger struct {
	level logger.Level
}

// New creates a new Logger from Config.
func New(cfg *logger.Config) Logger {
	if cfg == nil {
		cfg = &logger.Config{Level: "debug"}
	}

	return Logger{level: logger.ParseLevel(cfg.Level)}
}

// Debug logs at debug level.
func (l Logger) Debug(msg string, args ...any) {}

// Info logs at info level.
func (l Logger) Info(msg string, args ...any) {}

// Warn logs at warn level.
func (l Logger) Warn(msg string, args ...any) {}

// Error logs at error level.
func (l Logger) Error(msg string, args ...any) {}

// Fatal logs at error level then exits.
func (l Logger) Fatal(msg string, args ...any) {}
