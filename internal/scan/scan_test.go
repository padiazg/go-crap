package scan

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"regexp"
	"sync"
	"testing"

	"github.com/padiazg/go-crap/internal/coverage"
	"github.com/padiazg/go-crap/internal/score"
	"github.com/padiazg/go-crap/pkg/logger"
	"github.com/stretchr/testify/assert"
)

// helpers

func checkScanError(want string) func(*testing.T, *Entries, error) {
	return func(t *testing.T, _ *Entries, err error) {
		t.Helper()
		if want == "" {
			assert.NoErrorf(t, err, "checkScanError: expected no error, got %v", err)
			return
		}
		if assert.Errorf(t, err, "checkScanError: expected error %q", want) {
			assert.Containsf(t, err.Error(), want, "checkScanError mismatch")
		}
	}
}

func checkLen(count int) func(*testing.T, *Entries, error) {
	return func(t *testing.T, r *Entries, err error) {
		t.Helper()
		assert.Len(t, r.List, count)
	}
}

func checkValue(pos int, crap float64, name string) func(*testing.T, *Entries, error) {
	return func(t *testing.T, r *Entries, err error) {
		t.Helper()
		if assert.Less(t, pos, len(r.List), "index %d out of bounds for list length %d", pos, len(r.List)) {
			assert.Equal(t, name, r.List[pos].FuncName, "func at index %d", pos)
			assert.Equal(t, crap, r.List[pos].CRAP, "crap at index %d", pos)
		}
	}
}

func checkSortedDesc() func(*testing.T, *Entries, error) {
	return func(t *testing.T, r *Entries, err error) {
		t.Helper()
		for i := 1; i < len(r.List); i++ {
			assert.GreaterOrEqual(t, r.List[i-1].EffectiveCRAP, r.List[i].EffectiveCRAP,
				"entries not sorted descending: [%d]=%.2f < [%d]=%.2f",
				i-1, r.List[i-1].EffectiveCRAP, i, r.List[i].EffectiveCRAP)
		}
	}
}

// TestScan — integration tests exercising the full Scan pipeline against
// internal/testdata (real go test + complexity parsing + merge).
func TestScan(t *testing.T) {
	tests := []struct {
		name    string
		options *Options
		checks  []func(*testing.T, *Entries, error)
	}{
		{
			name: "default_scan_excludes_test_files",
			options: &Options{
				Path:    "../testdata",
				Exclude: []string{".*_test.go"},
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError(""),
				checkLen(5),
				checkValue(0, 90.00, "veryComplex"),
				checkValue(4, 1.00, "simple"),
				checkSortedDesc(),
			},
		},
		{
			name: "non_existent_path_returns_error",
			options: &Options{
				Path: "/no/such/dir/that/does/not/exist",
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError("coverage scan"),
			},
		},
		{
			name: "min_50_filters_low_scores",
			options: &Options{
				Path: "../testdata",
				Min:  50,
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError(""),
				checkLen(1),
				checkValue(0, 90.00, "veryComplex"),
			},
		},
		{
			name: "min_higher_than_all_returns_empty",
			options: &Options{
				Path: "../testdata",
				Min:  100,
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError(""),
				checkLen(0),
			},
		},
		{
			name: "top_2_limits_results",
			options: &Options{
				Path: "../testdata",
				Top:  2,
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError(""),
				checkLen(2),
				checkValue(0, 90.00, "veryComplex"),
				checkSortedDesc(),
			},
		},
		{
			name: "top_larger_than_result_set_is_no_op",
			options: &Options{
				Path: "../testdata",
				Top:  100,
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError(""),
				checkLen(6),
			},
		},
		{
			name: "invalid_missing_policy_returns_error",
			options: &Options{
				Path:    "../testdata",
				Missing: "invalid",
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError("unknown missing policy"),
			},
		},
		{
			name: "exclude_function_name_reduces_count",
			options: &Options{
				Path:    "../testdata",
				Exclude: []string{"veryComplex"},
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError(""),
				checkLen(5),
				checkSortedDesc(),
			},
		},
		{
			name: "missing_optimistic_assumes_100_percent_coverage",
			options: &Options{
				Path:    "../testdata",
				Missing: "optimistic",
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError(""),
				checkLen(6),
			},
		},
		{
			name: "missing_pessimistic_default_policy",
			options: &Options{
				Path:    "../testdata",
				Missing: "pessimistic",
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError(""),
				checkLen(6),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r, err := Scan(tt.options)
			for _, c := range tt.checks {
				c(t, r, err)
			}
		})
	}
}

