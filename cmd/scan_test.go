package cmd

import (
	"io"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/padiazg/go-crap/internal/scan"
	"github.com/padiazg/go-crap/internal/score"
)

type mockWriter struct {
	buf []byte
}

func (w *mockWriter) Write(p []byte) (n int, err error) {
	w.buf = append(w.buf, p...)
	return len(p), nil
}

func checkoutputOutputContains(want string) outputCheckFn {
	return func(t *testing.T, writer io.Writer, err error) {
		t.Helper()
		assert.NoErrorf(t, err, "expected no error, got %v", err)
		mw, ok := writer.(*mockWriter)
		assert.Truef(t, ok, "writer is not mockWriter")
		assert.Containsf(t, string(mw.buf), want, "output should contain %q", want)
	}
}

type outputCheckFn func(*testing.T, io.Writer, error)

var checkoutput = func(fns ...outputCheckFn) []outputCheckFn { return fns }

func checkoutputError(want string) outputCheckFn {
	return func(t *testing.T, _ io.Writer, err error) {
		t.Helper()
		if want == "" {
			assert.NoErrorf(t, err, "checkoutputError: expected no error, got %v", err)
			return
		}
		if assert.Errorf(t, err, "checkoutputError: expected error %q", want) {
			assert.Containsf(t, err.Error(), want, "checkoutputError mismatch")
		}
	}
}
func Test_output(t *testing.T) {
	entries := &scan.Entries{
		List: []score.CRAPEntry{
			{
				File:       "internal/foo.go",
				FuncName:   "Foo",
				Complexity: 1,
				Coverage:   50,
				CRAP:       4.5,
			},
			{
				File:       "internal/bar.go",
				FuncName:   "Bar",
				Complexity: 3,
				Coverage:   100,
				CRAP:       3,
			},
		},
	}

	tests := []struct {
		name    string
		config  outputConfig
		entries *scan.Entries
		checks  []outputCheckFn
	}{
		{
			name: "table format writes entries",
			config: outputConfig{
				format:    "table",
				threshold: 30,
				writer:    &mockWriter{},
				path:      "/base",
			},
			entries: entries,
			checks: checkoutput(
				checkoutputOutputContains("Foo"),
				checkoutputOutputContains("Bar"),
			),
		},
		{
			name: "json format writes JSON entries",
			config: outputConfig{
				format:    "json",
				threshold: 30,
				writer:    &mockWriter{},
				path:      "/base",
			},
			entries: entries,
			checks: checkoutput(
				checkoutputOutputContains("entries"),
				checkoutputOutputContains("Foo"),
				checkoutputOutputContains("Bar"),
			),
		},
		{
			name: "github format writes annotations",
			config: outputConfig{
				format:    "github",
				threshold: 4,
				writer:    &mockWriter{},
				path:      "/base",
			},
			entries: entries,
			checks: checkoutput(
				checkoutputOutputContains("::warning"),
			),
		},
		{
			name: "unknown format returns error",
			config: outputConfig{
				format:    "xml",
				threshold: 30,
				writer:    &mockWriter{},
				path:      "/base",
			},
			entries: entries,
			checks:  checkoutput(checkoutputError("unknown format")),
		},
		{
			name: "nil entries returns error",
			config: outputConfig{
				format:    "table",
				threshold: 30,
				writer:    &mockWriter{},
				path:      "/base",
			},
			entries: nil,
			checks:  checkoutput(checkoutputError("entries list shouldn't be nil")),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := output(tt.entries, tt.config)
			for _, c := range tt.checks {
				c(t, tt.config.writer, err)
			}
		})
	}
}
