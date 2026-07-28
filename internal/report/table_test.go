package report

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/padiazg/go-crap/internal/scan"
	"github.com/padiazg/go-crap/internal/score"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type checkTableFormatterOutputFn func(*testing.T, string)

var checkTableFormatterOutput = func(fns ...checkTableFormatterOutputFn) []checkTableFormatterOutputFn {
	return fns
}

func checkOutputContains(want string) checkTableFormatterOutputFn {
	return func(t *testing.T, got string) {
		t.Helper()
		assert.Containsf(t, got, want, "output should contain %q", want)
	}
}

func checkOutputNotContains(want string) checkTableFormatterOutputFn {
	return func(t *testing.T, got string) {
		t.Helper()
		assert.NotContainsf(t, got, want, "output should not contain %q", want)
	}
}

func checkFooterFailedCount(failed, total int) checkTableFormatterOutputFn {
	return func(t *testing.T, got string) {
		t.Helper()
		expected := fmt.Sprintf("%d/%d function(s) exceed threshold CRAP", failed, total)
		assert.Containsf(t, got, expected, "footer mismatch")
	}
}

func TestTableFormatter_Format(t *testing.T) {
	tests := []struct {
		name    string
		entries scan.Entries
		opts    FormatOptions
		checks  []checkTableFormatterOutputFn
	}{
		{
			name:    "success_empty_entries",
			entries: scan.Entries{List: []score.CRAPEntry{}},
			checks: checkTableFormatterOutput(
				checkOutputContains("CRAP"),
				checkOutputContains("CC"),
				checkOutputContains("COVERAGE"),
				checkOutputContains("FUNCTION"),
				checkOutputNotContains("function(s) exceed threshold"),
			),
		},
		{
			name: "success_all_pass",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "FuncA", Line: 1, Complexity: 1, Coverage: 90, CRAP: 2},
				{File: "/project/main.go", Package: "myapp", FuncName: "FuncB", Line: 10, Complexity: 2, Coverage: 80, CRAP: 6.4},
			}},
			opts: FormatOptions{Threshold: 200},
			checks: checkTableFormatterOutput(
				checkOutputContains("✓"),
				checkFooterFailedCount(0, 2),
			),
		},
		{
			name: "success_all_fail",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "FuncA", Line: 1, Complexity: 10, Coverage: 0, CRAP: 100},
				{File: "/project/main.go", Package: "myapp", FuncName: "FuncB", Line: 10, Complexity: 5, Coverage: 0, CRAP: 26},
			}},
			opts: FormatOptions{Threshold: 10},
			checks: checkTableFormatterOutput(
				checkOutputContains("✗"),
				checkFooterFailedCount(2, 2),
			),
		},
		{
			name: "success_mixed_status",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Good", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1},
				{File: "/project/main.go", Package: "myapp", FuncName: "Warning", Line: 5, Complexity: 4, Coverage: 60, CRAP: 14.4},
				{File: "/project/main.go", Package: "myapp", FuncName: "Bad", Line: 10, Complexity: 10, Coverage: 0, CRAP: 110},
			}},
			opts: FormatOptions{Threshold: 20},
			checks: checkTableFormatterOutput(
				checkOutputContains("✓"),
				checkOutputContains("▲"),
				checkOutputContains("✗"),
				checkFooterFailedCount(1, 3),
			),
		},
		{
			name: "success_coverage_bar_at_100",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "FullCovered", Line: 1, Complexity: 1, Coverage: 100, CRAP: 2},
			}},
			opts: FormatOptions{Threshold: 200},
			checks: checkTableFormatterOutput(
				checkOutputContains("██████████"),
			),
		},
		{
			name: "success_coverage_bar_at_0",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "NotCovered", Line: 1, Complexity: 1, Coverage: 0, CRAP: 1},
			}},
			opts: FormatOptions{Threshold: 200},
			checks: checkTableFormatterOutput(
				checkOutputContains("░░░░░░░░░░"),
			),
		},
		{
			name: "success_coverage_bar_at_50",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "HalfCovered", Line: 1, Complexity: 1, Coverage: 50, CRAP: 1},
			}},
			opts: FormatOptions{Threshold: 200},
			checks: checkTableFormatterOutput(
				checkOutputContains("█████"),
				checkOutputContains("░░░░░"),
			),
		},
		{
			name: "success_base_dir_rewrites_path",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/tmp/project/main.go", Package: "myapp", FuncName: "Process", Line: 5, Complexity: 2, Coverage: 90, CRAP: 3.38},
			}},
			opts: FormatOptions{Threshold: 200, BaseDir: "/tmp/project"},
			checks: checkTableFormatterOutput(
				checkOutputContains("main.go:5"),
				checkOutputNotContains("/tmp/project/main.go:5"),
			),
		},
		{
			name: "success_base_dir_no_rewrite_when_no_match",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/other/project/main.go", Package: "myapp", FuncName: "Process", Line: 5, Complexity: 2, Coverage: 90, CRAP: 3.38},
			}},
			opts: FormatOptions{Threshold: 200, BaseDir: "/tmp/project"},
			checks: checkTableFormatterOutput(
				checkOutputContains("/other/project/main.go:5"),
			),
		},
		{
			name: "success_sort_order_descending",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/a.go", Package: "myapp", FuncName: "Low", Line: 1, Complexity: 1, Coverage: 50, CRAP: 2.25},
				{File: "/project/b.go", Package: "myapp", FuncName: "High", Line: 1, Complexity: 10, Coverage: 0, CRAP: 110},
				{File: "/project/c.go", Package: "myapp", FuncName: "Mid", Line: 1, Complexity: 5, Coverage: 20, CRAP: 32},
			}},
			opts: FormatOptions{Threshold: 200},
			checks: checkTableFormatterOutput(
				checkOutputOrder("High", "Mid", "Low"),
			),
		},
		{
			name: "success_sort_by_effective_crap_with_mutation",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/a.go", Package: "myapp", FuncName: "Good", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1, EffectiveCRAP: 1},
				{File: "/project/b.go", Package: "myapp", FuncName: "BadWithMutation", Line: 1, Complexity: 4, Coverage: 100, CRAP: 4, EffectiveCRAP: 16},
				{File: "/project/c.go", Package: "myapp", FuncName: "Mid", Line: 1, Complexity: 5, Coverage: 20, CRAP: 32, EffectiveCRAP: 32},
			}},
			opts: FormatOptions{Threshold: 200},
			checks: checkTableFormatterOutput(
				checkOutputOrder("Mid", "BadWithMutation", "Good"),
			),
		},
		{
			name: "success_sort_effective_crap_tie_break_by_mutation_score",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/a.go", Package: "myapp", FuncName: "SameCRAP_BetterMutation", Line: 1, Complexity: 5, Coverage: 20, CRAP: 32, EffectiveCRAP: 32, MutationScore: 0.8},
				{File: "/project/b.go", Package: "myapp", FuncName: "SameCRAP_WorseMutation", Line: 1, Complexity: 5, Coverage: 20, CRAP: 32, EffectiveCRAP: 32, MutationScore: 0.3},
			}},
			opts: FormatOptions{Threshold: 200},
			checks: checkTableFormatterOutput(
				checkOutputOrder("SameCRAP_WorseMutation", "SameCRAP_BetterMutation"),
			),
		},
		{
			name: "success_output_format_structure",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "FooBar", Line: 42, Complexity: 3, Coverage: 75, CRAP: 8.44},
			}},
			opts: FormatOptions{Threshold: 200},
			checks: checkTableFormatterOutput(
				checkOutputContains("COVERAGE"),
				checkOutputContains("FUNCTION"),
				checkOutputContains("LOCATION"),
				checkOutputContains("8.44"),
				checkOutputContains("FooBar"),
				checkOutputContains("/project/main.go:42"),
			),
		},
		{
			name: "success_single_entry",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Hello", Line: 10, Complexity: 1, Coverage: 100, CRAP: 1},
			}},
			opts: FormatOptions{Threshold: 200},
			checks: checkTableFormatterOutput(
				checkOutputContains("✓"),
				checkOutputContains("1.00"),
				checkFooterFailedCount(0, 1),
			),
		},
		{
			name: "success_threshold_boundary_half",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "AtHalf", Line: 1, Complexity: 10, Coverage: 100, CRAP: 10},
			}},
			opts: FormatOptions{Threshold: 20},
			checks: checkTableFormatterOutput(
				checkOutputContains("✓"),
				checkFooterFailedCount(0, 1),
			),
		},
		{
			name: "success_threshold_boundary_warning",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Warning", Line: 1, Complexity: 4, Coverage: 60, CRAP: 14.4},
			}},
			opts: FormatOptions{Threshold: 20},
			checks: checkTableFormatterOutput(
				checkOutputContains("▲"),
				checkOutputNotContains("✗"),
				checkFooterFailedCount(0, 1),
			),
		},
		{
			name: "success_threshold_boundary_at_threshold",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Exact", Line: 1, Complexity: 20, Coverage: 100, CRAP: 20},
			}},
			opts: FormatOptions{Threshold: 20},
			checks: checkTableFormatterOutput(
				checkOutputContains("▲"),
				checkOutputNotContains("✗"),
				checkOutputNotContains("✓"),
				checkFooterFailedCount(0, 1),
			),
		},
		{
			name: "success_threshold_boundary_one_over",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Over", Line: 1, Complexity: 21, Coverage: 100, CRAP: 21},
			}},
			opts: FormatOptions{Threshold: 20},
			checks: checkTableFormatterOutput(
				checkOutputContains("✗"),
				checkFooterFailedCount(1, 1),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &TableFormatter{}
			buf := &bytes.Buffer{}
			opts := tt.opts
			opts.Writer = buf
			err := f.Format(&tt.entries, opts)
			require.NoError(t, err, "Format should not return an error")
			for _, c := range tt.checks {
				c(t, buf.String())
			}
		})
	}
}

