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
		tt := tt
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
		tt := tt
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
		tt := tt
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
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := buildMutantFileSuffix(tt.path)
			assert.Equal(t, tt.want, got)
		})
	}
}
