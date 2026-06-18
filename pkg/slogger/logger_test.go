package slogger

import (
	"log/slog"
	"testing"

	"github.com/padiazg/go-crap/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func Test_slogLevel(t *testing.T) {
	tests := []struct {
		name string
		l    logger.Level
		want slog.Level
	}{
		{name: "debug", l: logger.DebugLevel, want: slog.LevelDebug},
		{name: "info", l: logger.InfoLevel, want: slog.LevelInfo},
		{name: "warn", l: logger.WarnLevel, want: slog.LevelWarn},
		{name: "error", l: logger.ErrorLevel, want: slog.LevelError},
		{name: "error_falls_back_to_info", l: logger.FatalLevel, want: slog.LevelInfo},
		{name: "negative_falls_back_to_info", l: logger.Level(-1), want: slog.LevelInfo},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, slogLevel(tt.l))
		})
	}
}