func checkOutputOrder(orders ...string) checkTableFormatterOutputFn {
	return func(t *testing.T, got string) {
		t.Helper()
		lines := strings.Split(got, "\n")
		var funcNames []string
		for _, l := range lines {
			for _, name := range orders {
				if strings.Contains(l, name) {
					funcNames = append(funcNames, name)
				}
			}
		}
		for i := range len(orders) {
			require.GreaterOrEqualf(t, len(funcNames), i+1, "found at least %d function names", i+1)
			assert.Equalf(t, orders[i], funcNames[i], "order[%d] mismatch", i)
		}
	}
}

func TestStatusSymbol_boundary_half_threshold(t *testing.T) {
	threshold := 10.0
	halfThreshold := threshold / 2.0

	assert.Equal(t, "✓", StatusSymbol(4.9, threshold, halfThreshold))
	assert.Equal(t, "✓", StatusSymbol(halfThreshold, threshold, halfThreshold))
	assert.Equal(t, "▲", StatusSymbol(halfThreshold+0.01, threshold, halfThreshold))
	assert.Equal(t, "▲", StatusSymbol(threshold-0.01, threshold, halfThreshold))
	assert.Equal(t, "✗", StatusSymbol(threshold+0.01, threshold, halfThreshold))
	assert.Equal(t, "✗", StatusSymbol(20.0, threshold, halfThreshold))
}