// Test_runCoverageAnalysis exercises the coverage scanner pipeline.
func Test_runCoverageAnalysis(t *testing.T) {
	tests := []struct {
		name    string
		options *Options
		exclude *regexp.Regexp
		checks  []func(*testing.T, []coverage.ModuleCoverage, error)
	}{
		{
			name: "valid_path_returns_coverage_data",
			options: &Options{
				Path: "../testdata",
			},
			exclude: nil,
			checks: []func(*testing.T, []coverage.ModuleCoverage, error){
				func(t *testing.T, r []coverage.ModuleCoverage, err error) {
					t.Helper()
					assert.NoError(t, err)
					assert.NotEmpty(t, r)
					for _, mc := range r {
						if mc.Error != nil {
							t.Errorf("module %s had error: %v", mc.Dir, mc.Error)
						}
					}
				},
			},
		},
		{
			name: "non_existent_path_returns_error",
			options: &Options{
				Path: "/no/such/dir/that/does/not/exist",
			},
			exclude: nil,
			checks: []func(*testing.T, []coverage.ModuleCoverage, error){
				func(t *testing.T, _ []coverage.ModuleCoverage, err error) {
					t.Helper()
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "coverage scan")
				},
			},
		},
		{
			name: "exclude_pattern_filters_coverage_functions",
			options: &Options{
				Path: "../testdata",
			},
			exclude: regexp.MustCompile(".*_test.go"),
			checks: []func(*testing.T, []coverage.ModuleCoverage, error){
				func(t *testing.T, r []coverage.ModuleCoverage, err error) {
					t.Helper()
					assert.NoError(t, err)
					assert.NotEmpty(t, r)
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r, err := runCoverageAnalysis(context.Background(), tt.options, tt.exclude)
			for _, c := range tt.checks {
				c(t, r, err)
			}
		})
	}
}

func Test_parseMissingPolicy(t *testing.T) {
	tests := []struct {
		name    string
		s       string
		want    score.MissingPolicy
		wantErr string
	}{
		{name: "pessimistic", s: "pessimistic", want: score.MissingPessimistic},
		{name: "optimistic", s: "optimistic", want: score.MissingOptimistic},
		{name: "skip", s: "skip", want: score.MissingSkip},
		{name: "empty_defaults_to_pessimistic", s: "", want: score.MissingPessimistic},
		{name: "case_insensitive_PESSIMISTIC", s: "PESSIMISTIC", want: score.MissingPessimistic},
		{name: "case_insensitive_Optimistic", s: "Optimistic", want: score.MissingOptimistic},
		{name: "invalid", s: "invalid", wantErr: "unknown missing policy"},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r, err := parseMissingPolicy(tt.s)
			assert.Equal(t, tt.want, r)
			if tt.wantErr != "" {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), tt.wantErr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func Test_effectiveCRAP(t *testing.T) {
	tests := []struct {
		name string
		e    score.CRAPEntry
		want float64
	}{
		{name: "effective_crap_returns_effective", e: score.CRAPEntry{EffectiveCRAP: 50, CRAP: 30}, want: 50},
		{name: "zero_effective_falls_back_to_crap", e: score.CRAPEntry{EffectiveCRAP: 0, CRAP: 30}, want: 30},
		{name: "both_zero", e: score.CRAPEntry{EffectiveCRAP: 0, CRAP: 0}, want: 0},
		{name: "negative_effective", e: score.CRAPEntry{EffectiveCRAP: -10, CRAP: 100}, want: -10},
		{name: "zero_crap_with_effective", e: score.CRAPEntry{EffectiveCRAP: 75, CRAP: 0}, want: 75},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r := effectiveCRAP(tt.e)
			assert.Equal(t, tt.want, r)
		})
	}
}

// from https://rednafi.com/go/capture_console_output/
func captureStdOut(f func()) string {
	// Create a pipe to capture stdout
	custReader, custWriter, err := os.Pipe()
	if err != nil {
		panic(err)
	}

	// Save the original stdout and stderr to restore later
	origStdout := os.Stdout
	origStderr := os.Stderr

	// Restore stdout and stderr when done
	defer func() {
		os.Stdout = origStdout
		os.Stderr = origStderr
	}()

	// Set the stdout and stderr to the pipe
	os.Stdout, os.Stderr = custWriter, custWriter
	log.SetOutput(custWriter)

	// Create a channel to read the output from the pipe

	out := make(chan string)

	// Use a goroutine to read from the pipe and send the output to the channel
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		wg.Done()
		var buff bytes.Buffer
		io.Copy(&buff, custReader)
		out <- buff.String()
	}()
	wg.Wait()

	// Call the function that writes to stdout
	f()

	// Close the writer to signal that we're done
	_ = custWriter.Close()

	// Wait for the goroutine to finish reading from the pipe
	return <-out
}

