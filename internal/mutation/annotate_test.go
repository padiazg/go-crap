package mutation

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/padiazg/go-crap/internal/merge"
	"github.com/padiazg/go-crap/internal/score"
)

func TestParseReport(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		want    *Report
		wantErr bool
	}{
		{
			name:    "empty_path_returns_nil",
			setup:   func(t *testing.T) string { return "" },
			want:    nil,
			wantErr: false,
		},
		{
			name: "valid_json_nested_format",
			setup: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "report.json")
				data := `{
					"go_module": "github.com/example/test",
					"files": [
						{
							"file_name": "internal/score/score.go",
							"mutations": [
								{"type": "CONDITIONALS_BOUNDARY", "status": "LIVED", "line": 42},
								{"type": "ARITHMETIC", "status": "KILLED", "line": 43}
							]
						}
					],
					"mutants_killed": 1,
					"mutants_lived": 1,
					"mutants_not_covered": 0,
					"mutants_total": 2,
					"test_efficacy": 0.5
				}`
				err := os.WriteFile(path, []byte(data), 0644)
				require.NoError(t, err)
				return path
			},
			want: &Report{
				GoModule: "github.com/example/test",
				Mutants: []Mutant{
					{File: "internal/score/score.go", Line: 42, Type: "CONDITIONALS_BOUNDARY", Status: StatusLived},
					{File: "internal/score/score.go", Line: 43, Type: "ARITHMETIC", Status: StatusKilled},
				},
				MutantsKilled: 1,
				MutantsLived:  1,
				MutantsTotal:  2,
				TestEfficacy:  0.5,
			},
			wantErr: false,
		},
		{
			name: "malformed_json",
			setup: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "report.json")
				err := os.WriteFile(path, []byte("not json"), 0644)
				require.NoError(t, err)
				return path
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "nonexistent_file",
			setup: func(t *testing.T) string {
				return "/nonexistent/path/report.json"
			},
			want:    nil,
			wantErr: true,
		},
		{
			name: "empty_mutations",
			setup: func(t *testing.T) string {
				path := filepath.Join(t.TempDir(), "empty.json")
				data := `{"go_module": "test", "files": [], "mutants_killed": 0, "mutants_lived": 0, "mutants_not_covered": 0, "mutants_total": 0, "test_efficacy": 0}`
				err := os.WriteFile(path, []byte(data), 0644)
				require.NoError(t, err)
				return path
			},
			want: &Report{
				GoModule: "test",
				Mutants:  []Mutant{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := tt.setup(t)
			got, err := ParseReport(path)
			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, got)
				return
			}
			assert.NoError(t, err)
			if got == nil {
				assert.Nil(t, tt.want)
				return
			}
			assert.Equal(t, tt.want.GoModule, got.GoModule)
			assert.Equal(t, tt.want.MutantsKilled, got.MutantsKilled)
			assert.Equal(t, tt.want.MutantsLived, got.MutantsLived)
			assert.Equal(t, tt.want.MutantsTotal, got.MutantsTotal)
			assert.Equal(t, tt.want.TestEfficacy, got.TestEfficacy)
			assert.Equal(t, len(tt.want.Mutants), len(got.Mutants))
			for i, wantM := range tt.want.Mutants {
				assert.Equal(t, wantM.File, got.Mutants[i].File)
				assert.Equal(t, wantM.Line, got.Mutants[i].Line)
				assert.Equal(t, wantM.Type, got.Mutants[i].Type)
				assert.Equal(t, wantM.Status, got.Mutants[i].Status)
			}
		})
	}
}

