package report

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/padiazg/go-crap/internal/scan"
	"github.com/padiazg/go-crap/internal/score"
)

type ComputeSummaryFn func(*testing.T, Summary)

var checkComputeSummary = func(fns ...ComputeSummaryFn) []ComputeSummaryFn { return fns }

func checkSummaryCombined(want float64) ComputeSummaryFn {
	return func(t *testing.T, s Summary) {
		t.Helper()
		assert.InDeltaf(t, want, s.Combined, 0.01, "Combined mismatch")
	}
}

func checkSummaryAverage(want float64) ComputeSummaryFn {
	return func(t *testing.T, s Summary) {
		t.Helper()
		assert.InDeltaf(t, want, s.Average, 0.01, "Average mismatch")
	}
}

func checkSummaryTotalFuncs(want int) ComputeSummaryFn {
	return func(t *testing.T, s Summary) {
		t.Helper()
		assert.Equalf(t, want, s.TotalFuncs, "TotalFuncs mismatch")
	}
}

func checkSummaryExceeded(want int) ComputeSummaryFn {
	return func(t *testing.T, s Summary) {
		t.Helper()
		assert.Equalf(t, want, s.Exceeded, "Exceeded mismatch")
	}
}

func TestComputeSummary(t *testing.T) {
	tests := []struct {
		name      string
		entries   *scan.Entries
		threshold float64
		checks    []ComputeSummaryFn
	}{
		{
			name:      "nil_entries_returns_empty_summary",
			entries:   nil,
			threshold: 30.0,
			checks: checkComputeSummary(
				checkSummaryCombined(0),
				checkSummaryAverage(0),
				checkSummaryTotalFuncs(0),
				checkSummaryExceeded(0),
			),
		},
		{
			name:      "entries_with_both_nil_lists",
			entries:   &scan.Entries{},
			threshold: 30.0,
			checks: checkComputeSummary(
				checkSummaryCombined(0),
				checkSummaryAverage(0),
				checkSummaryTotalFuncs(0),
				checkSummaryExceeded(0),
			),
		},
		{
			name:      "entries_with_only_list_uses_fortable",
			entries:   &scan.Entries{List: []score.CRAPEntry{{FuncName: "foo", CRAP: 5, EffectiveCRAP: 5}}},
			threshold: 30.0,
			checks: checkComputeSummary(
				checkSummaryCombined(5),
				checkSummaryAverage(5),
				checkSummaryTotalFuncs(1),
				checkSummaryExceeded(0),
			),
		},
		{
			name: "entries_with_fulllist_only_ignores_list",
			entries: &scan.Entries{
				FullList: []score.CRAPEntry{
					{FuncName: "a", CRAP: 100, EffectiveCRAP: 100},
					{FuncName: "b", CRAP: 1, EffectiveCRAP: 1},
				},
				List: nil,
			},
			threshold: 30.0,
			checks: checkComputeSummary(
				checkSummaryCombined(101),
				checkSummaryAverage(50.5),
				checkSummaryTotalFuncs(2),
				checkSummaryExceeded(1),
			),
		},
		{
			name:      "success_empty_entries",
			entries:   &scan.Entries{List: []score.CRAPEntry{}},
			threshold: 30.0,
			checks: checkComputeSummary(
				checkSummaryCombined(0),
				checkSummaryAverage(0),
				checkSummaryTotalFuncs(0),
				checkSummaryExceeded(0),
			),
		},
		{
			name: "success_single_entry_below_threshold",
			entries: &scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Good", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1},
			}},
			threshold: 30.0,
			checks: checkComputeSummary(
				checkSummaryCombined(1),
				checkSummaryAverage(1),
				checkSummaryTotalFuncs(1),
				checkSummaryExceeded(0),
			),
		},
		{
			name: "success_single_entry_above_threshold",
			entries: &scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Bad", Line: 1, Complexity: 10, Coverage: 0, CRAP: 100},
			}},
			threshold: 30.0,
			checks: checkComputeSummary(
				checkSummaryCombined(100),
				checkSummaryAverage(100),
				checkSummaryTotalFuncs(1),
				checkSummaryExceeded(1),
			),
		},
		{
			name: "success_multiple_entries_mixed",
			entries: &scan.Entries{List: []score.CRAPEntry{
				{File: "/project/a.go", Package: "myapp", FuncName: "Good", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1},
				{File: "/project/b.go", Package: "myapp", FuncName: "Warning", Line: 5, Complexity: 4, Coverage: 60, CRAP: 14.4},
				{File: "/project/c.go", Package: "myapp", FuncName: "Bad", Line: 10, Complexity: 10, Coverage: 0, CRAP: 110},
			}},
			threshold: 20.0,
			checks: checkComputeSummary(
				checkSummaryCombined(125.4),
				checkSummaryAverage(41.8),
				checkSummaryTotalFuncs(3),
				checkSummaryExceeded(1),
			),
		},
		{
			name: "success_all_above_threshold",
			entries: &scan.Entries{List: []score.CRAPEntry{
				{File: "/project/a.go", Package: "myapp", FuncName: "Bad1", Line: 1, Complexity: 10, Coverage: 0, CRAP: 100},
				{File: "/project/b.go", Package: "myapp", FuncName: "Bad2", Line: 1, Complexity: 10, Coverage: 0, CRAP: 100},
			}},
			threshold: 30.0,
			checks: checkComputeSummary(
				checkSummaryCombined(200),
				checkSummaryAverage(100),
				checkSummaryTotalFuncs(2),
				checkSummaryExceeded(2),
			),
		},
		{
			name: "success_all_below_threshold",
			entries: &scan.Entries{List: []score.CRAPEntry{
				{File: "/project/a.go", Package: "myapp", FuncName: "Good1", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1},
				{File: "/project/b.go", Package: "myapp", FuncName: "Good2", Line: 1, Complexity: 2, Coverage: 100, CRAP: 2},
			}},
			threshold: 30.0,
			checks: checkComputeSummary(
				checkSummaryCombined(3),
				checkSummaryAverage(1.5),
				checkSummaryTotalFuncs(2),
				checkSummaryExceeded(0),
			),
		},
		{
			name: "success_exactly_at_threshold_not_exceeded",
			entries: &scan.Entries{List: []score.CRAPEntry{
				{File: "/project/a.go", Package: "myapp", FuncName: "Edge", Line: 1, Complexity: 1, Coverage: 100, CRAP: 30},
			}},
			threshold: 30.0,
			checks: checkComputeSummary(
				checkSummaryCombined(30),
				checkSummaryAverage(30),
				checkSummaryTotalFuncs(1),
				checkSummaryExceeded(0),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := ComputeSummary(tt.entries, tt.threshold)
			for _, c := range tt.checks {
				c(t, r)
			}
		})
	}
}