type logCoverageErrorsCheckFn func(*testing.T, string)

var checklogCoverageErrors = func(fns ...logCoverageErrorsCheckFn) []logCoverageErrorsCheckFn { return fns }

var checkContains = func(want string) logCoverageErrorsCheckFn {
	return func(t *testing.T, s string) {
		t.Helper()
		assert.Containsf(t, s, want, "output should contain %q", want)
	}
}

var checkNotContains = func(want string) logCoverageErrorsCheckFn {
	return func(t *testing.T, s string) {
		t.Helper()
		assert.NotContainsf(t, s, want, "output should not contain %q", want)
	}
}

var checkEmpty = func() logCoverageErrorsCheckFn {
	return func(t *testing.T, s string) {
		t.Helper()
		assert.Emptyf(t, s, "output should be empty")
	}
}

type captureLogger struct{}

func (captureLogger) Debug(msg string, args ...any) {
	fmt.Fprint(os.Stdout, msg, " ")
	for i := 0; i < len(args); i += 2 {
		if i+1 < len(args) {
			fmt.Fprintf(os.Stdout, "%s=%v ", args[i], args[i+1])
		}
	}
	fmt.Fprintln(os.Stdout)
}

func (captureLogger) Info(msg string, args ...any)  {}
func (captureLogger) Warn(msg string, args ...any)  {}
func (captureLogger) Error(msg string, args ...any) {}
func (captureLogger) Fatal(msg string, args ...any) {}

func Test_logCoverageErrors(t *testing.T) {
	tests := []struct {
		name      string
		l         logger.Logger
		coverages []coverage.ModuleCoverage
		checks    []logCoverageErrorsCheckFn
	}{
		{
			name: "nil_logger_no_output",
			l:    nil,
			coverages: []coverage.ModuleCoverage{
				{Dir: "/mod", Error: errors.New("fail")},
			},
			checks: checklogCoverageErrors(checkEmpty()),
		},
		{
			name:      "empty_coverages",
			l:         captureLogger{},
			coverages: []coverage.ModuleCoverage{},
			checks:    checklogCoverageErrors(checkEmpty()),
		},
		{
			name: "all_ok_no_errors",
			l:    captureLogger{},
			coverages: []coverage.ModuleCoverage{
				{Dir: "/mod", Error: nil},
			},
			checks: checklogCoverageErrors(checkEmpty()),
		},
		{
			name: "single_error_logged",
			l:    captureLogger{},
			coverages: []coverage.ModuleCoverage{
				{Dir: "/mod", Error: errors.New("boom")},
			},
			checks: checklogCoverageErrors(
				checkContains("coverage scan error"),
				checkContains("/mod"),
				checkContains("boom"),
			),
		},
		{
			name: "multiple_errors_logged",
			l:    captureLogger{},
			coverages: []coverage.ModuleCoverage{
				{Dir: "/a", Error: errors.New("err1")},
				{Dir: "/b", Error: errors.New("err2")},
			},
			checks: checklogCoverageErrors(
				checkContains("err1"),
				checkContains("err2"),
				checkContains("/a"),
				checkContains("/b"),
			),
		},
		{
			name: "mixed_errors_and_successes",
			l:    captureLogger{},
			coverages: []coverage.ModuleCoverage{
				{Dir: "/ok", Error: nil},
				{Dir: "/fail", Error: errors.New("fail")},
			},
			checks: checklogCoverageErrors(
				checkContains("/fail"),
				checkContains("fail"),
				checkNotContains("/ok"),
			),
		},
		{
			name: "empty_error_string_still_logged",
			l:    captureLogger{},
			coverages: []coverage.ModuleCoverage{
				{Dir: "/m", Error: errors.New("")},
			},
			checks: checklogCoverageErrors(
				checkContains("coverage scan error"),
				checkContains("/m"),
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := captureStdOut(func() {
				logCoverageErrors(tt.l, tt.coverages)
			})

			for _, c := range tt.checks {
				c(t, got)
			}
		})
	}
}