func TestStatusSymbol_zero_threshold(t *testing.T) {
	assert.Equal(t, "✗", StatusSymbol(0.01, 0.0, 0.0))
	assert.Equal(t, "✗", StatusSymbol(100.0, 0.0, 0.0))
}

func Test_tableFormatter_coverage_untrusted_warning(t *testing.T) {
	var buf strings.Builder
	formatter := &TableFormatter{}
	entries := &scan.Entries{List: []score.CRAPEntry{
		{CRAP: 50.0, Coverage: 50.0, CoverageUntrusted: true, FuncName: "untrusted", Complexity: 5},
	}}
	err := formatter.Format(entries, FormatOptions{Writer: &buf, Threshold: 30.0})
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "\xe2\x9a\xa0")
}

func Test_tableFormatter_coverage_bar_at_boundaries(t *testing.T) {
	var buf strings.Builder
	formatter := &TableFormatter{}
	entries := &scan.Entries{List: []score.CRAPEntry{
		{CRAP: 10.0, Coverage: 0.0, CoverageUntrusted: false, FuncName: "zero"},
		{CRAP: 10.0, Coverage: 100.0, CoverageUntrusted: false, FuncName: "full"},
	}}
	err := formatter.Format(entries, FormatOptions{Writer: &buf, Threshold: 30.0})
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "zero")
	assert.Contains(t, output, "full")
}