func TestAnnotate(t *testing.T) {
	report := &Report{
		GoModule: "github.com/example/test",
		Files: []FileMutations{
			{
				FileName: "internal/score/score.go",
				Mutations: []Mutant{
					{Type: "ARITHMETIC", Status: StatusLived, Line: 50},
					{Type: "CONDITIONAL", Status: StatusKilled, Line: 45},
				},
			},
		},
		MutantsKilled: 1,
		MutantsLived:  1,
		Mutants: []Mutant{
			{File: "internal/score/score.go", Type: "ARITHMETIC", Status: StatusLived, Line: 50},
			{File: "internal/score/score.go", Type: "CONDITIONAL", Status: StatusKilled, Line: 45},
		},
	}

	reportWithAllKilled := &Report{
		GoModule: "github.com/example/test",
		Files: []FileMutations{
			{
				FileName: "internal/score/score.go",
				Mutations: []Mutant{
					{Type: "ARITHMETIC", Status: StatusKilled, Line: 50},
					{Type: "CONDITIONAL", Status: StatusKilled, Line: 45},
				},
			},
		},
		MutantsKilled: 2,
		MutantsLived:  0,
		Mutants: []Mutant{
			{File: "internal/score/score.go", Type: "ARITHMETIC", Status: StatusKilled, Line: 50},
			{File: "internal/score/score.go", Type: "CONDITIONAL", Status: StatusKilled, Line: 45},
		},
	}

	reportWithNoMutants := &Report{
		GoModule: "github.com/example/test",
		Files: []FileMutations{
			{
				FileName: "other/file.go",
				Mutations: []Mutant{
					{Type: "ARITHMETIC", Status: StatusLived, Line: 10},
				},
			},
		},
		MutantsKilled: 0,
		MutantsLived:  1,
		Mutants: []Mutant{
			{File: "other/file.go", Type: "ARITHMETIC", Status: StatusLived, Line: 10},
		},
	}

	tests := []struct {
		name     string
		entries  []score.CRAPEntry
		report   *Report
		merged   []merge.MergedEntry
		expected []score.CRAPEntry
	}{
		{
			name:    "nil_report_no_changes",
			entries: []score.CRAPEntry{{File: "a.go", FuncName: "Foo", Line: 1, Complexity: 5, Coverage: 80, CRAP: 50}},
			report:  nil,
			merged:  nil,
			expected: []score.CRAPEntry{
				{File: "a.go", FuncName: "Foo", Line: 1, Complexity: 5, Coverage: 80, CRAP: 50, EffectiveCRAP: 50},
			},
		},
		{
			name:    "skipped_entry_no_changes",
			entries: []score.CRAPEntry{{File: "a.go", FuncName: "Foo", Line: 1, Complexity: 5, Coverage: 80, CRAP: 50, Skipped: true}},
			report:  report,
			merged: []merge.MergedEntry{
				{File: "internal/score/score.go", FuncName: "Foo", Receiver: "", Line: 1, EndLine: 100, Complexity: 5},
			},
			expected: []score.CRAPEntry{
				{File: "a.go", FuncName: "Foo", Line: 1, Complexity: 5, Coverage: 80, CRAP: 50, Skipped: true, EffectiveCRAP: 50},
			},
		},
		{
			name:    "zero_coverage_no_changes",
			entries: []score.CRAPEntry{{File: "a.go", FuncName: "Foo", Line: 1, Complexity: 5, Coverage: 0, CRAP: 5}},
			report:  report,
			merged: []merge.MergedEntry{
				{File: "internal/score/score.go", FuncName: "Foo", Receiver: "", Line: 1, EndLine: 100, Complexity: 5},
			},
			expected: []score.CRAPEntry{
				{File: "a.go", FuncName: "Foo", Line: 1, Complexity: 5, Coverage: 0, CRAP: 5, EffectiveCRAP: 5},
			},
		},
		{
			name: "lived_mutant_marks_untrusted",
			entries: []score.CRAPEntry{
				{File: "internal/score/score.go", FuncName: "Bar", Receiver: "", Line: 10, Complexity: 5, Coverage: 80, CRAP: 30},
			},
			report: report,
			merged: []merge.MergedEntry{
				{File: "internal/score/score.go", FuncName: "Bar", Receiver: "", Line: 10, EndLine: 100, Complexity: 5},
			},
			expected: []score.CRAPEntry{
				{
					File: "internal/score/score.go", FuncName: "Bar", Receiver: "", Line: 10,
					Complexity: 5, Coverage: 80, CRAP: 30,
					CoverageUntrusted: true, MutationScore: 0.5,
					EffectiveCRAP: 30,
					MutationDetails: []score.MutationDetail{
						{MutantType: "ARITHMETIC", Line: 50, Status: "LIVED", File: "internal/score/score.go"},
					},
				},
			},
		},
		{
			name: "all_killed_no_untrusted",
			entries: []score.CRAPEntry{
				{File: "internal/score/score.go", FuncName: "Baz", Receiver: "", Line: 10, Complexity: 3, Coverage: 90, CRAP: 10},
			},
			report: reportWithAllKilled,
			merged: []merge.MergedEntry{
				{File: "internal/score/score.go", FuncName: "Baz", Receiver: "", Line: 10, EndLine: 100, Complexity: 3},
			},
			expected: []score.CRAPEntry{
				{
					File: "internal/score/score.go", FuncName: "Baz", Receiver: "", Line: 10,
					Complexity: 3, Coverage: 90, CRAP: 10,
					CoverageUntrusted: false, MutationScore: 1.0,
					EffectiveCRAP: 10,
				},
			},
		},
		{
			name: "no_mutants_for_file",
			entries: []score.CRAPEntry{
				{File: "internal/score/score.go", FuncName: "Qux", Receiver: "", Line: 1, Complexity: 5, Coverage: 80, CRAP: 50},
			},
			report: reportWithNoMutants,
			merged: []merge.MergedEntry{
				{File: "internal/score/score.go", FuncName: "Qux", Receiver: "", Line: 1, EndLine: 50, Complexity: 5},
			},
			expected: []score.CRAPEntry{
				{File: "internal/score/score.go", FuncName: "Qux", Receiver: "", Line: 1, Complexity: 5, Coverage: 80, CRAP: 50, EffectiveCRAP: 50},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := Annotate(tt.entries, tt.report, tt.merged)
			assert.Equal(t, len(tt.expected), len(result))
			for i, exp := range tt.expected {
				assert.Equal(t, exp.File, result[i].File)
				assert.Equal(t, exp.CoverageUntrusted, result[i].CoverageUntrusted, "CoverageUntrusted mismatch for %s", exp.FuncName)
				assert.InDelta(t, exp.MutationScore, result[i].MutationScore, 0.001, "MutationScore mismatch for %s", exp.FuncName)
				assert.InDelta(t, exp.EffectiveCRAP, result[i].EffectiveCRAP, 0.001, "EffectiveCRAP mismatch for %s", exp.FuncName)
				assert.Equal(t, len(exp.MutationDetails), len(result[i].MutationDetails), "MutationDetails count mismatch for %s", exp.FuncName)
				for j, md := range exp.MutationDetails {
					assert.Equal(t, md.MutantType, result[i].MutationDetails[j].MutantType, "MutationDetail type mismatch for %s", exp.FuncName)
					assert.Equal(t, md.Line, result[i].MutationDetails[j].Line, "MutationDetail line mismatch for %s", exp.FuncName)
					assert.Equal(t, md.Status, result[i].MutationDetails[j].Status, "MutationDetail status mismatch for %s", exp.FuncName)
					assert.Equal(t, md.File, result[i].MutationDetails[j].File, "MutationDetail file mismatch for %s", exp.FuncName)
				}
			}
		})
	}
}

