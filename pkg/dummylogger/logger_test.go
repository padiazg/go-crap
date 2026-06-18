package dummylogger

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/padiazg/go-crap/pkg/logger"
)

type NewFn func(*testing.T, Logger)

var checkNew = func(fns ...NewFn) []NewFn { return fns }

func TestNew(t *testing.T) {
	tests := []struct {
		name   string
		cfg    *logger.Config
		checks []NewFn
	}{
		{
			name: "nil config defaults to debug",
			checks: checkNew(
				func(t *testing.T, r Logger) {
					assert.NotNil(t, r)
				},
			),
		},
		{
			name: "empty config defaults to debug",
			checks: checkNew(
				func(t *testing.T, r Logger) {
					assert.NotNil(t, r)
				},
			),
		},
		{
			name: "info level config",
			cfg:  &logger.Config{Level: "info"},
			checks: checkNew(
				func(t *testing.T, r Logger) {
					assert.NotNil(t, r)
				},
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := New(tt.cfg)
			for _, c := range tt.checks {
				c(t, r)
			}
		})
	}
}

func TestLogger_NilPanic(t *testing.T) {
	l := New(nil)
	assert.NotPanics(t, func() { l.Debug("msg") })
	assert.NotPanics(t, func() { l.Info("msg") })
	assert.NotPanics(t, func() { l.Warn("msg") })
	assert.NotPanics(t, func() { l.Error("msg") })
}