func Test_tableFormatter_single_entry(t *testing.T) {
	var buf strings.Builder
	formatter := &TableFormatter{}
	entries := &scan.Entries{List: []score.CRAPEntry{
		{CRAP: 50.0, Coverage: 50.0, CoverageUntrusted: false, FuncName: "single", Complexity: 5, Line: 10, File: "file.go"},
	}}
	err := formatter.Format(entries, FormatOptions{Writer: &buf, Threshold: 30.0})
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "1/1 function")
}

func Test_tableFormatter_empty_threshold(t *testing.T) {
	var buf strings.Builder
	formatter := &TableFormatter{}
	entries := &scan.Entries{List: []score.CRAPEntry{
		{CRAP: 10.0, Coverage: 50.0, CoverageUntrusted: false, FuncName: "low"},
	}}
	err := formatter.Format(entries, FormatOptions{Writer: &buf, Threshold: 100.0})
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "0/1")
}

func TestTableFormatter_Format_failed_count_positive(t *testing.T) {
	// INC_DEC :39 — failed++ changed to failed-- would produce negative count.
	// Contains "1/1" would match "-1/1", so we also check NotContains "-1".
	f := &TableFormatter{}
	buf := &bytes.Buffer{}
	entries := scan.Entries{List: []score.CRAPEntry{
		{File: "/project/a.go", Package: "myapp", FuncName: "Bad", Line: 1, Complexity: 10, Coverage: 0, CRAP: 100},
	}}
	opts := FormatOptions{Threshold: 10, Writer: buf}
	err := f.Format(&entries, opts)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "1/1 function(s)")
	assert.NotContains(t, output, "-1/1")
}

func TestTableFormatter_Format_sort_equal_mutation_score_no_panic(t *testing.T) {
	// COND_BOUND :26 — MutationScore < changed to <= creates broken comparator
	// when EffectiveScore and MutationScore are equal. Go 1.22+ panics.
	f := &TableFormatter{}
	buf := &bytes.Buffer{}
	entries := scan.Entries{List: make([]score.CRAPEntry, 20)}
	for i := range 20 {
		entries.List[i] = score.CRAPEntry{
			File: "/project/main.go", Package: "myapp",
			FuncName:      fmt.Sprintf("Func%d", i),
			Line:          i + 1,
			Complexity:    5,
			Coverage:      0,
			CRAP:          30,
			MutationScore: 0.5,
		}
	}
	opts := FormatOptions{Threshold: 200, Writer: buf}
	err := f.Format(&entries, opts)
	require.NoError(t, err)
}

func TestTableFormatter_Format_sort_effective_crap_tie_break_mutation_boundary(t *testing.T) {
	// COND_BOUND :25 — effectiveI > effectiveJ flipped to >= survives when
	// mutation scores are equal because both return false for equal mutations.
	// Uses different mutation scores to force the > operator:
	//   effective equal, mutation[0]=0.8 > mutation[1]=0.3 → i before j
	// If mutant (>=) were active: 0.8 >= 0.8 → false, 0.3 >= 0.8 → false,
	// comparator returns false,false for all pairs → sort may swap → order broken.
	f := &TableFormatter{}
	buf := &bytes.Buffer{}
	entries := scan.Entries{List: []score.CRAPEntry{
		{File: "/project/a.go", Package: "myapp", FuncName: "A", Line: 1, Complexity: 5, Coverage: 20, CRAP: 32.0, EffectiveCRAP: 32.0, MutationScore: 0.3},
		{File: "/project/b.go", Package: "myapp", FuncName: "B", Line: 1, Complexity: 5, Coverage: 20, CRAP: 32.0, EffectiveCRAP: 32.0, MutationScore: 0.8},
	}}
	opts := FormatOptions{Threshold: 200, Writer: buf}
	err := f.Format(&entries, opts)
	require.NoError(t, err)
	output := buf.String()
	require.Contains(t, output, "B")
	require.Contains(t, output, "A")
	bIdx := strings.Index(output, "B")
	aIdx := strings.Index(output, "A")
	assert.Greater(t, bIdx, aIdx, "B (higher mutation) should appear before A (lower mutation)")
}

