package report

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/padiazg/go-crap/internal/scan"
	"github.com/padiazg/go-crap/internal/score"
)

type checkSARIFFormatterOutputFn func(*testing.T, string)

var checkSARIFFormatterOutput = func(fns ...checkSARIFFormatterOutputFn) []checkSARIFFormatterOutputFn {
	return fns
}

func checkSarifOutputContains(want string) checkSARIFFormatterOutputFn {
	return func(t *testing.T, got string) {
		t.Helper()
		assert.Containsf(t, got, want, "output should contain %q", want)
	}
}

func checkSarifOutputNotContains(want string) checkSARIFFormatterOutputFn {
	return func(t *testing.T, got string) {
		t.Helper()
		assert.NotContainsf(t, got, want, "output should not contain %q", want)
	}
}

func TestSARIFFormatter_Format(t *testing.T) {
	tests := []struct {
		name    string
		entries *scan.Entries
		opts    FormatOptions
		checks  []checkSARIFFormatterOutputFn
	}{
		{
			name:    "success_empty_entries",
			entries: &scan.Entries{List: []score.CRAPEntry{}},
			checks: checkSARIFFormatterOutput(
				checkSarifOutputContains(`"$schema"`),
				checkSarifOutputContains(`"version"`),
				checkSarifOutputContains(`"runs"`),
				checkSarifOutputContains(`"results"`),
			),
		},
		{
			name: "success_all_below_threshold",
			entries: &scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Good", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1},
				{File: "/project/main.go", Package: "myapp", FuncName: "OK", Line: 10, Complexity: 2, Coverage: 80, CRAP: 5},
			}},
			opts: FormatOptions{Threshold: 200},
			checks: checkSARIFFormatterOutput(
				checkSarifOutputContains(`"results": []`),
			),
		},
		{
			name: "success_above_threshold",
			entries: &scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Bad", Line: 42, Complexity: 10, Coverage: 0, CRAP: 110},
			}},
			opts: FormatOptions{Threshold: 30, BaseDir: "/project"},
			checks: checkSARIFFormatterOutput(
				checkSarifOutputContains(`"ruleId": "crap/high-score"`),
				checkSarifOutputContains(`"level": "warning"`),
				checkSarifOutputContains(`"startLine": 42`),
				checkSarifOutputContains(`"uri": "main.go"`),
				checkSarifOutputContains(`"text": "Function Bad has CRAP score 110.0 (cyclomatic complexity 10, coverage 0.0%)"`),
			),
		},
		{
			name: "success_mixed_entries_filter",
			entries: &scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Good", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1},
				{File: "/project/main.go", Package: "myapp", FuncName: "Bad", Line: 10, Complexity: 10, Coverage: 0, CRAP: 110},
				{File: "/project/main.go", Package: "myapp", FuncName: "OK", Line: 20, Complexity: 3, Coverage: 50, CRAP: 13.5},
			}},
			opts: FormatOptions{Threshold: 30},
			checks: checkSARIFFormatterOutput(
				checkSarifOutputContains(`"text": "Function Bad has CRAP score 110.0`),
				checkSarifOutputNotContains(`"text": "Function Good has CRAP score`),
			),
		},
		{
			name: "success_receiver_in_message",
			entries: &scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Process", Receiver: "User", Line: 5, Complexity: 5, Coverage: 0, CRAP: 30},
			}},
			opts: FormatOptions{Threshold: 20},
			checks: checkSARIFFormatterOutput(
				checkSarifOutputContains(`"text": "Function User.Process has CRAP score 30.0`),
			),
		},
		{
			name: "success_path_normalization",
			entries: &scan.Entries{List: []score.CRAPEntry{
				{File: "C:\\Users\\test\\main.go", Package: "myapp", FuncName: "WinFunc", Line: 1, Complexity: 1, Coverage: 0, CRAP: 2},
			}},
			opts: FormatOptions{Threshold: 0},
			checks: checkSARIFFormatterOutput(
				checkSarifOutputContains(`"uri": "C:/Users/test/main.go"`),
				checkSarifOutputNotContains(`\`),
			),
		},
		{
			name: "success_schema_and_version",
			entries: &scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "A", Line: 1, Complexity: 1, Coverage: 0, CRAP: 2},
			}},
			opts: FormatOptions{Threshold: 0},
			checks: checkSARIFFormatterOutput(
				checkSarifOutputContains(`"$schema": "https://json.schemastore.org/sarif-2.1.0.json"`),
				checkSarifOutputContains(`"version": "2.1.0"`),
			),
		},
		{
			name: "success_base_dir_relativize",
			entries: &scan.Entries{List: []score.CRAPEntry{
				{File: "/tmp/project/main.go", Package: "myapp", FuncName: "Process", Line: 5, Complexity: 1, Coverage: 0, CRAP: 1},
			}},
			opts: FormatOptions{Threshold: 0, BaseDir: "/tmp/project"},
			checks: checkSARIFFormatterOutput(
				checkSarifOutputContains(`"uri": "main.go"`),
			),
		},
		{
			name: "success_tool_driver_name",
			entries: &scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "X", Line: 1, Complexity: 1, Coverage: 0, CRAP: 2},
			}},
			opts: FormatOptions{Threshold: 0},
			checks: checkSARIFFormatterOutput(
				checkSarifOutputContains(`"name": "go-crap"`),
				checkSarifOutputContains(`"informationUri": "https://github.com/padiazg/go-crap"`),
			),
		},
		{
			name: "success_single_rule_defined",
			entries: &scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "X", Line: 1, Complexity: 1, Coverage: 0, CRAP: 2},
			}},
			opts: FormatOptions{Threshold: 0},
			checks: checkSARIFFormatterOutput(
				checkSarifOutputContains(`"id": "crap/high-score"`),
				checkSarifOutputContains(`"CRAP score exceeds threshold"`),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &SARIFFormatter{}
			buf := &bytes.Buffer{}
			opts := tt.opts
			opts.Writer = buf

			err := f.Format(tt.entries, opts)
			require.NoError(t, err, "Format should not return an error")

			for _, c := range tt.checks {
				c(t, buf.String())
			}
		})
	}
}

func TestSARIFFormatter_Format_validates_json_output(t *testing.T) {
	f := &SARIFFormatter{}
	buf := &bytes.Buffer{}

	entries := &scan.Entries{List: []score.CRAPEntry{
		{File: "/project/main.go", Package: "myapp", FuncName: "Bad", Receiver: "User", Line: 42, Complexity: 10, Coverage: 0, CRAP: 110},
		{File: "/project/main.go", Package: "myapp", FuncName: "OK", Line: 10, Complexity: 2, Coverage: 80, CRAP: 5},
	}}

	opts := FormatOptions{
		Threshold: 30,
		Writer:    buf,
		BaseDir:   "/project",
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "2.1.0", parsed["version"])

	runs := parsed["runs"].([]any)
	require.Len(t, runs, 1)

	run := runs[0].(map[string]any)
	tool := run["tool"].(map[string]any)
	driver := tool["driver"].(map[string]any)
	assert.Equal(t, "go-crap", driver["name"])

	results := run["results"].([]any)
	require.Len(t, results, 1)

	result := results[0].(map[string]any)
	assert.Equal(t, "crap/high-score", result["ruleId"])
	assert.Equal(t, "warning", result["level"])

	locs := result["locations"].([]any)
	require.Len(t, locs, 1)
	loc := locs[0].(map[string]any)
	physLoc := loc["physicalLocation"].(map[string]any)
	artLoc := physLoc["artifactLocation"].(map[string]any)
	assert.Equal(t, "main.go", artLoc["uri"])

	region := physLoc["region"].(map[string]any)
	assert.Equal(t, float64(42), region["startLine"])
}

func TestSARIFFormatter_Format_empty_results_valid_sarif(t *testing.T) {
	f := &SARIFFormatter{}
	buf := &bytes.Buffer{}

	entries := &scan.Entries{List: []score.CRAPEntry{
		{File: "/project/main.go", Package: "myapp", FuncName: "Good", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1},
	}}

	opts := FormatOptions{
		Threshold: 200,
		Writer:    buf,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	runs := parsed["runs"].([]any)
	run := runs[0].(map[string]any)
	results := run["results"].([]any)
	assert.Len(t, results, 0)
}

func TestSARIFFormatter_Format_path_to_slash_normalization(t *testing.T) {
	f := &SARIFFormatter{}
	buf := &bytes.Buffer{}

	entries := &scan.Entries{List: []score.CRAPEntry{
		{File: "C:\\Users\\test\\file.go", Package: "myapp", FuncName: "WinFunc", Line: 10, Complexity: 1, Coverage: 0, CRAP: 2},
	}}

	opts := FormatOptions{
		Threshold: 0,
		Writer:    buf,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	assert.Contains(t, buf.String(), `"uri": "C:/Users/test/file.go"`)
	assert.NotContains(t, buf.String(), `\`)
}

func TestRelativizePath(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		baseDir string
		want    string
	}{
		{
			name:    "same_base_dir",
			path:    "/tmp/project/main.go",
			baseDir: "/tmp/project",
			want:    "main.go",
		},
		{
			name:    "no_match_base_dir",
			path:    "/other/project/main.go",
			baseDir: "/tmp/project",
			want:    "../../other/project/main.go",
		},
		{
			name:    "empty_base_dir",
			path:    "/project/main.go",
			baseDir: "",
			want:    "/project/main.go",
		},
		{
			name:    "windows_to_unix",
			path:    "C:\\Users\\test\\file.go",
			baseDir: "",
			want:    "C:/Users/test/file.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := relativizePath(tt.path, tt.baseDir)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestFormatMessage(t *testing.T) {
	tests := []struct {
		name          string
		entry         score.CRAPEntry
		effectiveCRAP float64
		wantMsg       string
	}{
		{
			name:          "simple_function",
			entry:         score.CRAPEntry{FuncName: "Process", Complexity: 10, Coverage: 50, CRAP: 60},
			effectiveCRAP: 60,
			wantMsg:       "Function Process has CRAP score 60.0 (cyclomatic complexity 10, coverage 50.0%)",
		},
		{
			name:          "with_receiver",
			entry:         score.CRAPEntry{Receiver: "User", FuncName: "Process", Complexity: 3, Coverage: 0, CRAP: 12},
			effectiveCRAP: 12,
			wantMsg:       "Function User.Process has CRAP score 12.0 (cyclomatic complexity 3, coverage 0.0%)",
		},
		{
			name:          "unreliable_coverage",
			entry:         score.CRAPEntry{FuncName: "Bad", Complexity: 10, Coverage: 90, CRAP: 50, CoverageUntrusted: true, MutationScore: 0.3},
			effectiveCRAP: 0,
			wantMsg:       "Function Bad has CRAP score 0.0 (cyclomatic complexity 10, coverage 90.0%) [coverage not reliable (mutation score: 30.0%)]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatMessage(tt.entry, tt.effectiveCRAP, false)
			assert.Equal(t, tt.wantMsg, got)
		})
	}
}

func TestStatusSymbol(t *testing.T) {
	tests := []struct {
		name      string
		crap      float64
		threshold float64
		want      string
	}{
		{"below_half_threshold", 5, 20, "✓"},
		{"at_half_threshold", 10, 20, "✓"},
		{"above_half_below_threshold", 15, 20, "▲"},
		{"at_threshold", 20, 20, "▲"},
		{"above_threshold", 21, 20, "✗"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StatusSymbol(tt.crap, tt.threshold, tt.threshold/2.0)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSARIFFormatter_Format_nil_entries(t *testing.T) {
	f := &SARIFFormatter{}
	buf := &bytes.Buffer{}

	err := f.Format(nil, FormatOptions{Writer: buf})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestSARIFFormatter_Format_multiple_results_ordering(t *testing.T) {
	f := &SARIFFormatter{}
	buf := &bytes.Buffer{}

	entries := &scan.Entries{List: []score.CRAPEntry{
		{File: "/project/a.go", Package: "myapp", FuncName: "Low", Line: 1, Complexity: 3, Coverage: 0, CRAP: 12},
		{File: "/project/b.go", Package: "myapp", FuncName: "High", Line: 2, Complexity: 10, Coverage: 0, CRAP: 110},
		{File: "/project/c.go", Package: "myapp", FuncName: "Mid", Line: 3, Complexity: 5, Coverage: 10, CRAP: 45},
	}}

	opts := FormatOptions{
		Threshold: 10,
		Writer:    buf,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	runs := parsed["runs"].([]any)
	run := runs[0].(map[string]any)
	results := run["results"].([]any)

	require.Len(t, results, 3)

	// SARIF doesn't require ordering, but we verify all are present
	found := make(map[int]bool)
	for _, r := range results {
		result := r.(map[string]any)
		locs := result["locations"].([]any)
		loc := locs[0].(map[string]any)
		region := loc["physicalLocation"].(map[string]any)
		regionMap := region["region"].(map[string]any)
		line := int(regionMap["startLine"].(float64))
		found[line] = true
	}
	assert.True(t, found[1])
	assert.True(t, found[2])
	assert.True(t, found[3])
}

func TestSARIFFormatter_Format_schema_url(t *testing.T) {
	f := &SARIFFormatter{}
	buf := &bytes.Buffer{}

	entries := &scan.Entries{List: []score.CRAPEntry{
		{File: "/project/main.go", Package: "myapp", FuncName: "X", Line: 1, Complexity: 1, Coverage: 0, CRAP: 2},
	}}

	opts := FormatOptions{
		Threshold: 0,
		Writer:    buf,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	var parsed map[string]any
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)

	assert.Equal(t, "https://json.schemastore.org/sarif-2.1.0.json", parsed["$schema"])
}

func TestSARIFFormatter_Format_rule_help_text(t *testing.T) {
	f := &SARIFFormatter{}
	buf := &bytes.Buffer{}

	entries := &scan.Entries{List: []score.CRAPEntry{
		{File: "/project/main.go", Package: "myapp", FuncName: "X", Line: 1, Complexity: 1, Coverage: 0, CRAP: 2},
	}}

	opts := FormatOptions{
		Threshold: 0,
		Writer:    buf,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	got := buf.String()
	assert.True(t, strings.Contains(got, `"CRAP score exceeds threshold"`) ||
		strings.Contains(got, `"CRAP score exceeds threshold`))
}

func TestSARIFFormatter_Format_detailed_mutations(t *testing.T) {
	f := &SARIFFormatter{}
	buf := &bytes.Buffer{}

	entries := &scan.Entries{List: []score.CRAPEntry{
		{
			File:              "/project/main.go",
			Package:           "myapp",
			FuncName:          "BadFunction",
			Line:              10,
			Complexity:        5,
			Coverage:          90.0,
			CRAP:              15.0,
			CoverageUntrusted: true,
			MutationScore:     0.5,
			MutationDetails: []score.MutationDetail{
				{MutantType: "CONDITIONALS_BOUNDARY", Line: 15, Status: "LIVED", OriginalText: "a < b", ReplacementText: "a >= b"},
			},
		},
	}}

	opts := FormatOptions{
		Threshold: 10,
		Writer:    buf,
		Detailed:  true,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	got := buf.String()
	assert.Contains(t, got, "CONDITIONALS_BOUNDARY@L15")
	assert.Contains(t, got, "survived mutations:")
	assert.Contains(t, got, "\\u003c")
	assert.Contains(t, got, "\\u003e")
}

func TestFormatMutantDetails(t *testing.T) {
	details := []score.MutationDetail{
		{MutantType: "CONDITIONALS_BOUNDARY", Line: 15, Status: "LIVED", OriginalText: "a < b", ReplacementText: "a >= b"},
	}
	got := formatMutantDetails(true, details)
	assert.Contains(t, got, "survived mutations:")
	assert.Contains(t, got, "CONDITIONALS_BOUNDARY@L15")
	assert.Contains(t, got, `"a < b" → "a >= b"`)
}

func TestFormatMutantDetails_no_details(t *testing.T) {
	assert.Empty(t, formatMutantDetails(false, nil))
	assert.Empty(t, formatMutantDetails(true, nil))
}

func TestFormatMutantDetails_no_code_strings(t *testing.T) {
	details := []score.MutationDetail{
		{MutantType: "CONDITIONALS_BOUNDARY", Line: 15, Status: "LIVED"},
	}
	got := formatMutantDetails(true, details)
	assert.Contains(t, got, "CONDITIONALS_BOUNDARY@L15")
}

func TestSARIFFormatter_Format_detailed_mutations_no_details(t *testing.T) {
	f := &SARIFFormatter{}
	buf := &bytes.Buffer{}

	entries := &scan.Entries{List: []score.CRAPEntry{
		{
			File:              "/project/main.go",
			Package:           "myapp",
			FuncName:          "BadFunction",
			Line:              10,
			Complexity:        5,
			Coverage:          90.0,
			CRAP:              15.0,
			CoverageUntrusted: true,
			MutationScore:     0.5,
			MutationDetails:   nil,
		},
	}}

	opts := FormatOptions{
		Threshold: 10,
		Writer:    buf,
		Detailed:  true,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	got := buf.String()
	assert.NotContains(t, got, "survived mutations:")
	assert.NotContains(t, got, "CONDITIONALS_BOUNDARY")
}

func TestSARIFFormatter_Format_detailed_disabled(t *testing.T) {
	f := &SARIFFormatter{}
	buf := &bytes.Buffer{}

	entries := &scan.Entries{List: []score.CRAPEntry{
		{
			File:              "/project/main.go",
			Package:           "myapp",
			FuncName:          "BadFunction",
			Line:              10,
			Complexity:        5,
			Coverage:          90.0,
			CRAP:              15.0,
			CoverageUntrusted: true,
			MutationScore:     0.5,
			MutationDetails: []score.MutationDetail{
				{MutantType: "CONDITIONALS_BOUNDARY", Line: 15, Status: "LIVED", OriginalText: "a < b", ReplacementText: "a >= b"},
			},
		},
	}}

	opts := FormatOptions{
		Threshold: 10,
		Writer:    buf,
		Detailed:  false,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	got := buf.String()
	assert.NotContains(t, got, "CONDITIONALS_BOUNDARY")
	assert.NotContains(t, got, "survived mutations:")
	assert.NotContains(t, got, "\u003c")
}

func TestSARIFFormatter_Format_exact_threshold_no_high_score_result(t *testing.T) {
	// COND_BOUND :23 — effectiveCRAP > opts.Threshold changed to >= would
	// include entries at exact threshold. Must produce empty results.
	f := &SARIFFormatter{}
	buf := &bytes.Buffer{}
	entries := &scan.Entries{List: []score.CRAPEntry{
		{File: "/project/main.go", Package: "myapp", FuncName: "Exact",
			Line: 1, Complexity: 5, Coverage: 100, CRAP: 30},
	}}
	opts := FormatOptions{Threshold: 30, Writer: buf}
	err := f.Format(entries, opts)
	require.NoError(t, err)
	var parsed sarifLog
	err = json.Unmarshal(buf.Bytes(), &parsed)
	require.NoError(t, err)
	assert.Empty(t, parsed.Runs[0].Results)
}

func TestSARIFFormatter_Format_untrusted_exact_mutation_score(t *testing.T) {
	// ARITH :46 — e.MutationScore*100 changed to /100 or +100 would
	// produce wrong percentage in untrusted warning message.
	f := &SARIFFormatter{}
	buf := &bytes.Buffer{}
	entries := &scan.Entries{List: []score.CRAPEntry{
		{
			File: "/project/main.go", Package: "myapp",
			FuncName:          "UnreliableFunc",
			Line:              10,
			Complexity:        2,
			Coverage:          90.0,
			CRAP:              5.0,
			CoverageUntrusted: true,
			MutationScore:     0.65,
		},
	}}
	opts := FormatOptions{Threshold: 30, Writer: buf}
	err := f.Format(entries, opts)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "mutation score: 65.0%")
}

func TestSARIFFormatter_nil_entries(t *testing.T) {
	formatter := &SARIFFormatter{}
	var buf strings.Builder
	err := formatter.Format(nil, FormatOptions{Writer: &buf})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestSARIFFormatter_untrusted_only(t *testing.T) {
	formatter := &SARIFFormatter{}
	var buf strings.Builder
	entries := &scan.Entries{List: []score.CRAPEntry{
		{CRAP: 20.0, Coverage: 50.0, CoverageUntrusted: true, FuncName: "untrusted",
			MutationScore: 0.5, File: "file.go", Line: 10},
	}}
	err := formatter.Format(entries, FormatOptions{Writer: &buf, Threshold: 30.0})
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "coverage-untrusted")
	assert.Contains(t, output, "Coverage not reliable")
}

func TestSARIFFormatter_high_score_only(t *testing.T) {
	formatter := &SARIFFormatter{}
	var buf strings.Builder
	entries := &scan.Entries{List: []score.CRAPEntry{
		{CRAP: 50.0, Coverage: 50.0, CoverageUntrusted: false, FuncName: "highScore", File: "file.go", Line: 10},
	}}
	err := formatter.Format(entries, FormatOptions{Writer: &buf, Threshold: 30.0})
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "high-score")
	assert.NotContains(t, output, "coverage-untrusted")
}

func TestSARIFFormatter_high_score_and_untrusted(t *testing.T) {
	formatter := &SARIFFormatter{}
	var buf strings.Builder
	entries := &scan.Entries{List: []score.CRAPEntry{
		{CRAP: 50.0, Coverage: 50.0, CoverageUntrusted: true, FuncName: "both",
			MutationScore: 0.5, File: "file.go", Line: 10},
	}}
	err := formatter.Format(entries, FormatOptions{Writer: &buf, Threshold: 30.0})
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "high-score")
	assert.Contains(t, output, "coverage-untrusted")
}

func Test_relativizePath_backslash_conversion(t *testing.T) {
	result := relativizePath("C:\\Users\\file.go", "")
	assert.Contains(t, result, "/")
	assert.NotContains(t, result, "\\")
}

func TestFormatMutantDetails_detailed_with_code(t *testing.T) {
	details := []score.MutationDetail{
		{MutantType: "CONDITIONALS_BOUNDARY", Line: 10, OriginalText: "a > b", ReplacementText: "a >= b"},
	}
	result := formatMutantDetails(true, details)
	assert.Contains(t, result, "CONDITIONALS_BOUNDARY@L10")
	assert.Contains(t, result, `"a > b"`)
	assert.Contains(t, result, `"a >= b"`)
}

func Test_formatMessage_untrusted_with_details(t *testing.T) {
	e := score.CRAPEntry{
		FuncName: "testFunc", CoverageUntrusted: true, MutationScore: 0.5,
		MutationDetails: []score.MutationDetail{
			{MutantType: "CONDITIONALS_NEGATION", Line: 5},
		},
	}
	msg := formatMessage(e, 30.0, true)
	assert.Contains(t, msg, "mutation score: 50.0%")
	assert.Contains(t, msg, "CONDITIONALS_NEGATION@L5")
}
