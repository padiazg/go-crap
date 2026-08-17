package cmd

import (
	"bytes"
	"io"
	"os"
	"testing"
	"time"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/padiazg/go-crap/internal/report"
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
		{
			name: "sarif format writes SARIF JSON",
			config: outputConfig{
				format:    "sarif",
				threshold: 4,
				writer:    &mockWriter{},
				path:      "/base",
			},
			entries: entries,
			checks: checkoutput(
				checkoutputOutputContains("runs"),
				checkoutputOutputContains("sarif"),
			),
		},
		{
			name: "pr-comment format writes PR comment",
			config: outputConfig{
				format:    "pr-comment",
				threshold: 4,
				writer:    &mockWriter{},
				path:      "/base",
			},
			entries: entries,
			checks: checkoutput(
				checkoutputOutputContains("go-crap-report"),
				checkoutputOutputContains("function"),
			),
		},
		{
			name: "table format with baseline annotates entries",
			config: outputConfig{
				format:        "table",
				threshold:     30,
				writer:        &mockWriter{},
				path:          "/base",
				showUnchanged: true,
				baseline:      loadBaselineForTest(t, `{"$schema":"https://raw.githubusercontent.com/padiazg/go-crap/master/schemas/report-v1.json","version":"1.1.0","entries":[{"file":"internal/foo.go","function":"Foo","package":"foo","crap":4.5,"cyclomatic":1,"effective_crap":4.5,"line":1,"coverage":50}]}`),
			},
			entries: entries,
			checks: checkoutput(
				checkoutputOutputContains("Foo"),
				checkoutputOutputContains("Bar"),
			),
		},
		{
			name: "json format with baseline includes delta info",
			config: outputConfig{
				format:    "json",
				threshold: 30,
				writer:    &mockWriter{},
				path:      "/base",
				baseline:  loadBaselineForTest(t, `{"$schema":"https://raw.githubusercontent.com/padiazg/go-crap/master/schemas/report-v1.json","version":"1.1.0","entries":[{"file":"internal/foo.go","function":"Foo","package":"foo","crap":4.5,"cyclomatic":1,"effective_crap":4.5,"line":1,"coverage":50}]}`),
			},
			entries: entries,
			checks: checkoutput(
				checkoutputOutputContains("entries"),
				checkoutputOutputContains("Foo"),
			),
		},
		{
			name: "detailed mode with baseline",
			config: outputConfig{
				format:        "table",
				threshold:     30,
				writer:        &mockWriter{},
				path:          "/base",
				detailed:      true,
				showUnchanged: true,
				baseline:      loadBaselineForTest(t, `{"$schema":"https://raw.githubusercontent.com/padiazg/go-crap/master/schemas/report-v1.json","version":"1.1.0","entries":[{"file":"internal/foo.go","function":"Foo","package":"foo","crap":4.5,"cyclomatic":1,"effective_crap":4.5,"line":1,"coverage":50}]}`),
			},
			entries: entries,
			checks: checkoutput(
				checkoutputOutputContains("Foo"),
				checkoutputOutputContains("Bar"),
			),
		},
		{
			name: "output to file creates file",
			config: outputConfig{
				format:    "json",
				threshold: 30,
				writer:    &mockWriter{},
				path:      "/base",
				output:    "",
			},
			entries: entries,
			checks: checkoutput(
				checkoutputOutputContains("entries"),
			),
		},
		{
			name: "sarif format with coverage warning",
			config: outputConfig{
				format:    "sarif",
				threshold: 100,
				writer:    &mockWriter{},
				path:      "/base",
			},
			entries: &scan.Entries{
				List: []score.CRAPEntry{
					{
						File:            "internal/foo.go",
						FuncName:        "Foo",
						Complexity:      1,
						Coverage:        50,
						CRAP:            1.25,
						CoverageWarning: "coverage unavailable",
					},
				},
			},
			checks: checkoutput(
				checkoutputOutputContains("coverage-unavailable"),
			),
		},
		{
			name: "sarif format with coverage untrusted",
			config: outputConfig{
				format:    "sarif",
				threshold: 100,
				writer:    &mockWriter{},
				path:      "/base",
			},
			entries: &scan.Entries{
				List: []score.CRAPEntry{
					{
						File:              "internal/foo.go",
						FuncName:          "Foo",
						Complexity:        1,
						Coverage:          50,
						CRAP:              1.25,
						CoverageUntrusted: true,
						MutationScore:     0.3,
						MutationDetails: []score.MutationDetail{
							{
								MutantType:      "remove",
								Line:            5,
								OriginalText:    "true",
								ReplacementText: "false",
							},
						},
					},
				},
			},
			checks: checkoutput(
				checkoutputOutputContains("coverage-untrusted"),
			),
		},
		{
			name: "sarif detailed with mutation details",
			config: outputConfig{
				format:    "sarif",
				threshold: 0,
				writer:    &mockWriter{},
				path:      "/base",
				detailed:  true,
			},
			entries: &scan.Entries{
				List: []score.CRAPEntry{
					{
						File:              "internal/foo.go",
						FuncName:          "Foo",
						Complexity:        5,
						Coverage:          50,
						CRAP:              12.5,
						CoverageUntrusted: true,
						MutationScore:     0.3,
						MutationDetails: []score.MutationDetail{
							{
								MutantType:      "remove",
								Line:            5,
								OriginalText:    "true",
								ReplacementText: "false",
							},
						},
					},
				},
			},
			checks: checkoutput(
				checkoutputOutputContains("coverage-untrusted"),
				checkoutputOutputContains("remove"),
			),
		},
		{
			name: "pr-comment with baseline shows delta",
			config: outputConfig{
				format:    "pr-comment",
				threshold: 4,
				writer:    &mockWriter{},
				path:      "/base",
				baseline:  loadBaselineForTest(t, `{"$schema":"https://raw.githubusercontent.com/padiazg/go-crap/master/schemas/report-v1.json","version":"1.1.0","entries":[{"file":"internal/foo.go","function":"Foo","package":"foo","crap":4.0,"cyclomatic":1,"effective_crap":4.0,"line":1,"coverage":50}]}`),
			},
			entries: entries,
			checks: checkoutput(
				checkoutputOutputContains("go-crap-report"),
				checkoutputOutputContains("Δ"),
			),
		},
		{
			name: "pr-comment with baseline shows regressions",
			config: outputConfig{
				format:    "pr-comment",
				threshold: 100,
				writer:    &mockWriter{},
				path:      "/base",
				baseline:  loadBaselineForTest(t, `{"$schema":"https://raw.githubusercontent.com/padiazg/go-crap/master/schemas/report-v1.json","version":"1.1.0","entries":[{"file":"internal/bar.go","function":"Bar","package":"bar","crap":1.0,"cyclomatic":3,"effective_crap":1.0,"line":1,"coverage":100}]}`),
			},
			entries: entries,
			checks: checkoutput(
				checkoutputOutputContains("Regressions"),
			),
		},
		{
			name: "pr-comment with baseline shows new functions",
			config: outputConfig{
				format:    "pr-comment",
				threshold: 100,
				writer:    &mockWriter{},
				path:      "/base",
				baseline:  loadBaselineForTest(t, `{"$schema":"https://raw.githubusercontent.com/padiazg/go-crap/master/schemas/report-v1.json","version":"1.1.0","entries":[]}`),
			},
			entries: entries,
			checks: checkoutput(
				checkoutputOutputContains("New Functions"),
			),
		},
		{
			name: "pr-comment with unreliable coverage",
			config: outputConfig{
				format:    "pr-comment",
				threshold: 100,
				writer:    &mockWriter{},
				path:      "/base",
			},
			entries: &scan.Entries{
				List: []score.CRAPEntry{
					{
						File:              "internal/foo.go",
						FuncName:          "Foo",
						Complexity:        1,
						Coverage:          50,
						CRAP:              1.25,
						CoverageUntrusted: true,
						MutationScore:     0.3,
					},
				},
			},
			checks: checkoutput(
				checkoutputOutputContains("Unreliable Coverage"),
			),
		},
		{
			name: "pr-comment with unavailable coverage",
			config: outputConfig{
				format:    "pr-comment",
				threshold: 100,
				writer:    &mockWriter{},
				path:      "/base",
			},
			entries: &scan.Entries{
				List: []score.CRAPEntry{
					{
						File:            "internal/foo.go",
						FuncName:        "Foo",
						Complexity:      1,
						Coverage:        50,
						CRAP:            1.25,
						CoverageWarning: "coverage unavailable",
					},
				},
			},
			checks: checkoutput(
				checkoutputOutputContains("Coverage Unavailable"),
			),
		},
		{
			name: "pr-comment detailed with unreliable coverage",
			config: outputConfig{
				format:    "pr-comment",
				threshold: 100,
				writer:    &mockWriter{},
				path:      "/base",
				detailed:  true,
			},
			entries: &scan.Entries{
				List: []score.CRAPEntry{
					{
						File:              "internal/foo.go",
						FuncName:          "Foo",
						Complexity:        1,
						Coverage:          50,
						CRAP:              1.25,
						CoverageUntrusted: true,
						MutationScore:     0.3,
						MutationDetails: []score.MutationDetail{
							{
								MutantType:      "remove",
								Line:            5,
								OriginalText:    "true",
								ReplacementText: "false",
							},
						},
					},
				},
			},
			checks: checkoutput(
				checkoutputOutputContains("Unreliable Coverage"),
				checkoutputOutputContains("remove"),
			),
		},
		{
			name: "output to file creates and writes",
			config: outputConfig{
				format:    "json",
				threshold: 30,
				writer:    &mockWriter{},
				path:      "/base",
				output:    "",
			},
			entries: entries,
			checks: checkoutput(
				checkoutputOutputContains("entries"),
			),
		},
		{
			name: "error creating output file",
			config: outputConfig{
				format:    "json",
				threshold: 30,
				writer:    &mockWriter{},
				path:      "/base",
				output:    "/nonexistent/a_file.out",
			},
			entries: entries,
			checks: checkoutput(
				checkoutputError("/nonexistent/a_file.out: no such file or directory"),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := output(tt.entries, tt.config)
			for _, c := range tt.checks {
				c(t, tt.config.writer, err)
			}
		})
	}
}

