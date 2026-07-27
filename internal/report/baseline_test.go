package report

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/padiazg/go-crap/internal/scan"
	"github.com/padiazg/go-crap/internal/score"
)

func TestLoadBaseline_valid_json(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "baseline.json")

	jsonData := `{
		"$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		"version": "1.0.0",
		"entries": [
			{
				"file": "src/main.go",
				"function": "ProcessData",
				"line": 42,
				"package": "myapp",
				"cyclomatic": 5,
				"coverage": 80.0,
				"crap": 23.0,
				"effective_crap": 23.0
			},
			{
				"file": "src/utils.go",
				"function": "HelperFunc",
				"line": 10,
				"package": "myapp",
				"cyclomatic": 2,
				"coverage": 100.0,
				"crap": 2.0,
				"effective_crap": 2.0
			}
		]
	}`

	err := os.WriteFile(jsonPath, []byte(jsonData), 0644)
	require.NoError(t, err)

	baseline, err := LoadBaseline(jsonPath)
	require.NoError(t, err)

	assert.NotNil(t, baseline)
	assert.Len(t, baseline.entries, 2)

	entry, ok := baseline.Lookup("src/main.go", "ProcessData")
	assert.True(t, ok)
	assert.Equal(t, "ProcessData", entry.FuncName)
	assert.Equal(t, 23.0, entry.EffectiveCRAP)
	assert.Equal(t, 23.0, entry.CRAP)
	assert.Equal(t, 80.0, entry.Coverage)
	assert.Equal(t, 5, entry.Complexity)
	assert.Equal(t, 42, entry.Line)
	assert.Equal(t, "src/main.go", entry.File)
}

func TestLoadBaseline_nonexistent_file(t *testing.T) {
	_, err := LoadBaseline("/nonexistent/path/baseline.json")
	assert.Error(t, err)
}

func TestLoadBaseline_invalid_json(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "invalid.json")

	err := os.WriteFile(jsonPath, []byte("not valid json"), 0644)
	require.NoError(t, err)

	_, err = LoadBaseline(jsonPath)
	assert.Error(t, err)
}

func TestLoadBaseline_empty_entries(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "empty.json")

	jsonData := `{
		"$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		"version": "1.0.0",
		"entries": []
	}`

	err := os.WriteFile(jsonPath, []byte(jsonData), 0644)
	require.NoError(t, err)

	baseline, err := LoadBaseline(jsonPath)
	require.NoError(t, err)

	assert.NotNil(t, baseline)
	assert.Len(t, baseline.entries, 0)
	assert.Equal(t, 0.0, baseline.Summary.Combined)
	assert.Equal(t, 0.0, baseline.Summary.Average)
}

func TestLoadBaseline_null_coverage(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "null_cov.json")

	jsonData := `{
		"$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		"version": "1.0.0",
		"entries": [
			{
				"file": "src/main.go",
				"function": "NoCoverage",
				"line": 1,
				"package": "myapp",
				"cyclomatic": 3,
				"coverage": null,
				"crap": 12.0,
				"effective_crap": 12.0
			}
		]
	}`

	err := os.WriteFile(jsonPath, []byte(jsonData), 0644)
	require.NoError(t, err)

	baseline, err := LoadBaseline(jsonPath)
	require.NoError(t, err)

	entry, ok := baseline.Lookup("src/main.go", "NoCoverage")
	assert.True(t, ok)
	assert.Equal(t, 0.0, entry.Coverage)
	assert.Equal(t, 12.0, entry.EffectiveCRAP)
}

func TestBaselineLookup_not_found(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "baseline.json")

	jsonData := `{
		"$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		"version": "1.0.0",
		"entries": [
			{
				"file": "src/main.go",
				"function": "FuncA",
				"line": 1,
				"package": "myapp",
				"cyclomatic": 3,
				"coverage": 100.0,
				"crap": 3.0,
				"effective_crap": 3.0
			}
		]
	}`

	err := os.WriteFile(jsonPath, []byte(jsonData), 0644)
	require.NoError(t, err)

	baseline, err := LoadBaseline(jsonPath)
	require.NoError(t, err)

	_, ok := baseline.Lookup("src/main.go", "NonExistent")
	assert.False(t, ok)

	_, ok = baseline.Lookup("other/file.go", "FuncA")
	assert.False(t, ok)
}