func TestComputeSummary_with_effective_score(t *testing.T) {
	entries := &scan.Entries{List: []score.CRAPEntry{
		{File: "/project/a.go", Package: "myapp", FuncName: "Mutated", Line: 1, Complexity: 4, Coverage: 100, CRAP: 4, EffectiveCRAP: 16},
		{File: "/project/b.go", Package: "myapp", FuncName: "Normal", Line: 1, Complexity: 2, Coverage: 100, CRAP: 2},
	}}
	threshold := 10.0
	s := ComputeSummary(entries, threshold)
	require.InDelta(t, 18.0, s.Combined, 0.01)
	require.InDelta(t, 9.0, s.Average, 0.01)
	require.Equal(t, 2, s.TotalFuncs)
	require.Equal(t, 1, s.Exceeded)
}

func TestComputeSummary_uses_unfiltered_list(t *testing.T) {
	filtered := []score.CRAPEntry{
		{FuncName: "top1", Complexity: 10, Coverage: 0, CRAP: 100, EffectiveCRAP: 100},
		{FuncName: "top2", Complexity: 5, Coverage: 50, CRAP: 10, EffectiveCRAP: 10},
	}
	full := []score.CRAPEntry{
		{FuncName: "top1", Complexity: 10, Coverage: 0, CRAP: 100, EffectiveCRAP: 100},
		{FuncName: "top2", Complexity: 5, Coverage: 50, CRAP: 10, EffectiveCRAP: 10},
		{FuncName: "top3", Complexity: 3, Coverage: 80, CRAP: 3, EffectiveCRAP: 3},
		{FuncName: "top4", Complexity: 2, Coverage: 90, CRAP: 2, EffectiveCRAP: 2},
		{FuncName: "top5", Complexity: 1, Coverage: 100, CRAP: 1, EffectiveCRAP: 1},
	}
	entries := &scan.Entries{
		List:     filtered,
		FullList: full,
	}
	threshold := 20.0
	s := ComputeSummary(entries, threshold)
	require.Equal(t, 5, s.TotalFuncs, "TotalFuncs should reflect full list")
	require.InDelta(t, 116.0, s.Combined, 0.01, "Combined should sum over full list")
	require.InDelta(t, 23.2, s.Average, 0.01, "Average should divide over full list")
	require.Equal(t, 1, s.Exceeded, "Exceeded should count over full list")
}

func TestRelativizePath_empty_base_dir(t *testing.T) {
	got := RelativizePath("/project/main.go", "")
	assert.Equal(t, "/project/main.go", got)
}

func TestRelativizePath_absolute_base_dir(t *testing.T) {
	got := RelativizePath("/tmp/project/main.go", "/tmp/project")
	assert.Equal(t, "main.go", got)
}

func TestRelativizePath_no_match(t *testing.T) {
	got := RelativizePath("/other/main.go", "/tmp/project")
	assert.Equal(t, "../../other/main.go", got)
}