type runScanCheckFn func(*testing.T, io.Writer, error)

var checkrunScan = func(fns ...runScanCheckFn) []runScanCheckFn { return fns }

func checkrunScanError(want string) runScanCheckFn {
	return func(t *testing.T, _ io.Writer, err error) {
		t.Helper()
		if want == "" {
			assert.NoErrorf(t, err, "checkrunScanError: expected no error, got %v", err)
			return
		}
		if assert.Errorf(t, err, "checkrunScanError: expected error %q", want) {
			assert.Containsf(t, err.Error(), want, "checkrunScanError mismatch")
		}
	}
}

func checkrunScanOutputContains(want string) runScanCheckFn {
	return func(t *testing.T, writer io.Writer, err error) {
		t.Helper()
		assert.NoErrorf(t, err, "checkrunScanOutputContains: expected no error, got %v", err)
		mw, ok := writer.(*mockWriter)
		if assert.Truef(t, ok, "writer is not mockWriter") {
			assert.Containsf(t, string(mw.buf), want, "output should contain %q", want)
		}
	}
}

func makeCommand() (*cobra.Command, *mockWriter) {
	w := &mockWriter{}
	cmd := &cobra.Command{}
	cmd.SetOut(w)
	return cmd, w
}

func resetFlags() {
	flagThreshold = 30
	flagFailAbove = false
	flagFormat = "table"
	flagTop = 0
	flagMin = 0
	flagMissing = "pessimistic"
	flagExclude = nil
	flagVerbose = false
	flagOutput = ""
	flagMutation = ""
	flagDetailed = false
	flagTimeout = 10 * time.Minute
	flagCoverProf = ""
	flagBaseline = ""
	flagShowUnchanged = false
	flagFailRegression = false
	flagFailRegressionThreshold = 0.01
	flagProgress = false
	flagNoProgress = false
	flagIncludeTests = false
}