func TestBaseline_compute_summary(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "baseline.json")

	jsonData := `{
		"$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		"version": "1.0.0",
		"entries": [
			{
				"file": "a.go",
				"function": "FuncA",
				"line": 1,
				"package": "pkg",
				"cyclomatic": 5,
				"coverage": 100.0,
				"crap": 5.0,
				"effective_crap": 5.0
			},
			{
				"file": "b.go",
				"function": "FuncB",
				"line": 10,
				"package": "pkg",
				"cyclomatic": 10,
				"coverage": 0.0,
				"crap": 110.0,
				"effective_crap": 110.0
			}
		]
	}`

	err := os.WriteFile(jsonPath, []byte(jsonData), 0644)
	require.NoError(t, err)

	baseline, err := LoadBaseline(jsonPath)
	require.NoError(t, err)

	assert.InDelta(t, 115.0, baseline.Summary.Combined, 0.01)
	assert.InDelta(t, 57.5, baseline.Summary.Average, 0.01)
	assert.Equal(t, 2, baseline.Summary.TotalFuncs)
}

func TestAnnotateWithBaseline_matched_entries(t *testing.T) {
	baseline := &Baseline{
		entries: map[string]BaselineEntry{
			"src/main.go:FuncA": {EffectiveCRAP: 50.0},
			"src/main.go:FuncB": {EffectiveCRAP: 20.0},
		},
	}

	entries := []score.CRAPEntry{
		{File: "src/main.go", FuncName: "FuncA", CRAP: 60.0, EffectiveCRAP: 60.0},
		{File: "src/main.go", FuncName: "FuncB", CRAP: 15.0, EffectiveCRAP: 15.0},
	}

	result := AnnotateWithBaseline(entries, baseline)

	assert.Equal(t, 50.0, result[0].BaselineCRAP)
	assert.Equal(t, 20.0, result[1].BaselineCRAP)
}

func TestAnnotateWithBaseline_unmatched_entries_new(t *testing.T) {
	baseline := &Baseline{
		entries: map[string]BaselineEntry{
			"src/main.go:OldFunc": {EffectiveCRAP: 30.0},
		},
	}

	entries := []score.CRAPEntry{
		{File: "src/main.go", FuncName: "NewFunc", CRAP: 10.0, EffectiveCRAP: 10.0},
	}

	result := AnnotateWithBaseline(entries, baseline)

	assert.Equal(t, -1.0, result[0].BaselineCRAP)
}

func TestAnnotateWithBaseline_nil_baseline(t *testing.T) {
	entries := []score.CRAPEntry{
		{File: "src/main.go", FuncName: "FuncA", CRAP: 10.0, EffectiveCRAP: 10.0},
	}

	result := AnnotateWithBaseline(entries, nil)

	assert.Equal(t, 0.0, result[0].BaselineCRAP)
	assert.Len(t, result, 1)
}

func TestAnnotateWithBaseline_multiple_mismatched(t *testing.T) {
	baseline := &Baseline{
		entries: map[string]BaselineEntry{
			"src/a.go:FuncA": {EffectiveCRAP: 10.0},
			"src/b.go:FuncB": {EffectiveCRAP: 20.0},
			"src/c.go:FuncC": {EffectiveCRAP: 30.0},
		},
	}

	entries := []score.CRAPEntry{
		{File: "src/a.go", FuncName: "FuncA", CRAP: 10.0, EffectiveCRAP: 10.0},
		{File: "src/b.go", FuncName: "FuncB", CRAP: 20.0, EffectiveCRAP: 20.0},
		{File: "src/x.go", FuncName: "NewFunc", CRAP: 5.0, EffectiveCRAP: 5.0},
		{File: "src/y.go", FuncName: "AnotherNew", CRAP: 8.0, EffectiveCRAP: 8.0},
	}

	result := AnnotateWithBaseline(entries, baseline)

	assert.Equal(t, 10.0, result[0].BaselineCRAP)
	assert.Equal(t, 20.0, result[1].BaselineCRAP)
	assert.Equal(t, -1.0, result[2].BaselineCRAP)
	assert.Equal(t, -1.0, result[3].BaselineCRAP)
}

