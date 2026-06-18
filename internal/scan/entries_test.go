package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/padiazg/go-crap/internal/merge"
	"github.com/padiazg/go-crap/internal/score"
)

type checkEntriesFn func(*testing.T, *Entries, error)

var checkEntries = func(fns ...checkEntriesFn) []checkEntriesFn { return fns }

func checkEntriesError(fn, want string) checkEntriesFn {
	return func(t *testing.T, _ *Entries, err error) {
		t.Helper()
		if want == "" {
			assert.NoErrorf(t, err, "check%sError: expected no error, got %v", fn, err)
			return
		}
		if assert.Errorf(t, err, "check%sError: expected error %q", fn, want) {
			assert.Containsf(t, err.Error(), want, "check%sError mismatch", fn)
		}
	}
}

func checkEntriesEmptyList(fn string, want bool) checkEntriesFn {
	return func(t *testing.T, r *Entries, err error) {
		t.Helper()
		if want {
			assert.Emptyf(t, r.List, "check%sError: list expected to be empty", fn)
		} else {
			assert.NotEmptyf(t, r.List, "check%sError: list expected to be not empty", fn)
		}
	}
}

type checkEntryFn func(*testing.T, *Entries)

func checkEntry(checks ...checkEntryFn) checkEntriesFn {
	return func(t *testing.T, e *Entries, err error) {
		for _, c := range checks {
			c(t, e)
		}
	}
}

func checkEntriesLen(fn string, want int) checkEntryFn {
	return func(t *testing.T, e *Entries) {
		t.Helper()
		l := len(e.List)
		assert.Equalf(t, want, l, "check%sEntryLen: %d, expected %d", fn, l, want)
	}
}

func checkEntryName(fn string, idx int, want string) checkEntryFn {
	return func(t *testing.T, e *Entries) {
		t.Helper()
		item := e.List[idx]
		assert.Equalf(t, want, item.FuncName, "checkEntry%sName: %s, expected %s", fn, item.FuncName, want)
	}
}

func checkEntryCoverage(fn string, idx int, want float64) checkEntryFn {
	return func(t *testing.T, e *Entries) {
		t.Helper()
		item := e.List[idx]
		assert.Equalf(t, want, item.Coverage, "checkEntry%sCoverage: %d, expected %d", fn, item.Coverage, want)
	}
}

func checkEntrySkipped(fn string, idx int, want bool) checkEntryFn {
	return func(t *testing.T, e *Entries) {
		t.Helper()
		item := e.List[idx]
		assert.Equalf(t, want, item.Skipped, "checkEntry%sSkipped: %t, expected %t", fn, item.Skipped, want)
	}
}

func checkEntryEffectiveCRAP(fn string, idx int, want float64) checkEntryFn {
	return func(t *testing.T, e *Entries) {
		t.Helper()
		item := e.List[idx]
		assert.Equalf(t, want, item.EffectiveCRAP, "checkEntry%s: %d, expected %d", fn, item.EffectiveCRAP, want)
	}
}