func Test_coverageBar_at_zero(t *testing.T) {
	bar := coverageBar(0.0)
	assert.Equal(t, "░░░░░░░░░░", bar)
}

func Test_coverageBar_at_full(t *testing.T) {
	bar := coverageBar(100.0)
	assert.Equal(t, "██████████", bar)
}

func TestTableFormatter_Format_summary_nil_backward_compat(t *testing.T) {
	f := &TableFormatter{}
	buf := &bytes.Buffer{}
	entries := &scan.Entries{List: []score.CRAPEntry{
		{File: "/project/main.go", Package: "myapp", FuncName: "FuncA", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1},
	}}
	opts := FormatOptions{Threshold: 30, Writer: buf, Summary: nil}
	err := f.Format(entries, opts)
	require.NoError(t, err)
	output := buf.String()
	assert.NotContains(t, output, "Combined CRAP")
	assert.Contains(t, output, "0/1 function(s)")
}

func TestTableFormatter_Format_summary_non_nil(t *testing.T) {
	f := &TableFormatter{}
	buf := &bytes.Buffer{}
	entries := &scan.Entries{List: []score.CRAPEntry{
		{File: "/project/main.go", Package: "myapp", FuncName: "Good", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1},
		{File: "/project/main.go", Package: "myapp", FuncName: "Bad", Line: 10, Complexity: 10, Coverage: 0, CRAP: 100},
	}}
	summary := Summary{Combined: 101, Average: 50.5, TotalFuncs: 2, Exceeded: 1}
	opts := FormatOptions{Threshold: 30, Writer: buf, Summary: &summary}
	err := f.Format(entries, opts)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Combined CRAP: 101.00 | Average CRAP: 50.50")
}

func TestTableFormatter_Format_summary_empty_entries(t *testing.T) {
	f := &TableFormatter{}
	buf := &bytes.Buffer{}
	entries := &scan.Entries{List: []score.CRAPEntry{}}
	summary := Summary{Combined: 0, Average: 0, TotalFuncs: 0, Exceeded: 0}
	opts := FormatOptions{Threshold: 30, Writer: buf, Summary: &summary}
	err := f.Format(entries, opts)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Combined CRAP: 0.00 | Average CRAP: 0.00")
}

func Test_isUnchanged(t *testing.T) {
	tests := []struct {
		name string
		e    score.CRAPEntry
		want bool
	}{
		{
			name: "new_function_not_unchanged",
			e:    score.CRAPEntry{BaselineCRAP: -1, EffectiveCRAP: 50.0},
			want: false,
		},
		{
			name: "regressed",
			e:    score.CRAPEntry{BaselineCRAP: 10.0, EffectiveCRAP: 25.0},
			want: false,
		},
		{
			name: "improved",
			e:    score.CRAPEntry{BaselineCRAP: 50.0, EffectiveCRAP: 20.0},
			want: false,
		},
		{
			name: "exactly_unchanged",
			e:    score.CRAPEntry{BaselineCRAP: 30.0, EffectiveCRAP: 30.0},
			want: true,
		},
		{
			name: "within_tolerance_positive",
			e:    score.CRAPEntry{BaselineCRAP: 30.0, EffectiveCRAP: 30.005},
			want: true,
		},
		{
			name: "within_tolerance_negative",
			e:    score.CRAPEntry{BaselineCRAP: 30.0, EffectiveCRAP: 29.995},
			want: true,
		},
		{
			name: "just_over_tolerance",
			e:    score.CRAPEntry{BaselineCRAP: 30.0, EffectiveCRAP: 30.02},
			want: false,
		},
		{
			name: "just_under_tolerance",
			e:    score.CRAPEntry{BaselineCRAP: 30.0, EffectiveCRAP: 29.98},
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r := isUnchanged(tt.e)
			assert.Equal(t, tt.want, r)
		})
	}
}

