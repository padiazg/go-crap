package scan

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"sync"
	"testing"
	"time"

	"github.com/padiazg/go-crap/internal/complexity"
	"github.com/padiazg/go-crap/internal/coverage"
	"github.com/padiazg/go-crap/internal/packages"
	"github.com/padiazg/go-crap/internal/score"
	"github.com/padiazg/go-crap/pkg/logger"
	"github.com/padiazg/go-crap/pkg/utils"
	"github.com/stretchr/testify/assert"
	"go.uber.org/goleak"
)

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}

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
func TestBuildExcludePatterns(t *testing.T) {
	t.Run("includeTests_true_passes_user_patterns", func(t *testing.T) {
		got := buildExcludePatterns([]string{`pb/.*`}, true)
		assert.Equal(t, []string{`pb/.*`}, got)
	})

	t.Run("includeTests_false_prepends_default", func(t *testing.T) {
		got := buildExcludePatterns(nil, false)
		assert.Equal(t, []string{utils.DefaultExcludePattern}, got)
	})

	t.Run("includeTests_false_composes_user_patterns", func(t *testing.T) {
		got := buildExcludePatterns([]string{`pb/.*`, `mock_`}, false)
		assert.Equal(t, []string{utils.DefaultExcludePattern, `pb/.*`, `mock_`}, got)
	})

	t.Run("includeTests_false_with_nil_user", func(t *testing.T) {
		got := buildExcludePatterns(nil, false)
		assert.Equal(t, []string{utils.DefaultExcludePattern}, got)
	})
}

func TestScan(t *testing.T) {
	tests := []struct {
		name    string
		options *Options
		checks  []func(*testing.T, *Entries, error)
	}{
		{
			name: "default_scan_excludes_test_files",
			options: &Options{
				Path:         "../testdata",
				IncludeTests: true,
				Exclude:      []string{".*_test.go"},
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
			name: "default_scan_excludes_test_files_without_explicit_exclude",
			options: &Options{
				Path:         "../testdata",
				IncludeTests: false,
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
			name: "include_tests_flag_includes_test_files",
			options: &Options{
				Path:         "../testdata",
				IncludeTests: true,
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError(""),
				checkLen(6),
				checkValue(0, 90.00, "veryComplex"),
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
				Path:         "../testdata",
				IncludeTests: true,
				Min:          50,
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
				Path:         "../testdata",
				IncludeTests: true,
				Min:          100,
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError(""),
				checkLen(0),
			},
		},
		{
			name: "top_2_limits_results",
			options: &Options{
				Path:         "../testdata",
				IncludeTests: true,
				Top:          2,
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
				Path:         "../testdata",
				IncludeTests: true,
				Top:          100,
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError(""),
				checkLen(6),
			},
		},
		{
			name: "invalid_missing_policy_returns_error",
			options: &Options{
				Path:         "../testdata",
				IncludeTests: true,
				Missing:      "invalid",
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError("unknown missing policy"),
			},
		},
		{
			name: "exclude_function_name_reduces_count",
			options: &Options{
				Path:         "../testdata",
				IncludeTests: true,
				Exclude:      []string{"veryComplex"},
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
				Path:         "../testdata",
				IncludeTests: true,
				Missing:      "optimistic",
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError(""),
				checkLen(6),
			},
		},
		{
			name: "missing_pessimistic_default_policy",
			options: &Options{
				Path:         "../testdata",
				IncludeTests: true,
				Missing:      "pessimistic",
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError(""),
				checkLen(6),
			},
		},
		{
			name: "zero_timeout_uses_default",
			options: &Options{
				Path:         "../testdata",
				IncludeTests: true,
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError(""),
				checkLen(6),
			},
		},
		{
			name: "missing_skip_policy",
			options: &Options{
				Path:         "../testdata",
				IncludeTests: true,
				Missing:      "skip",
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError(""),
				checkLen(6),
				checkSortedDesc(),
			},
		},
		{
			name: "coverage_profile_option",
			options: &Options{
				Path:            "../testdata",
				CoverageProfile: "../testdata/cover.out",
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError(""),
				checkLen(5),
				checkValue(0, 90.00, "veryComplex"),
				checkSortedDesc(),
			},
		},
		{
			name: "exclude_multiple_patterns",
			options: &Options{
				Path:         "../testdata",
				IncludeTests: true,
				Exclude:      []string{"veryComplex", "withSwitch"},
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError(""),
				checkLen(4),
				checkSortedDesc(),
			},
		},
		{
			name: "timeout_explicitly_set",
			options: &Options{
				Path:         "../testdata",
				IncludeTests: true,
				Timeout:      30 * time.Second,
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError(""),
				checkLen(6),
				checkValue(0, 90.00, "veryComplex"),
				checkSortedDesc(),
			},
		},
		{
			name: "incorrect_exclude_pattern",
			options: &Options{
				Path:         "../testdata",
				IncludeTests: true,
				Timeout:      30 * time.Second,
				Exclude:      []string{"["},
			},
			checks: []func(*testing.T, *Entries, error){
				checkScanError("regexp: missing closing ]"),
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

func TestScan_fallbackToDotSlashSlash(t *testing.T) {
	dir, err := os.MkdirTemp("", "scan-dot-slash")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })

	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origCwd) })

	for _, f := range []string{"complex.go", "cover.out", "simple.go", "simple_test.go", "go.mod"} {
		src, err := os.ReadFile(filepath.Join("../testdata", f))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, f), src, 0644); err != nil {
			t.Fatal(err)
		}
	}

	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	// Neither Patterns nor Path is set → line 78 triggers: patterns = ["./..."]
	r, err := Scan(&Options{
		IncludeTests: true,
	})
	if err != nil {
		t.Fatalf("Scan with empty options failed (line 78 fallback): %v", err)
	}
	if len(r.List) == 0 {
		t.Fatal("expected non-empty result from ./... fallback")
	}
}

