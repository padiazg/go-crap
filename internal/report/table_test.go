package report

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

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
		entries score.EntryList
		opts    FormatOptions
		checks  []checkTableFormatterOutputFn
	}{
		{
			name:    "success_empty_entries",
			entries: score.EntryList{List: []score.CRAPEntry{}},
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
			entries: score.EntryList{List: []score.CRAPEntry{
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
			entries: score.EntryList{List: []score.CRAPEntry{
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
			entries: score.EntryList{List: []score.CRAPEntry{
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
			entries: score.EntryList{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "FullCovered", Line: 1, Complexity: 1, Coverage: 100, CRAP: 2},
			}},
			opts: FormatOptions{Threshold: 200},
			checks: checkTableFormatterOutput(
				checkOutputContains("██████████"),
			),
		},
		{
			name: "success_coverage_bar_at_0",
			entries: score.EntryList{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "NotCovered", Line: 1, Complexity: 1, Coverage: 0, CRAP: 1},
			}},
			opts: FormatOptions{Threshold: 200},
			checks: checkTableFormatterOutput(
				checkOutputContains("░░░░░░░░░░"),
			),
		},
		{
			name: "success_coverage_bar_at_50",
			entries: score.EntryList{List: []score.CRAPEntry{
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
			entries: score.EntryList{List: []score.CRAPEntry{
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
			entries: score.EntryList{List: []score.CRAPEntry{
				{File: "/other/project/main.go", Package: "myapp", FuncName: "Process", Line: 5, Complexity: 2, Coverage: 90, CRAP: 3.38},
			}},
			opts: FormatOptions{Threshold: 200, BaseDir: "/tmp/project"},
			checks: checkTableFormatterOutput(
				checkOutputContains("/other/project/main.go:5"),
			),
		},
		{
			name: "success_sort_order_descending",
			entries: score.EntryList{List: []score.CRAPEntry{
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
			name: "success_output_format_structure",
			entries: score.EntryList{List: []score.CRAPEntry{
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
			entries: score.EntryList{List: []score.CRAPEntry{
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
			entries: score.EntryList{List: []score.CRAPEntry{
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
			entries: score.EntryList{List: []score.CRAPEntry{
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
			entries: score.EntryList{List: []score.CRAPEntry{
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
			entries: score.EntryList{List: []score.CRAPEntry{
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
		tt := tt
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