func TestNewEntries(t *testing.T) {
	const fn = "NewEntries"
	tests := []struct {
		name    string
		options *Options
		merged  []merge.MergedEntry
		policy  score.MissingPolicy
		checks  []checkEntriesFn
	}{
		{
			name:    "empty merged entries",
			options: &Options{},
			merged:  []merge.MergedEntry{},
			policy:  score.MissingPessimistic,
			checks: checkEntries(
				checkEntriesError(fn, ""),
				checkEntriesEmptyList(fn, true),
			),
		},
		{
			name:    "success pessimistic policy no coverage",
			options: &Options{},
			merged: []merge.MergedEntry{
				{File: "example.go", FuncName: "Foo", Package: "example", Complexity: 5, Coverage: nil, Line: 10},
			},
			policy: score.MissingPessimistic,
			checks: checkEntries(
				checkEntriesError(fn, ""),
				checkEntry(
					checkEntriesLen(fn, 1),
					checkEntryName(fn, 0, "Foo"),
					checkEntryCoverage(fn, 0, 0.0),
					checkEntrySkipped(fn, 0, false),
				),
			),
		},
		{
			name:    "success optimistic policy no coverage",
			options: &Options{},
			merged: []merge.MergedEntry{
				{File: "example.go", FuncName: "Foo", Package: "example", Complexity: 5, Coverage: nil, Line: 10},
			},
			policy: score.MissingOptimistic,
			checks: checkEntries(
				checkEntriesError(fn, ""),
				checkEntry(
					checkEntriesLen(fn, 1),
					checkEntryName(fn, 0, "Foo"),
					checkEntryCoverage(fn, 0, 100.0),
					checkEntrySkipped(fn, 0, false),
				),
				// 	func(t *testing.T, r *Entries, err error) {
				// 	t.Helper()
				// 	// assert.NoError(t, err)
				// 	assert.Len(t, r.List, 1)
				// 	e := r.List[0]
				// 	assert.Equal(t, "Foo", e.FuncName)
				// 	assert.Equal(t, 100.0, e.Coverage)
				// 	assert.False(t, e.Skipped)
				// }
			),
		},
		{
			name:    "skip policy with nil coverage",
			options: &Options{},
			merged: []merge.MergedEntry{
				{File: "example.go", FuncName: "Foo", Package: "example", Complexity: 5, Coverage: nil, Line: 10},
			},
			policy: score.MissingSkip,
			checks: checkEntries(
				checkEntriesError(fn, ""),
				checkEntry(
					checkEntriesLen(fn, 1),
					checkEntryName(fn, 0, "Foo"),
					checkEntrySkipped(fn, 0, true),
					checkEntryEffectiveCRAP(fn, 0, 5.0),
				),
			// 	func(t *testing.T, r *Entries, err error) {
			// 	t.Helper()
			// 	assert.NoError(t, err)
			// 	assert.Len(t, r.List, 1)
			// 	e := r.List[0]
			// 	assert.Equal(t, "Foo", e.FuncName)
			// 	assert.True(t, e.Skipped)
			// 	assert.Equal(t, 5.0, e.EffectiveCRAP)
			// }
			),
		},
		{
			name:    "success with coverage and complexity",
			options: &Options{},
			merged: []merge.MergedEntry{
				{
					File:       "example.go",
					FuncName:   "Bar",
					Package:    "example",
					Complexity: 10,
					Coverage:   new(80.0),
					Line:       20,
				},
				{
					File:       "example.go",
					FuncName:   "Baz",
					Package:    "example",
					Complexity: 2,
					Coverage:   new(10.0),
					Line:       30,
				},
			},
			policy: score.MissingPessimistic,
			checks: checkEntries(
				checkEntriesError(fn, ""),
				checkEntry(
					checkEntriesLen(fn, 2),
					func(t *testing.T, e *Entries) {
						// Should be sorted by effective CRAP descending
						assert.GreaterOrEqual(t, e.List[0].EffectiveCRAP, e.List[1].EffectiveCRAP)
					},
				),
				// 	func(t *testing.T, r *Entries, err error) {
				// 	t.Helper()
				// 	assert.NoError(t, err)
				// 	assert.Len(t, r.List, 2)
				//
				// 	assert.GreaterOrEqual(t, r.List[0].EffectiveCRAP, r.List[1].EffectiveCRAP)
				// }
			),
		},
		{
			name:    "success untrusted coverage not filtered by Min",
			options: &Options{Min: 50},
			merged: []merge.MergedEntry{
				{
					File:       "example.go",
					FuncName:   "LowCRAP",
					Package:    "example",
					Complexity: 2,
					Coverage:   new(90.0),
					Line:       10,
					Receiver:   "Foo",
				},
				{
					File:       "example.go",
					FuncName:   "HighCRAP",
					Package:    "example",
					Complexity: 10,
					Coverage:   new(5.0),
					Line:       20,
					Receiver:   "Foo",
				},
			},
			policy: score.MissingPessimistic,
			checks: checkEntries(
				checkEntriesError(fn, ""),
				checkEntry(
					checkEntriesLen(fn, 1),
					// HighCRAP (CRAP~96) exceeds Min=50, LowCRAP (CRAP~2) filtered out
					checkEntryName(fn, 0, "HighCRAP"),
				),
				// 	func(t *testing.T, r *Entries, err error) {
				// 	t.Helper()
				// 	assert.NoError(t, err)
				// 	// HighCRAP (CRAP~96) exceeds Min=50, LowCRAP (CRAP~2) filtered out
				// 	assert.Len(t, r.List, 1)
				// 	assert.Equal(t, "HighCRAP", r.List[0].FuncName)
				// }
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, err := NewEntries(tt.options, tt.merged, tt.policy)
			for _, c := range tt.checks {
				c(t, r, err)
			}
		})
	}
}