func Test_resolveTimeout(t *testing.T) {
	tests := []struct {
		name string
		in   time.Duration
		want time.Duration
	}{
		{name: "zero_uses_default", in: 0, want: DefaultTimeout},
		{name: "explicit_timeout_is_preserved", in: 5 * time.Second, want: 5 * time.Second},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, resolveTimeout(tt.in))
		})
	}
}

// Test_runCoverageAnalysis exercises the coverage scanner pipeline.
func Test_runCoverageAnalysis(t *testing.T) {
	tests := []struct {
		name     string
		options  *Options
		patterns []string
		exclude  *regexp.Regexp
		checks   []func(*testing.T, []coverage.ModuleCoverage, error)
	}{
		{
			name: "valid_path_returns_coverage_data",
			options: &Options{
				Path: "../testdata",
			},
			patterns: nil,
			exclude:  nil,
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
			patterns: nil,
			exclude:  nil,
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
			patterns: nil,
			exclude:  regexp.MustCompile(".*_test.go"),
			checks: []func(*testing.T, []coverage.ModuleCoverage, error){
				func(t *testing.T, r []coverage.ModuleCoverage, err error) {
					t.Helper()
					assert.NoError(t, err)
					assert.NotEmpty(t, r)
					for _, mc := range r {
						for _, fn := range mc.Functions {
							assert.NotContains(t, fn.File, "_test.go", "exclude pattern should filter test files")
						}
					}
				},
			},
		},
		{
			name: "profile_based_scan",
			options: &Options{
				Path:            "../testdata",
				CoverageProfile: "../testdata/cover.out",
			},
			patterns: nil,
			exclude:  nil,
			checks: []func(*testing.T, []coverage.ModuleCoverage, error){
				func(t *testing.T, r []coverage.ModuleCoverage, err error) {
					t.Helper()
					assert.NoError(t, err)
					assert.NotEmpty(t, r)
					for _, mc := range r {
						if mc.Error != nil {
							t.Errorf("module %s had error: %v", mc.Dir, mc.Error)
						}
						assert.Equal(t, "../testdata/cover.out", mc.Profile)
					}
				},
			},
		},
		{
			name: "profile_nonexistent_returns_error",
			options: &Options{
				Path:            "../testdata",
				CoverageProfile: "/no/such/profile.out",
			},
			patterns: nil,
			exclude:  nil,
			checks: []func(*testing.T, []coverage.ModuleCoverage, error){
				func(t *testing.T, _ []coverage.ModuleCoverage, err error) {
					t.Helper()
					assert.Error(t, err)
					assert.Contains(t, err.Error(), "coverage scan")
				},
			},
		},
		{
			name: "explicit_patterns_resolved_via_go_list",
			options: &Options{
				Path:         "../testdata",
				IncludeTests: true,
			},
			patterns: []string{"../testdata"},
			exclude:  nil,
			checks: []func(*testing.T, []coverage.ModuleCoverage, error){
				func(t *testing.T, r []coverage.ModuleCoverage, err error) {
					t.Helper()
					assert.NoError(t, err)
					assert.NotEmpty(t, r)
				},
			},
		},
		{
			name: "path_only_no_patterns_uses_path_fallback",
			options: &Options{
				Path:         "../testdata",
				IncludeTests: false,
			},
			patterns: nil,
			exclude:  nil,
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
			name: "timeout_zero_uses_default",
			options: &Options{
				Path:         "../testdata",
				IncludeTests: true,
			},
			patterns: nil,
			exclude:  nil,
			checks: []func(*testing.T, []coverage.ModuleCoverage, error){
				func(t *testing.T, r []coverage.ModuleCoverage, err error) {
					t.Helper()
					assert.NoError(t, err)
					assert.NotEmpty(t, r)
				},
			},
		},
		{
			name: "include_tests_false_excludes_test_files",
			options: &Options{
				Path:         "../testdata",
				IncludeTests: false,
			},
			patterns: nil,
			exclude:  nil,
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
			name: "path_with_existing_profile_skips_go_list",
			options: &Options{
				Path:            "../testdata",
				CoverageProfile: "../testdata/cover.out",
				IncludeTests:    true,
			},
			patterns: nil,
			exclude:  nil,
			checks: []func(*testing.T, []coverage.ModuleCoverage, error){
				func(t *testing.T, r []coverage.ModuleCoverage, err error) {
					t.Helper()
					assert.NoError(t, err)
					assert.NotEmpty(t, r)
					for _, mc := range r {
						assert.NotEmpty(t, mc.Functions, "profile scan should have parsed functions")
					}
				},
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			var ctx context.Context
			var cancel context.CancelFunc
			if tt.name == "context_cancelled_returns_context_error" {
				ctx, cancel = context.WithCancel(context.Background())
				cancel()
			} else {
				ctx, cancel = context.WithCancel(context.Background())
				defer cancel()
			}
			r, err := runCoverageAnalysis(ctx, tt.options, tt.patterns, tt.exclude, 0)
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

func Test_runCoverageAnalysis_resolveErrorNoPathFallback(t *testing.T) {
	dir := t.TempDir()
	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origCwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, err = runCoverageAnalysis(ctx, &Options{}, []string{"./..."}, nil, 0)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "coverage scan: resolve patterns")
	}
}

func Test_runComplexityAnalysis_resolveErrorNoPathFallback(t *testing.T) {
	dir := t.TempDir()
	origCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(origCwd) })
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Suppress debug output from the logger.
	l := &discardLogger{}
	result := runComplexityAnalysis(ctx, &Options{Logger: l}, []string{"./..."}, nil)
	assert.Empty(t, result)
}