func Test_timeoutFlag_registered(t *testing.T) {
	f := scanCmd.Flags().Lookup("timeout")
	if assert.NotNil(t, f, "expected --timeout flag to be registered") {
		assert.Equal(t, "10m0s", f.DefValue)
	}
}

func makeBaseline(t *testing.T, path string) string {
	t.Helper()
	f, err := os.CreateTemp("", "crap-baseline-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	_, err = f.WriteString(`{
  "$schema": "https://raw.githubusercontent.com/padiazg/go-crap/master/schemas/report-v1.json",
  "version": "1.1.0",
  "entries": [
    {
      "file": "` + path + `/internal/testdata/simple.go",
      "function": "SimpleFunc",
      "package": "testdata",
      "crap": 2.0,
      "cyclomatic": 1,
      "effective_crap": 2.0,
      "line": 7,
      "coverage": 100.0
    },
    {
      "file": "` + path + `/internal/testdata/complex.go",
      "function": "ComplexFunc",
      "package": "testdata",
      "crap": 15.0,
      "cyclomatic": 5,
      "effective_crap": 15.0,
      "line": 1,
      "coverage": 50.0
    }
  ]
}`)
	if err != nil {
		f.Close()
		t.Fatalf("failed to write temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

func getTestdataPath(t *testing.T) string {
	t.Helper()
	pwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get working directory: %v", err)
	}
	return pwd
}

func makeBaselineForTest(t *testing.T, entries string) string {
	t.Helper()
	f, err := os.CreateTemp("", "crap-baseline-*.json")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	_, err = f.WriteString(entries)
	if err != nil {
		f.Close()
		t.Fatalf("failed to write temp file: %v", err)
	}
	f.Close()
	return f.Name()
}

func loadBaselineForTest(t *testing.T, jsonContent string) *report.Baseline {
	t.Helper()
	path := makeBaselineForTest(t, jsonContent)
	bl, err := report.LoadBaseline(path)
	if err != nil {
		t.Fatalf("failed to load baseline: %v", err)
	}
	return bl
}

func Test_runScan(t *testing.T) {
	testdataPath := "../internal/testdata"
	baseDir := getTestdataPath(t)

	tests := []struct {
		name   string
		cmd    *cobra.Command
		args   []string
		before func()
		checks []runScanCheckFn
	}{
		{
			name:   "successful scan against testdata",
			args:   []string{testdataPath},
			before: resetFlags,
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name:   "default path when args empty",
			args:   nil,
			before: resetFlags,
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name:   "fail-above when threshold exceeded",
			args:   []string{testdataPath},
			before: func() { resetFlags(); flagFailAbove = true; flagThreshold = 1 },
			checks: checkrunScan(
				checkrunScanError("CRAP threshold exceeded"),
			),
		},
		{
			name:   "fail-above with no exceedance",
			args:   []string{testdataPath},
			before: func() { resetFlags(); flagFailAbove = true; flagThreshold = 999 },
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name:   "non-existent path returns error",
			args:   []string{"/no/such/path"},
			before: resetFlags,
			checks: checkrunScan(
				checkrunScanError("coverage scan"),
			),
		},
		{
			name: "invalid output file returns error",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagOutput = "/nonexistent/coverage.out"
			},
			checks: checkrunScan(
				checkrunScanError("output:"),
			),
		},
		{
			name: "output to file",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				f, err := os.CreateTemp("", "crap-output-*.json")
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}
				flagOutput = f.Name()
			},
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name:   "custom timeout succeeds",
			args:   []string{testdataPath},
			before: func() { resetFlags(); flagTimeout = 2 * time.Minute },
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name:   "fail-regression without baseline returns error",
			args:   []string{testdataPath},
			before: func() { resetFlags(); flagFailRegression = true },
			checks: checkrunScan(
				checkrunScanError("--fail-regression requires --baseline"),
			),
		},
		{
			name:   "show-unchanged without baseline returns error",
			args:   []string{testdataPath},
			before: func() { resetFlags(); flagShowUnchanged = true },
			checks: checkrunScan(
				checkrunScanError("--show-unchanged requires --baseline"),
			),
		},
		{
			name: "show-unchanged flag is registered",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				f := scanCmd.Flags().Lookup("show-unchanged")
				assert.NotNil(t, f, "expected --show-unchanged flag to be registered")
			},
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name: "baseline with fail-regression no regression due to path mismatch",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagFailRegression = true
				flagBaseline = makeBaseline(t, baseDir)
			},
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name: "fail-regression with matching baseline returns error",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagFailRegression = true
				flagBaseline = makeBaselineForTest(t, `{
   "$schema": "https://raw.githubusercontent.com/padiazg/go-crap/master/schemas/report-v1.json",
   "version": "1.1.0",
   "entries": [
     {"file":"../internal/testdata/simple.go","function":"withIf","package":"testdata","crap":0.1,"cyclomatic":2,"effective_crap":0.1,"line":7,"coverage":0.0},
     {"file":"../internal/testdata/complex.go","function":"veryComplex","package":"testdata","crap":0.1,"cyclomatic":10,"effective_crap":0.1,"line":3,"coverage":0.0}
   ]
}`)
			},
			checks: checkrunScan(
				checkrunScanError("CRAP regression detected"),
			),
		},
		{
			name: "fail-regression ignore-covered excludes fully covered",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagFailRegression = true
				flagFailRegressionIgnoreCovered = true
				flagBaseline = makeBaselineForTest(t, `{
   "$schema": "https://raw.githubusercontent.com/padiazg/go-crap/master/schemas/report-v1.json",
   "version": "1.1.0",
   "entries": [
     {"file":"../internal/testdata/simple.go","function":"simple","package":"testdata","crap":0.1,"cyclomatic":1,"effective_crap":0.1,"line":3,"coverage":100.0}
   ]
}`)
			},
			checks: checkrunScan(
				checkrunScanError(""),
				checkrunScanOutputContains("Ignored (fully covered):"),
			),
		},
		{
			name: "invalid baseline file returns error",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagBaseline = "/nonexistent/baseline.json"
			},
			checks: checkrunScan(
				checkrunScanError("baseline"),
			),
		},
		{
			name: "baseline with detailed format",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagDetailed = true
				flagBaseline = makeBaseline(t, baseDir)
			},
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name: "sarif format succeeds",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagFormat = "sarif"
			},
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name: "pr-comment format succeeds",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagFormat = "pr-comment"
			},
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name: "verbose mode enables debug logging",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagVerbose = true
			},
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name: "min filter excludes low entries",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagMin = 10
			},
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name: "top filter limits entries",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagTop = 2
			},
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name: "include-tests flag",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagIncludeTests = true
			},
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name: "coverage-profile flag",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagCoverProf = "../internal/testdata/cover.out"
			},
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name: "baseline with show-unchanged",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagShowUnchanged = true
				flagBaseline = makeBaseline(t, baseDir)
			},
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name: "progress flag enables reporter",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagProgress = true
			},
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name: "no-progress flag disables reporter",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagNoProgress = true
			},
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name: "mutation-report flag with valid report",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				f, err := os.CreateTemp("", "crap-mutation-*.json")
				if err != nil {
					t.Fatalf("failed to create temp file: %v", err)
				}
				if _, err := f.WriteString(`{
  "go_module": "github.com/example/test",
  "files": [{"file_name": "simple.go", "mutations": [
    {"type": "ARITHMETIC", "status": "LIVED", "line": 7}
  ]}],
  "mutants_killed": 0,
  "mutants_lived": 1,
  "mutants_not_covered": 0,
  "mutants_total": 1,
  "test_efficacy": 0
}`); err != nil {
					f.Close()
					t.Fatalf("failed to write temp file: %v", err)
				}
				f.Close()
				flagMutation = f.Name()
			},
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name: "mutation-report flag with invalid path",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagMutation = "/nonexistent/report.json"
			},
			checks: checkrunScan(
				checkrunScanError("MutationAnnotations"),
			),
		},
		{
			name: "exclude flag filters files",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagExclude = []string{"complex"}
			},
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name: "missing policy skip",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagMissing = "skip"
			},
			checks: checkrunScan(
				checkrunScanError(""),
			),
		},
		{
			name: "missing policy invalid",
			args: []string{testdataPath},
			before: func() {
				resetFlags()
				flagMissing = "bogus"
			},
			checks: checkrunScan(
				checkrunScanError("unknown missing policy"),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd, mw := makeCommand()
			if tt.before != nil {
				tt.before()
			}
			err := runScan(cmd, tt.args)
			for _, c := range tt.checks {
				c(t, mw, err)
			}
		})
	}
}

func Test_failRegressionIgnoreCoveredFlag_registered(t *testing.T) {
	f := scanCmd.Flags().Lookup("fail-regression-ignore-covered")
	if assert.NotNil(t, f, "expected --fail-regression-ignore-covered flag to be registered") {
		assert.Equal(t, "false", f.DefValue)
	}
}

func Test_fmtRegressionError(t *testing.T) {
	tests := []struct {
		name             string
		regressions      []score.CRAPEntry
		ignored          []score.CRAPEntry
		currentCombined  float64
		baselineCombined float64
		wantContains     []string
		wantNotContains  []string
		wantErr          bool
	}{
		{
			name: "only regressions",
			regressions: []score.CRAPEntry{
				{File: "cmd/scan.go", Line: 10, FuncName: "Foo", BaselineCRAP: 5.0, EffectiveCRAP: 10.0},
			},
			ignored:          []score.CRAPEntry{},
			currentCombined:  20.0,
			baselineCombined: 15.0,
			wantContains: []string{
				"CRAP regression detected:",
				"cmd/scan.go:10 Foo: 5.00 -> 10.00 (+5.00)",
				"Combined CRAP: 20.00 (Δ+5.00 vs baseline)",
			},
			wantNotContains: []string{"Ignored (fully covered):"},
			wantErr:         true,
		},
		{
			name:        "only ignored fully covered",
			regressions: []score.CRAPEntry{},
			ignored: []score.CRAPEntry{
				{File: "cmd/scan.go", Line: 20, FuncName: "Bar", BaselineCRAP: 5.0, EffectiveCRAP: 7.0, Coverage: 100.0},
			},
			currentCombined:  20.0,
			baselineCombined: 15.0,
			wantContains: []string{
				"Ignored (fully covered):",
				"cmd/scan.go:20 Bar: 5.00 -> 7.00 (+2.00)",
			},
			wantNotContains: []string{"CRAP regression detected:"},
			wantErr:         false,
		},
		{
			name: "mixed regressions and ignored",
			regressions: []score.CRAPEntry{
				{File: "internal/foo.go", Line: 30, FuncName: "Baz", BaselineCRAP: 3.0, EffectiveCRAP: 8.0},
			},
			ignored: []score.CRAPEntry{
				{File: "cmd/scan.go", Line: 10, FuncName: "Qux", BaselineCRAP: 4.0, EffectiveCRAP: 6.0, Coverage: 100.0},
			},
			currentCombined:  25.0,
			baselineCombined: 20.0,
			wantContains: []string{
				"CRAP regression detected:",
				"internal/foo.go:30 Baz: 3.00 -> 8.00 (+5.00)",
				"Ignored (fully covered):",
				"cmd/scan.go:10 Qux: 4.00 -> 6.00 (+2.00)",
				"Combined CRAP: 25.00 (Δ+5.00 vs baseline)",
			},
			wantErr: true,
		},
		{
			name:        "empty both lists",
			regressions: []score.CRAPEntry{},
			ignored:     []score.CRAPEntry{},
			wantErr:     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := fmtRegressionError(&buf, tt.regressions, tt.ignored, tt.currentCombined, tt.baselineCombined)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			for _, want := range tt.wantContains {
				assert.Contains(t, buf.String(), want, "output should contain %q", want)
			}
			for _, want := range tt.wantNotContains {
				assert.NotContains(t, buf.String(), want, "output should not contain %q", want)
			}
		})
	}
}