func TestComputeSummaryWithBaseline_no_baseline(t *testing.T) {
	entries := &scan.Entries{
		List: []score.CRAPEntry{
			{File: "a.go", FuncName: "A", CRAP: 10.0, EffectiveCRAP: 10.0},
			{File: "b.go", FuncName: "B", CRAP: 20.0, EffectiveCRAP: 20.0},
		},
	}

	summary := ComputeSummaryWithBaseline(entries, 15.0, nil)

	assert.InDelta(t, 30.0, summary.Combined, 0.01)
	assert.InDelta(t, 15.0, summary.Average, 0.01)
	assert.Equal(t, 2, summary.TotalFuncs)
	assert.Equal(t, 1, summary.Exceeded)
	assert.Equal(t, 0.0, summary.BaselineCombined)
	assert.Equal(t, 0.0, summary.BaselineAverage)
	assert.Equal(t, 0.0, summary.DeltaCombined)
	assert.Equal(t, 0.0, summary.DeltaAverage)
}

func TestComputeSummaryWithBaseline_with_baseline(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "baseline.json")

	jsonData := `{
		"$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		"version": "1.0.0",
		"entries": [
			{
				"file": "a.go",
				"function": "A",
				"line": 1,
				"package": "pkg",
				"cyclomatic": 4,
				"coverage": 100.0,
				"crap": 4.0,
				"effective_crap": 8.0
			},
			{
				"file": "b.go",
				"function": "B",
				"line": 10,
				"package": "pkg",
				"cyclomatic": 10,
				"coverage": 0.0,
				"crap": 100.0,
				"effective_crap": 18.0
			}
		]
	}`

	err := os.WriteFile(jsonPath, []byte(jsonData), 0644)
	require.NoError(t, err)

	baseline, err := LoadBaseline(jsonPath)
	require.NoError(t, err)

	entries := &scan.Entries{
		List: []score.CRAPEntry{
			{File: "a.go", FuncName: "A", CRAP: 12.0, EffectiveCRAP: 12.0},
			{File: "b.go", FuncName: "B", CRAP: 18.0, EffectiveCRAP: 18.0},
		},
	}

	summary := ComputeSummaryWithBaseline(entries, 15.0, baseline)

	assert.InDelta(t, 30.0, summary.Combined, 0.01)
	assert.InDelta(t, 15.0, summary.Average, 0.01)
	assert.Equal(t, 2, summary.TotalFuncs)
	assert.Equal(t, 1, summary.Exceeded)
	assert.InDelta(t, 26.0, summary.BaselineCombined, 0.01)
	assert.InDelta(t, 13.0, summary.BaselineAverage, 0.01)
	assert.InDelta(t, 4.0, summary.DeltaCombined, 0.01)
	assert.InDelta(t, 2.0, summary.DeltaAverage, 0.01)
}

func TestComputeSummaryWithBaseline_regression(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "baseline.json")

	jsonData := `{
		"$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		"version": "1.0.0",
		"entries": [
			{
				"file": "main.go",
				"function": "FuncA",
				"line": 1,
				"package": "main",
				"cyclomatic": 2,
				"coverage": 100.0,
				"crap": 2.0,
				"effective_crap": 5.0
			}
		]
	}`

	err := os.WriteFile(jsonPath, []byte(jsonData), 0644)
	require.NoError(t, err)

	baseline, err := LoadBaseline(jsonPath)
	require.NoError(t, err)

	entries := &scan.Entries{
		List: []score.CRAPEntry{
			{File: "main.go", FuncName: "FuncA", CRAP: 50.0, EffectiveCRAP: 50.0},
		},
	}

	summary := ComputeSummaryWithBaseline(entries, 30.0, baseline)

	assert.InDelta(t, 50.0, summary.Combined, 0.01)
	assert.InDelta(t, 45.0, summary.DeltaCombined, 0.01)
	assert.True(t, summary.DeltaCombined > 0, "positive delta means regression")
}

func TestComputeSummaryWithBaseline_improvement(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "baseline.json")

	jsonData := `{
		"$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		"version": "1.0.0",
		"entries": [
			{
				"file": "main.go",
				"function": "FuncA",
				"line": 1,
				"package": "main",
				"cyclomatic": 10,
				"coverage": 0.0,
				"crap": 100.0,
				"effective_crap": 50.0
			}
		]
	}`

	err := os.WriteFile(jsonPath, []byte(jsonData), 0644)
	require.NoError(t, err)

	baseline, err := LoadBaseline(jsonPath)
	require.NoError(t, err)

	entries := &scan.Entries{
		List: []score.CRAPEntry{
			{File: "main.go", FuncName: "FuncA", CRAP: 5.0, EffectiveCRAP: 5.0},
		},
	}

	summary := ComputeSummaryWithBaseline(entries, 30.0, baseline)

	assert.InDelta(t, 5.0, summary.Combined, 0.01)
	assert.InDelta(t, -45.0, summary.DeltaCombined, 0.01)
	assert.True(t, summary.DeltaCombined < 0, "negative delta means improvement")
}

