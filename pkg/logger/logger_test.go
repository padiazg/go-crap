package logger

import (
	"log/slog"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestLevel_String(t *testing.T) {
	tests := []struct {
		name  string
		want  string
		level Level
	}{
		{name: "DebugLevel", level: DebugLevel, want: "debug"},
		{name: "InfoLevel", level: InfoLevel, want: "info"},
		{name: "WarnLevel", level: WarnLevel, want: "warn"},
		{name: "ErrorLevel", level: ErrorLevel, want: "error"},
		{name: "FatalLevel", level: FatalLevel, want: "fatal"},
		{name: "InvalidLevel", level: -1, want: "info"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, tt.level.String())
		})
	}
}

func Test_parseLevel(t *testing.T) {
	tests := []struct {
		name  string
		level string
		want  Level
	}{
		{name: "debug", level: "debug", want: DebugLevel},
		{name: "info", level: "info", want: InfoLevel},
		{name: "warn", level: "warn", want: WarnLevel},
		{name: "error", level: "error", want: ErrorLevel},
		{name: "fatal_falls_back_to_info", level: "fatal", want: InfoLevel},
		{name: "empty_string_falls_back_to_info", level: "", want: InfoLevel},
		{name: "unknown_string_falls_back_to_info", level: "trace", want: InfoLevel},
		{name: "uppercase_falls_back_to_info", level: "DEBUG", want: InfoLevel},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, parseLevel(tt.level))
		})
	}
}

func Test_slogLevel(t *testing.T) {
	tests := []struct {
		name string
		l    Level
		want slog.Level
	}{
		{name: "debug", l: DebugLevel, want: slog.LevelDebug},
		{name: "info", l: InfoLevel, want: slog.LevelInfo},
		{name: "warn", l: WarnLevel, want: slog.LevelWarn},
		{name: "error", l: ErrorLevel, want: slog.LevelError},
		{name: "fatal_falls_back_to_info", l: FatalLevel, want: slog.LevelInfo},
		{name: "negative_falls_back_to_info", l: Level(-1), want: slog.LevelInfo},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, slogLevel(tt.l))
		})
	}
}