func TestEntries_ThresholdExceeded(t *testing.T) {
	const fn = "ThresholdExceeded"
	tests := []struct {
		name      string
		entries   *Entries
		threshold float64
		want      bool
	}{
		{
			name:      "empty_list_returns_false",
			entries:   &Entries{List: []score.CRAPEntry{}},
			threshold: 100.0,
			want:      false,
		},
		{
			name: "all_below_threshold",
			entries: &Entries{List: []score.CRAPEntry{
				{FuncName: "a", EffectiveCRAP: 50},
				{FuncName: "b", EffectiveCRAP: 30},
			}},
			threshold: 100.0,
			want:      false,
		},
		{
			name: "all_at_threshold_not_exceeded",
			entries: &Entries{List: []score.CRAPEntry{
				{FuncName: "exact", EffectiveCRAP: 100},
			}},
			threshold: 100.0,
			want:      false,
		},
		{
			name: "one_entry_exceeding_threshold",
			entries: &Entries{List: []score.CRAPEntry{
				{FuncName: "low", EffectiveCRAP: 30},
				{FuncName: "high", EffectiveCRAP: 200},
			}},
			threshold: 100.0,
			want:      true,
		},
		{
			name: "all_above_threshold",
			entries: &Entries{List: []score.CRAPEntry{
				{FuncName: "a", EffectiveCRAP: 200},
				{FuncName: "b", EffectiveCRAP: 150},
			}},
			threshold: 100.0,
			want:      true,
		},
		{
			name: "single_entry_below_threshold",
			entries: &Entries{List: []score.CRAPEntry{
				{FuncName: "single", EffectiveCRAP: 50},
			}},
			threshold: 100.0,
			want:      false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.entries.ThresholdExceeded(tt.threshold)
			assert.Equal(t, tt.want, r, "%s: %s", fn, tt.name)
		})
	}
}

func TestEntries_applyMutationAnnotations(t *testing.T) {
	const fn = "applyMutationAnnotations"

	reportJSON := `{
		"go_module": "github.com/example/test",
		"files": [{"file_name": "example.go", "mutations": [
			{"type": "ARITHMETIC", "status": "LIVED", "line": 50}
		]}],
		"mutants_killed": 0,
		"mutants_lived": 1,
		"mutants_not_covered": 0,
		"mutants_total": 1,
		"test_efficacy": 0
	}`

	writeReport := func(t *testing.T, path string) {
		t.Helper()
		err := os.WriteFile(path, []byte(reportJSON), 0644)
		assert.NoError(t, err)
	}

	tests := []struct {
		name    string
		entries *Entries
		merged  []merge.MergedEntry
		checks  []checkEntriesFn
		before  func(*Entries)
		wantErr string
	}{
		{
			name:    "no_report_path_no_annotation",
			entries: &Entries{List: []score.CRAPEntry{{File: "a.go", FuncName: "Foo", Line: 1, Complexity: 5, Coverage: 80, CRAP: 50}}},
			merged:  []merge.MergedEntry{},
			checks: checkEntries(
				checkEntriesError(fn, ""),
			),
		},
		{
			name:    "invalid_path_returns_error",
			entries: &Entries{List: []score.CRAPEntry{{File: "b.go", FuncName: "Bar", Line: 1, Complexity: 3, Coverage: 70, CRAP: 15}}},
			merged:  []merge.MergedEntry{},
			checks: checkEntries(
				checkEntriesError(fn, "open"),
			),
			wantErr: "/nonexistent/report.json",
		},
		{
			name: "valid_report_applies_annotations",
			entries: &Entries{List: []score.CRAPEntry{
				{File: "example.go", FuncName: "Baz", Line: 10, Complexity: 5, Coverage: 80, CRAP: 30},
			}},
			merged: []merge.MergedEntry{
				{File: "example.go", FuncName: "Baz", Line: 10, EndLine: 100, Complexity: 5},
			},
			before: func(e *Entries) {
				path := filepath.Join(t.TempDir(), "report.json")
				writeReport(t, path)
				e.options = &Options{
					Logger:         nilLogger{},
					MutationReport: path,
				}
			},
			checks: checkEntries(
				checkEntriesError(fn, ""),
				checkEntry(
					checkEntriesLen(fn, 1),
					checkEntryCoverage(fn, 0, 80.0),
					checkEntrySkipped(fn, 0, false),
					checkEntryEffectiveCRAP(fn, 0, score.CRAP(5, 0)),
				),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.before != nil {
				tt.before(tt.entries)
			}
			if tt.entries.options == nil {
				tt.entries.options = &Options{
					Logger:         nilLogger{},
					MutationReport: tt.wantErr,
				}
			}
			err := tt.entries.applyMutationAnnotations(tt.merged)
			for _, c := range tt.checks {
				c(t, tt.entries, err)
			}
		})
	}
}