func TestBaselineEntries(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "baseline.json")

	jsonData := `{
		"$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		"version": "1.0.0",
		"entries": [
			{
				"file": "a.go",
				"function": "FuncA",
				"line": 1,
				"package": "pkg",
				"cyclomatic": 3,
				"coverage": 100.0,
				"crap": 3.0,
				"effective_crap": 3.0
			},
			{
				"file": "b.go",
				"function": "FuncB",
				"line": 2,
				"package": "pkg",
				"cyclomatic": 5,
				"coverage": 50.0,
				"crap": 25.0,
				"effective_crap": 25.0
			}
		]
	}`

	err := os.WriteFile(jsonPath, []byte(jsonData), 0644)
	require.NoError(t, err)

	baseline, err := LoadBaseline(jsonPath)
	require.NoError(t, err)

	entries := baseline.BaselineEntries()
	assert.Len(t, entries, 2)

	entryMap := make(map[string]BaselineEntry)
	for _, e := range entries {
		entryMap[e.FuncName] = e
	}

	assert.Contains(t, entryMap, "FuncA")
	assert.Contains(t, entryMap, "FuncB")
}

func TestAnnotateWithBaseline_preserves_original_fields(t *testing.T) {
	baseline := &Baseline{
		entries: map[string]BaselineEntry{
			"src/main.go:FuncA": {EffectiveCRAP: 10.0},
		},
	}

	entries := []score.CRAPEntry{
		{
			File:              "src/main.go",
			FuncName:          "FuncA",
			Package:           "mypkg",
			Receiver:          "MyRecv",
			CRAP:              50.0,
			EffectiveCRAP:     50.0,
			Complexity:        10,
			Coverage:          50.0,
			Line:              42,
			CoverageUntrusted: true,
			Skipped:           false,
		},
	}

	result := AnnotateWithBaseline(entries, baseline)

	assert.Equal(t, "mypkg", result[0].Package)
	assert.Equal(t, "MyRecv", result[0].Receiver)
	assert.Equal(t, 50.0, result[0].CRAP)
	assert.Equal(t, 50.0, result[0].EffectiveCRAP)
	assert.Equal(t, 10.0, result[0].BaselineCRAP)
	assert.Equal(t, 10, result[0].Complexity)
	assert.Equal(t, 50.0, result[0].Coverage)
	assert.Equal(t, 42, result[0].Line)
	assert.Equal(t, true, result[0].CoverageUntrusted)
	assert.Equal(t, false, result[0].Skipped)
}

func TestLoadBaseline_json_envelope_shape(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "baseline.json")

	jsonData := `{
		"$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		"version": "1.0.0",
		"entries": [
			{
				"file": "src/main.go",
				"function": "Main",
				"line": 1,
				"package": "main",
				"cyclomatic": 1,
				"coverage": 100.0,
				"crap": 1.0,
				"effective_crap": 1.0
			}
		]
	}`

	err := os.WriteFile(jsonPath, []byte(jsonData), 0644)
	require.NoError(t, err)

	baseline, err := LoadBaseline(jsonPath)
	require.NoError(t, err)

	assert.Equal(t, "Main", baseline.entries["src/main.go:Main"].FuncName)
	assert.Equal(t, "src/main.go", baseline.entries["src/main.go:Main"].File)
}

func TestBaseline_empty_baseline_entries_map(t *testing.T) {
	b := &Baseline{
		entries: make(map[string]BaselineEntry),
	}

	_, ok := b.Lookup("any.go", "AnyFunc")
	assert.False(t, ok)

	assert.Len(t, b.BaselineEntries(), 0)

	entries := []score.CRAPEntry{
		{File: "any.go", FuncName: "AnyFunc", CRAP: 5.0, EffectiveCRAP: 5.0},
	}

	result := AnnotateWithBaseline(entries, b)
	assert.Equal(t, -1.0, result[0].BaselineCRAP)
}