func Test_mergeKey(t *testing.T) {
	tests := []struct {
		name     string
		file     string
		funcName string
		receiver string
		want     string
	}{
		{name: "no_receiver", file: "a.go", funcName: "Foo", receiver: "", want: "a.go::Foo"},
		{name: "with_receiver", file: "a.go", funcName: "Bar", receiver: "MyType", want: "a.go::MyType.Bar"},
		{name: "pointer_receiver", file: "b.go", funcName: "Baz", receiver: "*MyType", want: "b.go::*MyType.Baz"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mergeKey(tt.file, tt.funcName, tt.receiver)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_buildMutantsByFile(t *testing.T) {
	mutants := []Mutant{
		{File: "internal/score/score.go", Line: 10, Status: StatusKilled},
		{File: "internal/score/score.go", Line: 20, Status: StatusLived},
		{File: "other/other.go", Line: 5, Status: StatusKilled},
	}

	result := buildMutantsByFile(mutants)

	assert.Equal(t, 2, len(result))
	assert.Equal(t, 2, len(result["internal/score/score.go"]))
	assert.Equal(t, 1, len(result["other/other.go"]))
	assert.Equal(t, StatusKilled, result["internal/score/score.go"][0].Status)
	assert.Equal(t, StatusLived, result["internal/score/score.go"][1].Status)
}

func Test_buildMutantFileSuffix(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{name: "three_parts", path: "/home/user/project/internal/score/score.go", want: "internal/score/score.go"},
		{name: "two_parts", path: "score/score.go", want: "score/score.go"},
		{name: "one_part", path: "score.go", want: "score.go"},
		{name: "windows_path", path: "C:\\Users\\project\\internal\\score\\score.go", want: "internal/score/score.go"},
		{name: "exactly_three_parts", path: "a/b/c", want: "a/b/c"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildMutantFileSuffix(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestAnnotate_endline_fallback_when_key_missing(t *testing.T) {
	report := &Report{
		GoModule: "github.com/example/test",
		Mutants: []Mutant{
			{File: "internal/score/score.go", Type: "ARITHMETIC", Status: StatusLived, Line: 200},
		},
		MutantsLived: 1,
	}

	entries := []score.CRAPEntry{
		{File: "internal/score/score.go", FuncName: "NoMatch", Line: 10, Complexity: 5, Coverage: 80, CRAP: 30},
	}

	merged := []merge.MergedEntry{
		{File: "other/file.go", FuncName: "OtherFunc", Receiver: "", Line: 1, EndLine: 50, Complexity: 3},
	}

	result := Annotate(entries, report, merged)
	assert.Equal(t, 1, len(result))
	// Since endLineIdx has no key for this entry, endLine defaults to startLine + 100 = 110
	// The mutant at line 200 is NOT in range [10, 110], so no mutation is counted
	// The mutant file suffix "score.go" doesn't match "internal/score/score.go" → mutantsByFile lookup fails
	// So EffectiveCRAP stays unchanged
	assert.Equal(t, 30.0, result[0].EffectiveCRAP)
}

func TestAnnotate_mutation_score_boundary_3killed_1lived(t *testing.T) {
	report := &Report{
		GoModule: "github.com/example/test",
		Mutants: []Mutant{
			{File: "internal/score/score.go", Type: "COND1", Status: StatusKilled, Line: 50},
			{File: "internal/score/score.go", Type: "COND2", Status: StatusKilled, Line: 51},
			{File: "internal/score/score.go", Type: "COND3", Status: StatusKilled, Line: 52},
			{File: "internal/score/score.go", Type: "COND4", Status: StatusLived, Line: 53},
		},
		MutantsKilled: 3,
		MutantsLived:  1,
	}

	entries := []score.CRAPEntry{
		{File: "internal/score/score.go", FuncName: "Bar", Receiver: "", Line: 10, Complexity: 5, Coverage: 80, CRAP: 30},
	}

	merged := []merge.MergedEntry{
		{File: "internal/score/score.go", FuncName: "Bar", Receiver: "", Line: 10, EndLine: 100, Complexity: 5},
	}

	result := Annotate(entries, report, merged)
	assert.Equal(t, 1, len(result))
	assert.True(t, result[0].CoverageUntrusted)
	assert.InDelta(t, 0.75, result[0].MutationScore, 0.001, "MutationScore should be 3/(3+1) = 0.75")
	assert.Equal(t, 30.0, result[0].EffectiveCRAP) // CRAP(5, 0) = 25... wait
}

func TestAnnotate_mutation_score_boundary_exact_half(t *testing.T) {
	report := &Report{
		GoModule: "github.com/example/test",
		Mutants: []Mutant{
			{File: "internal/score/score.go", Type: "COND1", Status: StatusKilled, Line: 50},
			{File: "internal/score/score.go", Type: "COND2", Status: StatusLived, Line: 51},
		},
		MutantsKilled: 1,
		MutantsLived:  1,
	}

	entries := []score.CRAPEntry{
		{File: "internal/score/score.go", FuncName: "HalfScore", Receiver: "", Line: 10, Complexity: 5, Coverage: 80, CRAP: 30},
	}

	merged := []merge.MergedEntry{
		{File: "internal/score/score.go", FuncName: "HalfScore", Receiver: "", Line: 10, EndLine: 100, Complexity: 5},
	}

	result := Annotate(entries, report, merged)
	assert.Equal(t, 1, len(result))
	assert.True(t, result[0].CoverageUntrusted)
	assert.InDelta(t, 0.5, result[0].MutationScore, 0.001, "MutationScore should be 1/(1+1) = 0.5")
}

func TestAnnotate_no_mutants_in_range(t *testing.T) {
	report := &Report{
		GoModule: "github.com/example/test",
		Mutants: []Mutant{
			{File: "internal/score/score.go", Type: "COND1", Status: StatusLived, Line: 500},
		},
		MutantsLived: 1,
	}

	entries := []score.CRAPEntry{
		{File: "internal/score/score.go", FuncName: "InRange", Receiver: "", Line: 10, Complexity: 5, Coverage: 80, CRAP: 30},
	}

	merged := []merge.MergedEntry{
		{File: "internal/score/score.go", FuncName: "InRange", Receiver: "", Line: 10, EndLine: 100, Complexity: 5},
	}

	result := Annotate(entries, report, merged)
	// Mutant at line 500 is outside the range [10, 100], so no mutation is counted
	// No mutants in range → same as "no_mutants_for_file" → EffectiveCRAP unchanged
	assert.Equal(t, 30.0, result[0].EffectiveCRAP)
	assert.False(t, result[0].CoverageUntrusted)
}

func TestAnnotate_endline_boundary_exact_match(t *testing.T) {
	report := &Report{
		GoModule: "github.com/example/test",
		Mutants: []Mutant{
			{File: "internal/score/score.go", Type: "COND1", Status: StatusLived, Line: 100},
			{File: "internal/score/score.go", Type: "COND2", Status: StatusKilled, Line: 50},
		},
		MutantsLived: 1,
	}

	entries := []score.CRAPEntry{
		{File: "internal/score/score.go", FuncName: "Boundary", Receiver: "", Line: 10, Complexity: 5, Coverage: 80, CRAP: 30},
	}

	merged := []merge.MergedEntry{
		{File: "internal/score/score.go", FuncName: "Boundary", Receiver: "", Line: 10, EndLine: 100, Complexity: 5},
	}

	result := Annotate(entries, report, merged)
	// Mutant at line 100 == EndLine 100 → should be included (line >= startLine && line <= endLine)
	assert.True(t, result[0].CoverageUntrusted)
	assert.InDelta(t, 0.5, result[0].MutationScore, 0.001)
}

func TestAnnotate_endline_boundary_one_past(t *testing.T) {
	report := &Report{
		GoModule: "github.com/example/test",
		Mutants: []Mutant{
			{File: "internal/score/score.go", Type: "COND1", Status: StatusLived, Line: 101},
			{File: "internal/score/score.go", Type: "COND2", Status: StatusKilled, Line: 50},
		},
		MutantsLived: 1,
	}

	entries := []score.CRAPEntry{
		{File: "internal/score/score.go", FuncName: "Boundary", Receiver: "", Line: 10, Complexity: 5, Coverage: 80, CRAP: 30},
	}

	merged := []merge.MergedEntry{
		{File: "internal/score/score.go", FuncName: "Boundary", Receiver: "", Line: 10, EndLine: 100, Complexity: 5},
	}

	result := Annotate(entries, report, merged)
	// Mutant at line 101 > EndLine 100 → should NOT be included
	assert.False(t, result[0].CoverageUntrusted)
	assert.Equal(t, 30.0, result[0].EffectiveCRAP)
}

func Test_resolveEndLine_fallback(t *testing.T) {
	endLineIdx := map[string]int{
		"some/key": 50,
	}

	// Key exists, endLine > startLine → return endLine
	got := resolveEndLine(endLineIdx, "some/key", 10)
	assert.Equal(t, 50, got)

	// Key exists, endLine < startLine → return startLine + 100
	got = resolveEndLine(endLineIdx, "some/key", 60)
	assert.Equal(t, 160, got)

	// Key doesn't exist → return startLine + 100
	got = resolveEndLine(endLineIdx, "unknown/key", 10)
	assert.Equal(t, 110, got)
}

func Test_resolveEndLine_boundary_equal(t *testing.T) {
	endLineIdx := map[string]int{
		"some/key": 50,
	}

	// endLine == startLine → return endLine (no fallback needed)
	got := resolveEndLine(endLineIdx, "some/key", 50)
	assert.Equal(t, 50, got)
}

func Test_classifyMutants_boundary_inclusive(t *testing.T) {
	mutants := []Mutant{
		{Line: 10, Status: StatusLived},
		{Line: 100, Status: StatusKilled},
		{Line: 101, Status: StatusLived},
		{Line: 9, Status: StatusLived},
		{Line: 50, Status: StatusKilled},
	}

	killed, lived, livedMutants := classifyMutants(mutants, 10, 100)
	assert.Equal(t, 2, killed)
	assert.Equal(t, 1, lived)
	assert.Equal(t, 1, len(livedMutants))
	assert.Equal(t, StatusLived, livedMutants[0].Status)
}

func Test_classifyMutants_boundary(t *testing.T) {
	mutants := []Mutant{
		{Line: 10, Status: StatusLived},   // startLine = 10, should be included
		{Line: 100, Status: StatusKilled}, // endLine = 100, should be included
		{Line: 101, Status: StatusLived},  // past endLine, should NOT be included
		{Line: 9, Status: StatusLived},    // before startLine, should NOT be included
		{Line: 50, Status: StatusKilled},  // in range
	}

	killed, lived, livedMutants := classifyMutants(mutants, 10, 100)
	assert.Equal(t, 2, killed, "2 mutants should be killed (lines 100 and 50)")
	assert.Equal(t, 1, lived, "1 mutant should be lived (line 10)")
	assert.Equal(t, 1, len(livedMutants))
	assert.Equal(t, StatusLived, livedMutants[0].Status)
}

type annotateEntryCheckFn func(*testing.T, *score.CRAPEntry)

var checkannotateEntry = func(fns ...annotateEntryCheckFn) []annotateEntryCheckFn { return fns }

func checkCoverageUntrusted(want bool) annotateEntryCheckFn {
	return func(t *testing.T, e *score.CRAPEntry) {
		t.Helper()
		assert.Equal(t, want, e.CoverageUntrusted, "CoverageUntrusted mismatch")
	}
}

func checkMutationScore(want float64) annotateEntryCheckFn {
	return func(t *testing.T, e *score.CRAPEntry) {
		t.Helper()
		assert.InDelta(t, want, e.MutationScore, 0.001, "MutationScore mismatch")
	}
}

func checkEffectiveCRAP(want float64) annotateEntryCheckFn {
	return func(t *testing.T, e *score.CRAPEntry) {
		t.Helper()
		assert.InDelta(t, want, e.EffectiveCRAP, 0.001, "EffectiveCRAP mismatch")
	}
}

func checkMutationDetailsLen(want int) annotateEntryCheckFn {
	return func(t *testing.T, e *score.CRAPEntry) {
		t.Helper()
		assert.Equal(t, want, len(e.MutationDetails), "MutationDetails count mismatch")
	}
}

func checkMutationDetailsNil(want bool) annotateEntryCheckFn {
	return func(t *testing.T, e *score.CRAPEntry) {
		t.Helper()
		if want {
			assert.Nilf(t, e.MutationDetails, "MutationDetails should be nil")
		} else {
			assert.NotNilf(t, e.MutationDetails, "MutationDetails should'nt be nil")
		}

	}
}

func Test_annotateEntry(t *testing.T) {
	tests := []struct {
		name         string
		e            *score.CRAPEntry
		killed       int
		lived        int
		livedMutants []Mutant
		checks       []annotateEntryCheckFn
	}{
		{
			name:   "lived_killed_equal",
			e:      &score.CRAPEntry{File: "a.go", FuncName: "Foo", Complexity: 5, Coverage: 80, CRAP: score.CRAP(5, 80)},
			killed: 1,
			lived:  1,
			livedMutants: []Mutant{{
				Type:            "COND",
				Line:            10,
				Status:          StatusLived,
				OriginalCode:    "x > 0",
				ReplacementCode: "x <= 0",
			}},
			checks: checkannotateEntry(
				checkCoverageUntrusted(true),
				checkMutationScore(0.5),
				checkEffectiveCRAP(score.CRAP(5, 0)),
				checkMutationDetailsLen(1),
			),
		},
		{
			name:         "all_killed",
			e:            &score.CRAPEntry{File: "a.go", FuncName: "Foo", Complexity: 5, Coverage: 80, CRAP: 30},
			killed:       3,
			lived:        0,
			livedMutants: nil,
			checks: checkannotateEntry(
				checkCoverageUntrusted(false),
				checkMutationScore(1.0),
				checkEffectiveCRAP(30),
				checkMutationDetailsLen(0),
				checkMutationDetailsNil(true),
			),
		},
		{
			name:         "no_mutants",
			e:            &score.CRAPEntry{File: "a.go", FuncName: "Foo", Complexity: 5, Coverage: 80, CRAP: 30},
			killed:       0,
			lived:        0,
			livedMutants: nil,
			checks: checkannotateEntry(
				checkCoverageUntrusted(false),
				checkMutationScore(-1),
				checkEffectiveCRAP(30),
				checkMutationDetailsLen(0),
			),
		},
		{
			name:   "all_lived_no_killed",
			e:      &score.CRAPEntry{File: "b.go", FuncName: "Bar", Complexity: 3, Coverage: 60, CRAP: score.CRAP(3, 60)},
			killed: 0,
			lived:  2,
			livedMutants: []Mutant{
				{Type: "COND", Line: 5, Status: StatusLived},
				{Type: "ARITHMETIC", Line: 8, Status: StatusLived},
			},
			checks: checkannotateEntry(
				checkCoverageUntrusted(true),
				checkMutationScore(0),
				checkEffectiveCRAP(score.CRAP(3, 0)),
				checkMutationDetailsLen(2),
			),
		},
		{
			name:   "most_killed_one_lived",
			e:      &score.CRAPEntry{File: "c.go", FuncName: "Baz", Complexity: 10, Coverage: 90, CRAP: 10.01},
			killed: 9,
			lived:  1,
			livedMutants: []Mutant{{
				Type:   "COND",
				Line:   20,
				Status: StatusLived,
			}},
			checks: checkannotateEntry(
				checkCoverageUntrusted(true),
				checkMutationScore(0.9),
				checkEffectiveCRAP(score.CRAP(10, 0)),
				checkMutationDetailsLen(1),
			),
		},
		{
			name:   "high_complexity",
			e:      &score.CRAPEntry{File: "d.go", FuncName: "Qux", Complexity: 50, Coverage: 100, CRAP: 2501},
			killed: 5,
			lived:  5,
			livedMutants: []Mutant{{
				Type:   "COND",
				Line:   30,
				Status: StatusLived,
			}},
			checks: checkannotateEntry(
				checkCoverageUntrusted(true),
				checkMutationScore(0.5),
				checkEffectiveCRAP(score.CRAP(50, 0)),
				checkMutationDetailsLen(1),
			),
		},
		{
			name:   "complexity_one",
			e:      &score.CRAPEntry{File: "e.go", FuncName: "Simple", Complexity: 1, Coverage: 50, CRAP: score.CRAP(1, 50)},
			killed: 0,
			lived:  1,
			livedMutants: []Mutant{{
				Type:   "COND",
				Line:   1,
				Status: StatusLived,
			}},
			checks: checkannotateEntry(
				checkCoverageUntrusted(true),
				checkMutationScore(0),
				checkEffectiveCRAP(score.CRAP(1, 0)),
				checkMutationDetailsLen(1),
			),
		},
		{
			name:   "mutation_details_populated",
			e:      &score.CRAPEntry{File: "f.go", FuncName: "Details", Complexity: 2, Coverage: 40, CRAP: score.CRAP(2, 40)},
			killed: 2,
			lived:  1,
			livedMutants: []Mutant{
				{
					Type:            "CONDITIONALS_BOUNDARY",
					MutatorName:     "conditional1",
					Line:            15,
					Status:          StatusLived,
					OriginalCode:    "x > y",
					ReplacementCode: "x >= y",
				},
			},
			checks: checkannotateEntry(
				checkCoverageUntrusted(true),
				checkMutationScore(2.0/3.0),
				checkEffectiveCRAP(score.CRAP(2, 0)),
				checkMutationDetailsLen(1),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			annotateEntry(tt.e, tt.killed, tt.lived, tt.livedMutants)
			for _, c := range tt.checks {
				c(t, tt.e)
			}
		})
	}
}