type discardLogger struct{}

func (discardLogger) Debug(_ string, _ ...any) {}
func (discardLogger) Info(_ string, _ ...any)  {}
func (discardLogger) Warn(_ string, _ ...any)  {}
func (discardLogger) Error(_ string, _ ...any) {}
func (discardLogger) Fatal(_ string, _ ...any) {}

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

type groupTargetsFn func(*testing.T, []coverage.ModuleTarget)

var checkgroupTargets = func(fns ...groupTargetsFn) []groupTargetsFn { return fns }

var checkGroupTargetsLen = func(n int) groupTargetsFn {
	return func(t *testing.T, mt []coverage.ModuleTarget) {
		t.Helper()
		assert.Lenf(t, mt, n, "groupTargets: expected %d module targets", n)
	}
}

var checkGroupTargetsModule = func(modDir string, pkgDirs ...string) groupTargetsFn {
	return func(t *testing.T, mt []coverage.ModuleTarget) {
		t.Helper()
		var found *coverage.ModuleTarget
		for i := range mt {
			if mt[i].ModDir == modDir {
				found = &mt[i]
				break
			}
		}
		if !assert.NotNilf(t, found, "groupTargets: module %q not found in %+v", modDir, mt) {
			return
		}
		assert.ElementsMatchf(t, pkgDirs, found.PkgDirs, "module %q pkg dirs mismatch", modDir)
	}
}