func TestComputeSummaryWithBaseline_empty_entries_with_baseline(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "baseline.json")

	jsonData := `{
		"$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		"version": "1.0.0",
		"entries": [
			{
				"file": "a.go",
				"function": "FuncA",
				"line": 1,
				"package": "pkg",
				"cyclomatic": 3,
				"coverage": 100.0,
				"crap": 3.0,
				"effective_crap": 3.0
			}
		]
	}`

	err := os.WriteFile(jsonPath, []byte(jsonData), 0644)
	require.NoError(t, err)

	baseline, err := LoadBaseline(jsonPath)
	require.NoError(t, err)

	entries := &scan.Entries{List: []score.CRAPEntry{}}

	summary := ComputeSummaryWithBaseline(entries, 30.0, baseline)

	assert.Equal(t, 0.0, summary.Combined)
	assert.Equal(t, 0.0, summary.Average)
	assert.Equal(t, 0, summary.TotalFuncs)
	assert.Equal(t, 0, summary.Exceeded)
	assert.InDelta(t, 3.0, summary.BaselineCombined, 0.01)
	assert.InDelta(t, -3.0, summary.DeltaCombined, 0.01)
}

func TestBaseline_entry_key_format(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "baseline.json")

	jsonData := `{
		"$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		"version": "1.0.0",
		"entries": [
			{
				"file": "src/pkg/file.go",
				"function": "FuncWithDots",
				"line": 42,
				"package": "pkg",
				"cyclomatic": 3,
				"coverage": 100.0,
				"crap": 3.0,
				"effective_crap": 3.0
			}
		]
	}`

	err := os.WriteFile(jsonPath, []byte(jsonData), 0644)
	require.NoError(t, err)

	baseline, err := LoadBaseline(jsonPath)
	require.NoError(t, err)

	_, ok := baseline.Lookup("src/pkg/file.go", "FuncWithDots")
	assert.True(t, ok)

	_, ok = baseline.Lookup("src/pkg/file.go:FuncWithDots", "")
	assert.False(t, ok)
}

func TestBaseline_multiple_functions_same_file(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "baseline.json")

	jsonData := `{
		"$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		"version": "1.0.0",
		"entries": [
			{
				"file": "main.go",
				"function": "FuncA",
				"line": 1,
				"package": "main",
				"cyclomatic": 2,
				"coverage": 100.0,
				"crap": 2.0,
				"effective_crap": 2.0
			},
			{
				"file": "main.go",
				"function": "FuncB",
				"line": 10,
				"package": "main",
				"cyclomatic": 4,
				"coverage": 50.0,
				"crap": 18.0,
				"effective_crap": 18.0
			},
			{
				"file": "main.go",
				"function": "FuncC",
				"line": 20,
				"package": "main",
				"cyclomatic": 6,
				"coverage": 0.0,
				"crap": 42.0,
				"effective_crap": 42.0
			}
		]
	}`

	err := os.WriteFile(jsonPath, []byte(jsonData), 0644)
	require.NoError(t, err)

	baseline, err := LoadBaseline(jsonPath)
	require.NoError(t, err)

	assert.Equal(t, 2.0, baseline.entries["main.go:FuncA"].EffectiveCRAP)
	assert.Equal(t, 18.0, baseline.entries["main.go:FuncB"].EffectiveCRAP)
	assert.Equal(t, 42.0, baseline.entries["main.go:FuncC"].EffectiveCRAP)

	assert.InDelta(t, 62.0, baseline.Summary.Combined, 0.01)
	assert.InDelta(t, 20.67, baseline.Summary.Average, 0.01)
	assert.Equal(t, 3, baseline.Summary.TotalFuncs)
}

func TestBaseline_json_with_receiver(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "baseline.json")

	jsonData := `{
		"$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		"version": "1.0.0",
		"entries": [
			{
				"file": "main.go",
				"function": "MethodFunc",
				"line": 10,
				"package": "main",
				"receiver": "MyType",
				"cyclomatic": 3,
				"coverage": 80.0,
				"crap": 13.5,
				"effective_crap": 13.5
			}
		]
	}`

	err := os.WriteFile(jsonPath, []byte(jsonData), 0644)
	require.NoError(t, err)

	baseline, err := LoadBaseline(jsonPath)
	require.NoError(t, err)

	entry, ok := baseline.Lookup("main.go", "MethodFunc")
	assert.True(t, ok)
	assert.Equal(t, 13.5, entry.EffectiveCRAP)
}