func TestEntries_applyFilters(t *testing.T) {
	const fn = "applyFilters"
	tests := []struct {
		name    string
		entries *Entries
		checks  []checkEntriesFn
		before  func(*Entries)
	}{
		{
			name:    "sort descending by effective CRAP",
			entries: &Entries{List: nil},
			checks: checkEntries(func(t *testing.T, r *Entries, err error) {
				t.Helper()
				for i := 1; i < len(r.List); i++ {
					assert.GreaterOrEqual(t, r.List[i-1].EffectiveCRAP, r.List[i].EffectiveCRAP)
				}
			}),
			before: func(e *Entries) {
				e.options = &Options{}
				e.List = []score.CRAPEntry{
					{FuncName: "low", EffectiveCRAP: 30, CRAP: 50},
					{FuncName: "high", EffectiveCRAP: 200, CRAP: 200},
					{FuncName: "mid", EffectiveCRAP: 100, CRAP: 150},
				}
			},
		},
		{
			name:    "sort with Min and Top filters",
			entries: &Entries{List: nil},
			checks: checkEntries(
				checkEntry(
					checkEntriesLen(fn, 2),
					checkEntryName(fn, 0, "untrusted"),
					checkEntryName(fn, 1, "high"),
				),
			),
			before: func(e *Entries) {
				e.options = &Options{Min: 100, Top: 1}
				e.List = []score.CRAPEntry{
					{FuncName: "low", EffectiveCRAP: 30, CRAP: 50, CoverageUntrusted: false},
					{FuncName: "high", EffectiveCRAP: 200, CRAP: 200, CoverageUntrusted: false},
					{FuncName: "untrusted", EffectiveCRAP: 50, CRAP: 50, CoverageUntrusted: true},
				}
			},
		},
		{
			name:    "no filters when Min and Top are zero",
			entries: &Entries{List: nil},
			checks: checkEntries(
				checkEntry(
					checkEntriesLen(fn, 3),
				),
			),
			before: func(e *Entries) {
				e.options = &Options{}
				e.List = []score.CRAPEntry{
					{FuncName: "a", EffectiveCRAP: 30},
					{FuncName: "b", EffectiveCRAP: 200},
					{FuncName: "c", EffectiveCRAP: 100},
				}
			},
		},
		{
			name:    "Min zero skips filterByMinCRAP_boundary",
			entries: &Entries{List: nil},
			checks: checkEntries(
				// Min=0: condition `Min > 0` is false, filterByMinCRAP not called.
				// Dead mutant: changing `> 0` to `>= 0` would call it, but
				// filterByMinCRAP(Min=0) keeps everything since CRAP >= 0 always.
				checkEntry(
					checkEntriesLen(fn, 3),
					checkEntryName(fn, 0, "high"),
					checkEntryName(fn, 1, "mid"),
					checkEntryName(fn, 2, "low"),
				),
			),
			before: func(e *Entries) {
				e.options = &Options{Min: 0}
				e.List = []score.CRAPEntry{
					{FuncName: "low", EffectiveCRAP: 30},
					{FuncName: "mid", EffectiveCRAP: 100},
					{FuncName: "high", EffectiveCRAP: 200},
				}
			},
		},
		{
			name:    "Top equal to len skips filterByTop_boundary",
			entries: &Entries{List: nil},
			checks: checkEntries(
				// Top==len: condition `Top < len` is false, filterByTop not called.
				// Dead mutant: changing `<` to `<=` would call it, but
				// filterByTop(Top==len) keeps all since kept < Top always holds.
				checkEntry(
					checkEntriesLen(fn, 3),
					checkEntryName(fn, 0, "high"),
					checkEntryName(fn, 1, "mid"),
					checkEntryName(fn, 2, "low"),
				),
			),
			before: func(e *Entries) {
				e.options = &Options{Top: 3}
				e.List = []score.CRAPEntry{
					{FuncName: "low", EffectiveCRAP: 30},
					{FuncName: "mid", EffectiveCRAP: 100},
					{FuncName: "high", EffectiveCRAP: 200},
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.before != nil {
				tt.before(tt.entries)
			}
			tt.entries.applyFilters()
			for _, c := range tt.checks {
				c(t, tt.entries, nil)
			}
		})
	}
}

func TestEntries_filterByMinCRAP(t *testing.T) {
	tests := []struct {
		name    string
		entries *Entries
		checks  []checkEntriesFn
		before  func(*Entries)
	}{
		{
			name:    "filter below Min threshold",
			entries: &Entries{},
			checks: checkEntries(func(t *testing.T, r *Entries, err error) {
				t.Helper()
				assert.Len(t, r.List, 2)
				names := funcNames(r.List)
				assert.Contains(t, names, "ok")
				assert.Contains(t, names, "untrusted_low")
				assert.NotContains(t, names, "below_min")
			}),
			before: func(e *Entries) {
				e.options = &Options{Min: 100}
				e.List = []score.CRAPEntry{
					{FuncName: "ok", EffectiveCRAP: 200, CoverageUntrusted: false},
					{FuncName: "below_min", EffectiveCRAP: 50, CoverageUntrusted: false},
					{FuncName: "untrusted_low", EffectiveCRAP: 50, CoverageUntrusted: true},
				}
			},
		},
		{
			name:    "untrusted coverage always kept",
			entries: &Entries{},
			checks: checkEntries(func(t *testing.T, r *Entries, err error) {
				t.Helper()
				assert.Len(t, r.List, 3)
				names := funcNames(r.List)
				assert.Contains(t, names, "untrusted_low", "untrusted entries should be kept regardless of Min")
				assert.Contains(t, names, "untrusted_high")
				assert.Contains(t, names, "above_min")
			}),
			before: func(e *Entries) {
				e.options = &Options{Min: 100}
				e.List = []score.CRAPEntry{
					{FuncName: "above_min", EffectiveCRAP: 200, CoverageUntrusted: false},
					{FuncName: "untrusted_low", EffectiveCRAP: 10, CoverageUntrusted: true},
					{FuncName: "untrusted_high", EffectiveCRAP: 300, CoverageUntrusted: true},
				}
			},
		},
		{
			name:    "all entries pass Min threshold",
			entries: &Entries{},
			checks: checkEntries(func(t *testing.T, r *Entries, err error) {
				t.Helper()
				assert.Len(t, r.List, 2)
			}),
			before: func(e *Entries) {
				e.options = &Options{Min: 10}
				e.List = []score.CRAPEntry{
					{FuncName: "a", EffectiveCRAP: 100},
					{FuncName: "b", EffectiveCRAP: 200},
				}
			},
		},
		{
			name:    "no entries pass Min threshold",
			entries: &Entries{},
			checks: checkEntries(func(t *testing.T, r *Entries, err error) {
				t.Helper()
				assert.Empty(t, r.List)
			}),
			before: func(e *Entries) {
				e.options = &Options{Min: 500}
				e.List = []score.CRAPEntry{
					{FuncName: "low1", EffectiveCRAP: 50},
					{FuncName: "low2", EffectiveCRAP: 100},
				}
			},
		},
		{
			name:    "equal_to_min_kept_boundary",
			entries: &Entries{},
			checks: checkEntries(func(t *testing.T, r *Entries, err error) {
				t.Helper()
				// >= vs >: at EffectiveCRAP == Min, entry should be kept
				assert.Len(t, r.List, 2)
				names := funcNames(r.List)
				assert.Contains(t, names, "at_min")
				assert.Contains(t, names, "above_min")
				assert.NotContains(t, names, "below_min")
			}),
			before: func(e *Entries) {
				e.options = &Options{Min: 100}
				e.List = []score.CRAPEntry{
					{FuncName: "below_min", EffectiveCRAP: 50},
					{FuncName: "at_min", EffectiveCRAP: 100},
					{FuncName: "above_min", EffectiveCRAP: 200},
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.before != nil {
				tt.before(tt.entries)
			}
			tt.entries.filterByMinCRAP()
			for _, c := range tt.checks {
				c(t, tt.entries, nil)
			}
		})
	}
}

func TestEntries_filterByTop(t *testing.T) {
	tests := []struct {
		name    string
		entries *Entries
		checks  []checkEntriesFn
		before  func(*Entries)
	}{
		{
			name:    "keep only top N non-untrusted entries",
			entries: &Entries{},
			checks: checkEntries(func(t *testing.T, r *Entries, err error) {
				t.Helper()
				assert.Len(t, r.List, 3)
				// untrusted always kept, then top 2 non-untrusted by effective CRAP
				names := funcNames(r.List)
				assert.Contains(t, names, "untrusted")
				assert.Contains(t, names, "high")
				assert.Contains(t, names, "mid")
				assert.NotContains(t, names, "low")
			}),
			before: func(e *Entries) {
				e.options = &Options{Top: 2}
				e.List = []score.CRAPEntry{
					{FuncName: "high", EffectiveCRAP: 200, CoverageUntrusted: false},
					{FuncName: "mid", EffectiveCRAP: 100, CoverageUntrusted: false},
					{FuncName: "low", EffectiveCRAP: 50, CoverageUntrusted: false},
					{FuncName: "untrusted", EffectiveCRAP: 10, CoverageUntrusted: true},
				}
			},
		},
		{
			name:    "all entries untrusted kept",
			entries: &Entries{},
			checks: checkEntries(func(t *testing.T, r *Entries, err error) {
				t.Helper()
				assert.Len(t, r.List, 3)
			}),
			before: func(e *Entries) {
				e.options = &Options{Top: 1}
				e.List = []score.CRAPEntry{
					{FuncName: "a", EffectiveCRAP: 50, CoverageUntrusted: true},
					{FuncName: "b", EffectiveCRAP: 100, CoverageUntrusted: true},
					{FuncName: "c", EffectiveCRAP: 150, CoverageUntrusted: true},
				}
			},
		},
		{
			name:    "top larger than available entries keeps all",
			entries: &Entries{},
			checks: checkEntries(func(t *testing.T, r *Entries, err error) {
				t.Helper()
				assert.Len(t, r.List, 2)
			}),
			before: func(e *Entries) {
				e.options = &Options{Top: 10}
				e.List = []score.CRAPEntry{
					{FuncName: "a", EffectiveCRAP: 50},
					{FuncName: "b", EffectiveCRAP: 100},
				}
			},
		},
		{
			name:    "empty list",
			entries: &Entries{},
			checks: checkEntries(func(t *testing.T, r *Entries, err error) {
				t.Helper()
				assert.Empty(t, r.List)
			}),
			before: func(e *Entries) {
				e.options = &Options{Top: 5}
				e.List = []score.CRAPEntry{}
			},
		},
		{
			name:    "untrusted entries come first in output",
			entries: &Entries{},
			checks: checkEntries(func(t *testing.T, r *Entries, err error) {
				t.Helper()
				// untrusted should appear before non-untrusted in result
				assert.Equal(t, "untrusted", r.List[0].FuncName)
				assert.Equal(t, "high", r.List[1].FuncName)
			}),
			before: func(e *Entries) {
				e.options = &Options{Top: 1}
				// Already sorted descending by EffectiveCRAP (as applyFilters does)
				e.List = []score.CRAPEntry{
					{FuncName: "high", EffectiveCRAP: 200, CoverageUntrusted: false},
					{FuncName: "low", EffectiveCRAP: 30, CoverageUntrusted: false},
					{FuncName: "untrusted", EffectiveCRAP: 10, CoverageUntrusted: true},
				}
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.before != nil {
				tt.before(tt.entries)
			}
			tt.entries.filterByTop()
			for _, c := range tt.checks {
				c(t, tt.entries, nil)
			}
		})
	}
}

// ponytail: minimal helper to avoid polluting package scope with multiple
// helper functions. All test helpers live below this comment.

func funcNames(entries []score.CRAPEntry) []string {
	names := make([]string, len(entries))
	for i, e := range entries {
		names[i] = e.FuncName
	}
	return names
}

type nilLogger struct{}

func (nilLogger) Debug(format string, args ...any) {}
func (nilLogger) Info(format string, args ...any)  {}
func (nilLogger) Warn(format string, args ...any)  {}
func (nilLogger) Error(format string, args ...any) {}
func (nilLogger) Fatal(format string, args ...any) {}

func TestEntries_sort(t *testing.T) {
	const fn = "sort"
	tests := []struct {
		name    string
		entries *Entries
		checks  []checkEntriesFn
		before  func(*Entries)
	}{
		{
			name:    "empty_list",
			entries: &Entries{List: []score.CRAPEntry{}},
			checks: checkEntries(
				checkEntry(
					checkEntriesLen(fn, 0),
				),
			),
		},
		{
			name:    "single_entry_unchanged",
			entries: &Entries{List: []score.CRAPEntry{{FuncName: "solo", EffectiveCRAP: 50}}},
			checks: checkEntries(
				checkEntry(
					checkEntriesLen(fn, 1),
					checkEntryName(fn, 0, "solo"),
					checkEntryEffectiveCRAP(fn, 0, 50),
				),
			),
		},
		{
			name: "already_sorted",
			entries: &Entries{List: []score.CRAPEntry{
				{FuncName: "high", EffectiveCRAP: 200},
				{FuncName: "mid", EffectiveCRAP: 100},
				{FuncName: "low", EffectiveCRAP: 50},
			}},
			checks: checkEntries(
				checkEntry(
					checkEntriesLen(fn, 3),
					checkEntryName(fn, 0, "high"),
					checkEntryName(fn, 1, "mid"),
					checkEntryName(fn, 2, "low"),
				),
			),
		},
		{
			name: "reverse_sorted",
			entries: &Entries{List: []score.CRAPEntry{
				{FuncName: "low", EffectiveCRAP: 50},
				{FuncName: "mid", EffectiveCRAP: 100},
				{FuncName: "high", EffectiveCRAP: 200},
			}},
			checks: checkEntries(
				checkEntry(
					checkEntriesLen(fn, 3),
					checkEntryName(fn, 0, "high"),
					checkEntryName(fn, 1, "mid"),
					checkEntryName(fn, 2, "low"),
				),
			),
		},
		{
			name: "random_order",
			entries: &Entries{List: []score.CRAPEntry{
				{FuncName: "mid", EffectiveCRAP: 100},
				{FuncName: "high", EffectiveCRAP: 200},
				{FuncName: "low", EffectiveCRAP: 50},
				{FuncName: "lowest", EffectiveCRAP: 10},
			}},
			checks: checkEntries(
				checkEntry(
					checkEntriesLen(fn, 4),
					checkEntryName(fn, 0, "high"),
					checkEntryName(fn, 1, "mid"),
					checkEntryName(fn, 2, "low"),
					checkEntryName(fn, 3, "lowest"),
				),
			),
		},
		{
			name: "equal_values_stable",
			entries: &Entries{List: []score.CRAPEntry{
				{FuncName: "same_a", EffectiveCRAP: 100},
				{FuncName: "same_b", EffectiveCRAP: 100},
				{FuncName: "same_c", EffectiveCRAP: 100},
			}},
			checks: checkEntries(
				checkEntry(
					checkEntriesLen(fn, 3),
					checkEntryName(fn, 0, "same_a"),
					checkEntryName(fn, 1, "same_b"),
					checkEntryName(fn, 2, "same_c"),
				),
			),
		},
		{
			name: "mix_of_equal_and_different",
			entries: &Entries{List: []score.CRAPEntry{
				{FuncName: "low", EffectiveCRAP: 10},
				{FuncName: "mid_a", EffectiveCRAP: 100},
				{FuncName: "high", EffectiveCRAP: 200},
				{FuncName: "mid_b", EffectiveCRAP: 100},
				{FuncName: "low_b", EffectiveCRAP: 10},
			}},
			checks: checkEntries(
				checkEntry(
					checkEntriesLen(fn, 5),
					checkEntryName(fn, 0, "high"),
					checkEntryName(fn, 1, "mid_a"),
					checkEntryName(fn, 2, "mid_b"),
					checkEntryName(fn, 3, "low"),
					checkEntryName(fn, 4, "low_b"),
				),
			),
		},
		{
			name: "zero_values",
			entries: &Entries{List: []score.CRAPEntry{
				{FuncName: "zero_b", EffectiveCRAP: 0},
				{FuncName: "zero_a", EffectiveCRAP: 0},
				{FuncName: "zero_c", EffectiveCRAP: 0},
			}},
			checks: checkEntries(
				checkEntry(
					checkEntriesLen(fn, 3),
					checkEntryName(fn, 0, "zero_b"),
					checkEntryName(fn, 1, "zero_a"),
					checkEntryName(fn, 2, "zero_c"),
				),
			),
		},
		{
			name: "negative_effective_crap",
			entries: &Entries{List: []score.CRAPEntry{
				{FuncName: "zero", EffectiveCRAP: 0},
				{FuncName: "negative", EffectiveCRAP: -50},
				{FuncName: "positive", EffectiveCRAP: 100},
			}},
			checks: checkEntries(
				checkEntry(
					checkEntriesLen(fn, 3),
					checkEntryName(fn, 0, "positive"),
					checkEntryName(fn, 1, "zero"),
					checkEntryName(fn, 2, "negative"),
				),
			),
		},
		{
			name: "large_list",
			entries: &Entries{List: []score.CRAPEntry{
				{FuncName: "f1", EffectiveCRAP: 50},
				{FuncName: "f2", EffectiveCRAP: 300},
				{FuncName: "f3", EffectiveCRAP: 150},
				{FuncName: "f4", EffectiveCRAP: 500},
				{FuncName: "f5", EffectiveCRAP: 25},
				{FuncName: "f6", EffectiveCRAP: 400},
				{FuncName: "f7", EffectiveCRAP: 200},
				{FuncName: "f8", EffectiveCRAP: 350},
				{FuncName: "f9", EffectiveCRAP: 75},
				{FuncName: "f10", EffectiveCRAP: 450},
			}},
			checks: checkEntries(
				checkEntry(
					checkEntriesLen(fn, 10),
					checkEntryName(fn, 0, "f4"),
					checkEntryName(fn, 1, "f10"),
					checkEntryName(fn, 2, "f6"),
					checkEntryName(fn, 3, "f8"),
					checkEntryName(fn, 4, "f2"),
					checkEntryName(fn, 5, "f7"),
					checkEntryName(fn, 6, "f3"),
					checkEntryName(fn, 7, "f9"),
					checkEntryName(fn, 8, "f1"),
					checkEntryName(fn, 9, "f5"),
				),
			),
		},
		{
			name: "two_elements",
			entries: &Entries{List: []score.CRAPEntry{
				{FuncName: "b", EffectiveCRAP: 10},
				{FuncName: "a", EffectiveCRAP: 100},
			}},
			checks: checkEntries(
				checkEntry(
					checkEntriesLen(fn, 2),
					checkEntryName(fn, 0, "a"),
					checkEntryName(fn, 1, "b"),
				),
			),
		},
		{
			name: "two_elements_same_value",
			entries: &Entries{List: []score.CRAPEntry{
				{FuncName: "x", EffectiveCRAP: 50},
				{FuncName: "y", EffectiveCRAP: 50},
			}},
			checks: checkEntries(
				checkEntry(
					checkEntriesLen(fn, 2),
					checkEntryName(fn, 0, "x"),
					checkEntryName(fn, 1, "y"),
				),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.before != nil {
				tt.before(tt.entries)
			}
			tt.entries.sort()
			for _, c := range tt.checks {
				c(t, tt.entries, nil)
			}
		})
	}
}

func TestEntries_shouldApplyMinFilter(t *testing.T) {
	tests := []struct {
		name        string
		min         float64
		expectApply bool
	}{
		{
			name:        "zero_no_filter",
			min:         0,
			expectApply: false,
		},
		{
			name:        "positive_threshold_filters",
			min:         1,
			expectApply: true,
		},
		{
			name:        "large_threshold_filters",
			min:         1000,
			expectApply: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Entries{options: &Options{Min: tt.min}}
			got := e.shouldApplyMinFilter()
			assert.Equal(t, tt.expectApply, got, "Min=%v: expectApply=%v", tt.min, tt.expectApply)
		})
	}
}

func TestEntries_shouldApplyTopFilter(t *testing.T) {
	tests := []struct {
		name        string
		top         int
		listLen     int
		expectApply bool
	}{
		{
			name:        "zero_no_filter",
			top:         0,
			listLen:     5,
			expectApply: false,
		},
		{
			name:        "less_than_len_filters",
			top:         2,
			listLen:     5,
			expectApply: true,
		},
		{
			name:        "equal_to_len_no_filter_boundary",
			top:         5,
			listLen:     5,
			expectApply: false,
		},
		{
			name:        "greater_than_len_no_filter",
			top:         10,
			listLen:     5,
			expectApply: false,
		},
		{
			name:        "empty_list_no_filter",
			top:         3,
			listLen:     0,
			expectApply: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := &Entries{
				options: &Options{Top: tt.top},
				List:    make([]score.CRAPEntry, tt.listLen),
			}
			for i := range e.List {
				e.List[i].FuncName = fmt.Sprintf("item%d", i)
				e.List[i].EffectiveCRAP = float64(i * 10)
			}
			got := e.shouldApplyTopFilter()
			assert.Equal(t, tt.expectApply, got, "Top=%d, len=%d: expectApply=%v", tt.top, tt.listLen, tt.expectApply)
		})
	}
}