func Test_groupTargets(t *testing.T) {
	tests := []struct {
		name    string
		targets []packages.Target
		checks  []groupTargetsFn
	}{
		{
			name:    "no targets",
			targets: nil,
			checks:  checkgroupTargets(checkGroupTargetsLen(0)),
		},
		{
			name: "skips targets without module dir",
			targets: []packages.Target{
				{ImportPath: "example.com/foo", Dir: "/work/foo", ModuleDir: ""},
			},
			checks: checkgroupTargets(checkGroupTargetsLen(0)),
		},
		{
			name: "single module single package",
			targets: []packages.Target{
				{ImportPath: "example.com/foo", Dir: "/mod/foo", ModulePath: "example.com", ModuleDir: "/mod"},
			},
			checks: checkgroupTargets(
				checkGroupTargetsLen(1),
				checkGroupTargetsModule("/mod", "/mod/foo"),
			),
		},
		{
			name: "single module multiple packages",
			targets: []packages.Target{
				{ImportPath: "example.com/a", Dir: "/mod/a", ModuleDir: "/mod"},
				{ImportPath: "example.com/b", Dir: "/mod/b", ModuleDir: "/mod"},
				{ImportPath: "example.com/c", Dir: "/mod/c", ModuleDir: "/mod"},
			},
			checks: checkgroupTargets(
				checkGroupTargetsLen(1),
				checkGroupTargetsModule("/mod", "/mod/a", "/mod/b", "/mod/c"),
			),
		},
		{
			name: "multiple modules",
			targets: []packages.Target{
				{ImportPath: "example.com/a", Dir: "/modA/a", ModuleDir: "/modA"},
				{ImportPath: "example.com/b", Dir: "/modB/b", ModuleDir: "/modB"},
			},
			checks: checkgroupTargets(
				checkGroupTargetsLen(2),
				checkGroupTargetsModule("/modA", "/modA/a"),
				checkGroupTargetsModule("/modB", "/modB/b"),
			),
		},
		{
			name: "mixed modules with skipped targets",
			targets: []packages.Target{
				{ImportPath: "example.com/standalone", Dir: "/solo", ModuleDir: ""},
				{ImportPath: "example.com/a", Dir: "/mod/a", ModuleDir: "/mod"},
				{ImportPath: "example.com/b", Dir: "/mod/b", ModuleDir: "/mod"},
				{ImportPath: "example.com/other", Dir: "/other", ModuleDir: "/other"},
			},
			checks: checkgroupTargets(
				checkGroupTargetsLen(2),
				checkGroupTargetsModule("/mod", "/mod/a", "/mod/b"),
				checkGroupTargetsModule("/other", "/other"),
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r := groupTargets(tt.targets)
			for _, c := range tt.checks {
				c(t, r)
			}
		})
	}
}

type runComplexityAnalysisFn func(*testing.T, []complexity.Stat)

var checkrunComplexityAnalysis = func(fns ...runComplexityAnalysisFn) []runComplexityAnalysisFn { return fns }

var checkComplexityLen = func(n int) runComplexityAnalysisFn {
	return func(t *testing.T, stats []complexity.Stat) {
		t.Helper()
		assert.Lenf(t, stats, n, "runComplexityAnalysis: expected %d stats", n)
	}
}

var checkComplexityContains = func(name string) runComplexityAnalysisFn {
	return func(t *testing.T, stats []complexity.Stat) {
		t.Helper()
		for _, s := range stats {
			if s.FuncName == name {
				return
			}
		}
		t.Errorf("runComplexityAnalysis: expected stat %q in %+v", name, stats)
	}
}