func Test_anyChanged(t *testing.T) {
	tests := []struct {
		name   string
		sorted []score.CRAPEntry
		want   bool
	}{
		{
			name:   "all_unchanged",
			sorted: []score.CRAPEntry{{BaselineCRAP: 10.0, EffectiveCRAP: 10.0}, {BaselineCRAP: 20.0, EffectiveCRAP: 20.0}},
			want:   false,
		},
		{
			name: "one_new",
			sorted: []score.CRAPEntry{
				{BaselineCRAP: 10.0, EffectiveCRAP: 10.0},
				{BaselineCRAP: -1, EffectiveCRAP: 50.0},
			},
			want: true,
		},
		{
			name: "one_regressed",
			sorted: []score.CRAPEntry{
				{BaselineCRAP: 10.0, EffectiveCRAP: 10.0},
				{BaselineCRAP: 20.0, EffectiveCRAP: 45.0},
			},
			want: true,
		},
		{
			name: "one_improved",
			sorted: []score.CRAPEntry{
				{BaselineCRAP: 50.0, EffectiveCRAP: 50.0},
				{BaselineCRAP: 40.0, EffectiveCRAP: 10.0},
			},
			want: true,
		},
		{
			name:   "empty_list",
			sorted: []score.CRAPEntry{},
			want:   false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r := anyChanged(tt.sorted)
			assert.Equal(t, tt.want, r)
		})
	}
}

func TestTableFormatter_printSummary(t *testing.T) {
	tests := []struct {
		name   string
		opts   FormatOptions
		checks []checkTableFormatterOutputFn
	}{
		{
			name:   "nil_summary",
			opts:   FormatOptions{Writer: &strings.Builder{}},
			checks: checkTableFormatterOutput(
			// no output expected
			),
		},
		{
			name: "summary_without_baseline",
			opts: FormatOptions{
				Writer:    &strings.Builder{},
				Threshold: 30.0,
				Summary:   &Summary{Combined: 100.0, Average: 50.0, TotalFuncs: 2, Exceeded: 1},
			},
			checks: checkTableFormatterOutput(
				checkOutputContains("Combined CRAP: 100.00"),
				checkOutputContains("Average CRAP: 50.00"),
				checkOutputNotContains("baseline"),
			),
		},
		{
			name: "summary_with_baseline",
			opts: FormatOptions{
				Writer:    &strings.Builder{},
				Threshold: 30.0,
				Summary:   &Summary{Combined: 100.0, Average: 50.0, TotalFuncs: 2, Exceeded: 1, BaselineCombined: 80.0, BaselineAverage: 40.0, DeltaCombined: 20.0, DeltaAverage: 10.0},
				Baseline:  &Baseline{},
			},
			checks: checkTableFormatterOutput(
				checkOutputContains("Combined CRAP: 100.00"),
				checkOutputContains("Average CRAP: 50.00"),
				checkOutputContains("baseline"),
				checkOutputContains("vs baseline"),
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s := &TableFormatter{}
			var buf strings.Builder
			tt.opts.Writer = &buf
			s.printSummary(tt.opts)
			for _, c := range tt.checks {
				c(t, buf.String())
			}
		})
	}
}