func TestBaseline_json_with_mutation_details(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "baseline.json")

	jsonData := `{
		"$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		"version": "1.0.0",
		"entries": [
			{
				"file": "main.go",
				"function": "FuncA",
				"line": 1,
				"package": "main",
				"cyclomatic": 3,
				"coverage": 100.0,
				"crap": 3.0,
				"effective_crap": 3.0,
				"mutation_score": 0.8
			}
		]
	}`

	err := os.WriteFile(jsonPath, []byte(jsonData), 0644)
	require.NoError(t, err)

	_, err = LoadBaseline(jsonPath)
	assert.NoError(t, err)
}

func TestBaseline_roundtrip(t *testing.T) {
	originalJSON := `{
		"$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		"version": "1.0.0",
		"entries": [
			{
				"file": "a.go",
				"function": "Foo",
				"line": 10,
				"package": "pkg",
				"receiver": "T",
				"cyclomatic": 5,
				"coverage": 80.0,
				"crap": 23.0,
				"effective_crap": 23.0,
				"mutation_score": 0.7
			}
		]
	}`

	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "baseline.json")
	err := os.WriteFile(jsonPath, []byte(originalJSON), 0644)
	require.NoError(t, err)

	baseline, err := LoadBaseline(jsonPath)
	require.NoError(t, err)

	entry, ok := baseline.Lookup("a.go", "Foo")
	assert.True(t, ok)
	assert.Equal(t, "Foo", entry.FuncName)
	assert.Equal(t, "a.go", entry.File)
	assert.Equal(t, 23.0, entry.CRAP)
	assert.Equal(t, 23.0, entry.EffectiveCRAP)
	assert.Equal(t, 5, entry.Complexity)
	assert.Equal(t, 80.0, entry.Coverage)
	assert.Equal(t, 10, entry.Line)
}

func BenchmarkLoadBaseline(b *testing.B) {
	jsonData := `{
		"$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		"version": "1.0.0",
		"entries": [
			{
				"file": "file.go",
				"function": "FuncA",
				"line": 1,
				"package": "pkg",
				"cyclomatic": 3,
				"coverage": 100.0,
				"crap": 3.0,
				"effective_crap": 3.0
			},
			{
				"file": "file.go",
				"function": "FuncB",
				"line": 10,
				"package": "pkg",
				"cyclomatic": 5,
				"coverage": 50.0,
				"crap": 25.0,
				"effective_crap": 25.0
			}
		]
	}`
	dir := b.TempDir()
	jsonPath := filepath.Join(dir, "baseline.json")
	err := os.WriteFile(jsonPath, []byte(jsonData), 0644)
	require.NoError(b, err)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = LoadBaseline(jsonPath)
	}
}

func TestLoadBaseline_json_roundtrip(t *testing.T) {
	dir := t.TempDir()
	jsonPath := filepath.Join(dir, "baseline.json")

	jsonData := `{
		"$schema": "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		"version": "1.0.0",
		"entries": [
			{
				"file": "src/main.go",
				"function": "ProcessData",
				"line": 42,
				"package": "myapp",
				"cyclomatic": 5,
				"coverage": 80.0,
				"crap": 23.0,
				"effective_crap": 23.0
			},
			{
				"file": "src/utils.go",
				"function": "HelperFunc",
				"line": 10,
				"package": "myapp",
				"cyclomatic": 2,
				"coverage": 100.0,
				"crap": 2.0,
				"effective_crap": 2.0
			}
		]
	}`

	err := os.WriteFile(jsonPath, []byte(jsonData), 0644)
	require.NoError(t, err)

	baseline, err := LoadBaseline(jsonPath)
	require.NoError(t, err)

	assert.Len(t, baseline.entries, 2)
	assert.InDelta(t, 25.0, baseline.Summary.Combined, 0.01)
	assert.InDelta(t, 12.5, baseline.Summary.Average, 0.01)

	entry, ok := baseline.Lookup("src/main.go", "ProcessData")
	assert.True(t, ok)
	assert.Equal(t, 23.0, entry.EffectiveCRAP)
}