var checkComplexityNotContains = func(name string) runComplexityAnalysisFn {
	return func(t *testing.T, stats []complexity.Stat) {
		t.Helper()
		for _, s := range stats {
			assert.NotEqualf(t, name, s.FuncName, "runComplexityAnalysis: unexpected stat %q", name)
		}
	}
}

var checkComplexityEmpty = func() runComplexityAnalysisFn {
	return func(t *testing.T, stats []complexity.Stat) {
		t.Helper()
		assert.Emptyf(t, stats, "runComplexityAnalysis: expected no stats")
	}
}

var checkComplexityLenGreaterThan = func(n int) runComplexityAnalysisFn {
	return func(t *testing.T, stats []complexity.Stat) {
		t.Helper()
		assert.Greaterf(t, len(stats), n, "runComplexityAnalysis: expected >%d stats, got %d", n, len(stats))
	}
}

func Test_runComplexityAnalysis(t *testing.T) {
	tests := []struct {
		name     string
		options  *Options
		patterns []string
		exclude  *regexp.Regexp
		checks   []runComplexityAnalysisFn
	}{
		{
			name:   "resolve_success_returns_stats",
			options: &Options{
				IncludeTests: false,
			},
			patterns: []string{"."},
			checks: checkrunComplexityAnalysis(
				checkComplexityLen(20),
				checkComplexityContains("Scan"),
				checkComplexityContains("runComplexityAnalysis"),
				checkComplexityContains("NewEntries"),
			),
		},
		{
			name:   "include_tests_true_appends_test_stats",
			options: &Options{
				IncludeTests: true,
			},
			patterns: []string{"."},
			checks: checkrunComplexityAnalysis(
				checkComplexityContains("TestScan"),
				checkComplexityContains("Test_groupTargets"),
				checkComplexityLenGreaterThan(10),
			),
		},
		{
			name:    "exclude_pattern_filters_functions",
			options: &Options{
				IncludeTests: false,
			},
			patterns: []string{"."},
			exclude:  regexp.MustCompile("runComplexityAnalysis"),
			checks: checkrunComplexityAnalysis(
				checkComplexityLen(19),
				checkComplexityNotContains("runComplexityAnalysis"),
				checkComplexityContains("Scan"),
			),
		},
		{
			name:    "exclude_test_files_filters_file_paths",
			options: &Options{
				IncludeTests: true,
			},
			patterns: []string{"."},
			exclude:  regexp.MustCompile(`_test\.go`),
			checks: checkrunComplexityAnalysis(
				checkComplexityLen(20),
				checkComplexityNotContains("TestScan"),
				checkComplexityNotContains("Test_groupTargets"),
			),
		},
		{
			name:   "resolve_error_path_fallback_walks_directory",
			options: &Options{
				IncludeTests: true,
				Path:         "../testdata",
			},
			patterns: []string{"../testdata"},
			checks: checkrunComplexityAnalysis(
				checkComplexityLen(6),
				checkComplexityContains("veryComplex"),
				checkComplexityContains("withSwitch"),
				checkComplexityContains("simple"),
				checkComplexityContains("withIf"),
				checkComplexityContains("complex"),
				checkComplexityContains("TestSimple"),
			),
		},
		{
			name:    "path_fallback_excludes_test_files",
			options: &Options{
				IncludeTests: true,
				Path:         "../testdata",
			},
			patterns: []string{"../testdata"},
			exclude:  regexp.MustCompile(`_test\.go`),
			checks: checkrunComplexityAnalysis(
				checkComplexityLen(5),
				checkComplexityNotContains("TestSimple"),
				checkComplexityContains("veryComplex"),
				checkComplexityContains("simple"),
			),
		},
		{
			name:   "resolve_error_without_path_returns_empty",
			options: &Options{
				IncludeTests: false,
			},
			patterns: []string{"../nonexistent/..."},
			checks: checkrunComplexityAnalysis(checkComplexityEmpty()),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r := runComplexityAnalysis(context.Background(), tt.options, tt.patterns, tt.exclude)
			for _, c := range tt.checks {
				c(t, r)
			}
		})
	}
}