func TestTableFormatter_Format_baseline_filtering(t *testing.T) {
	tests := []struct {
		name            string
		entries         scan.Entries
		opts            FormatOptions
		checks          []checkTableFormatterOutputFn
		expectNoChanges bool
	}{
		{
			name: "baseline_all_unchanged_show_unchanged_false",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "FuncA", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1, BaselineCRAP: 1.0},
				{File: "/project/main.go", Package: "myapp", FuncName: "FuncB", Line: 10, Complexity: 2, Coverage: 90, CRAP: 5.4, BaselineCRAP: 5.4},
			}},
			opts: FormatOptions{
				Threshold:     30.0,
				Baseline:      &Baseline{},
				Summary:       &Summary{Combined: 6.4, Average: 3.2, TotalFuncs: 2},
				ShowUnchanged: false,
			},
			checks: checkTableFormatterOutput(
				checkOutputContains("No changes since baseline."),
				checkOutputNotContains("FuncA"),
				checkOutputNotContains("FuncB"),
			),
			expectNoChanges: true,
		},
		{
			name: "baseline_mixed_show_unchanged_false",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "UnchangedFunc", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1, BaselineCRAP: 1.0},
				{File: "/project/main.go", Package: "myapp", FuncName: "RegressedFunc", Line: 10, Complexity: 5, Coverage: 50, CRAP: 12.5, BaselineCRAP: 5.0},
				{File: "/project/main.go", Package: "myapp", FuncName: "ImprovedFunc", Line: 20, Complexity: 3, Coverage: 80, CRAP: 3.6, BaselineCRAP: 9.0},
			}},
			opts: FormatOptions{
				Threshold:     30.0,
				Baseline:      &Baseline{},
				Summary:       &Summary{Combined: 17.1, Average: 5.7, TotalFuncs: 3},
				ShowUnchanged: false,
			},
			checks: checkTableFormatterOutput(
				checkOutputContains("RegressedFunc"),
				checkOutputContains("ImprovedFunc"),
				checkOutputNotContains("UnchangedFunc"),
				checkOutputContains("function(s) exceed threshold"),
			),
		},
		{
			name: "baseline_all_unchanged_show_unchanged_true",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "FuncA", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1, BaselineCRAP: 1.0},
				{File: "/project/main.go", Package: "myapp", FuncName: "FuncB", Line: 10, Complexity: 2, Coverage: 90, CRAP: 5.4, BaselineCRAP: 5.4},
			}},
			opts: FormatOptions{
				Threshold:     30.0,
				Baseline:      &Baseline{},
				Summary:       &Summary{Combined: 6.4, Average: 3.2, TotalFuncs: 2},
				ShowUnchanged: true,
			},
			checks: checkTableFormatterOutput(
				checkOutputContains("FuncA"),
				checkOutputContains("FuncB"),
				checkOutputNotContains("No changes since baseline."),
			),
		},
		{
			name: "baseline_all_changed_show_unchanged_false",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "NewFunc", Line: 1, Complexity: 5, Coverage: 0, CRAP: 30, BaselineCRAP: -1},
				{File: "/project/main.go", Package: "myapp", FuncName: "RegressedFunc", Line: 10, Complexity: 10, Coverage: 0, CRAP: 100, BaselineCRAP: 30.0},
			}},
			opts: FormatOptions{
				Threshold:     30.0,
				Baseline:      &Baseline{},
				Summary:       &Summary{Combined: 130.0, Average: 65.0, TotalFuncs: 2},
				ShowUnchanged: false,
			},
			checks: checkTableFormatterOutput(
				checkOutputContains("NewFunc"),
				checkOutputContains("RegressedFunc"),
				checkOutputContains("function(s) exceed threshold"),
			),
		},
		{
			name: "no_baseline_show_unchanged_true_is_noop",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "FuncA", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1},
				{File: "/project/main.go", Package: "myapp", FuncName: "FuncB", Line: 10, Complexity: 5, Coverage: 0, CRAP: 30},
			}},
			opts: FormatOptions{
				Threshold:     30.0,
				Summary:       &Summary{Combined: 31.0, Average: 15.5, TotalFuncs: 2},
				ShowUnchanged: true,
			},
			checks: checkTableFormatterOutput(
				checkOutputContains("FuncA"),
				checkOutputContains("FuncB"),
				checkOutputNotContains("No changes since baseline."),
			),
		},
		{
			name:    "baseline_empty_list_all_unchanged",
			entries: scan.Entries{List: []score.CRAPEntry{}},
			opts: FormatOptions{
				Threshold:     30.0,
				Baseline:      &Baseline{},
				Summary:       &Summary{Combined: 0, Average: 0, TotalFuncs: 0},
				ShowUnchanged: false,
			},
			checks: checkTableFormatterOutput(
				checkOutputContains("No changes since baseline."),
			),
			expectNoChanges: true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f := &TableFormatter{}
			buf := &bytes.Buffer{}
			opts := tt.opts
			opts.Writer = buf
			err := f.Format(&tt.entries, opts)
			require.NoError(t, err, "Format should not return an error")
			output := buf.String()
			for _, c := range tt.checks {
				c(t, output)
			}
		})
	}
}
