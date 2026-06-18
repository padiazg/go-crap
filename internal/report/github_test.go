package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/padiazg/go-crap/internal/scan"
	"github.com/padiazg/go-crap/internal/score"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type checkGithubFormatterOutputFn func(*testing.T, string)

var checkGithubFormatterOutput = func(fns ...checkGithubFormatterOutputFn) []checkGithubFormatterOutputFn {
	return fns
}

func checkGithubOutputContains(want string) checkGithubFormatterOutputFn {
	return func(t *testing.T, got string) {
		t.Helper()
		assert.Containsf(t, got, want, "output should contain %q", want)
	}
}

func checkGithubOutputNotContains(want string) checkGithubFormatterOutputFn {
	return func(t *testing.T, got string) {
		t.Helper()
		assert.NotContainsf(t, got, want, "output should not contain %q", want)
	}
}

func checkOutputLineCount(want int) checkGithubFormatterOutputFn {
	return func(t *testing.T, got string) {
		t.Helper()
		count := 0
		for _, ch := range got {
			if ch == '\n' {
				count++
			}
		}
		assert.Equalf(t, want, count, "output should have %d newlines, got %d", want, count)
	}
}

func checkOutputEmpty() checkGithubFormatterOutputFn {
	return func(t *testing.T, got string) {
		t.Helper()
		assert.Empty(t, got, "output should be empty")
	}
}

func TestGithubFormatter_Format(t *testing.T) {
	tests := []struct {
		name    string
		entries scan.Entries
		opts    FormatOptions
		checks  []checkGithubFormatterOutputFn
	}{
		{
			name:    "success_empty_entries",
			entries: scan.Entries{List: []score.CRAPEntry{}},
			opts:    FormatOptions{Threshold: 200},
			checks: checkGithubFormatterOutput(
				checkOutputEmpty(),
			),
		},
		{
			name: "success_exceeds_threshold",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "BadFunc", Line: 10, Complexity: 10, Coverage: 0, CRAP: 100},
			}},
			opts: FormatOptions{Threshold: 50},
			checks: checkGithubFormatterOutput(
				checkGithubOutputContains("::warning"),
				checkGithubOutputContains("file=/project/main.go"),
				checkGithubOutputContains("line=10"),
				checkGithubOutputContains("BadFunc"),
				checkGithubOutputContains("CRAP score 100.0"),
				checkGithubOutputContains("CC=10"),
				checkGithubOutputContains("cov=0.0%"),
				checkGithubOutputContains("exceeds threshold 50"),
			),
		},
		{
			name: "success_below_threshold",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "GoodFunc", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1},
			}},
			opts: FormatOptions{Threshold: 50},
			checks: checkGithubFormatterOutput(
				checkGithubOutputNotContains("::warning"),
				checkOutputEmpty(),
			),
		},
		{
			name: "success_method_with_receiver",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "*MyType.Process", Line: 20, Complexity: 8, Coverage: 0, CRAP: 72},
			}},
			opts: FormatOptions{Threshold: 50},
			checks: checkGithubFormatterOutput(
				checkGithubOutputContains("::warning"),
				checkGithubOutputContains("*MyType.Process"),
				checkGithubOutputContains("line=20"),
			),
		},
		{
			name: "success_mixed_entries",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "GoodFunc", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1},
				{File: "/project/main.go", Package: "myapp", FuncName: "BadFunc", Line: 10, Complexity: 10, Coverage: 0, CRAP: 100},
				{File: "/project/utils.go", Package: "myapp", FuncName: "UglyFunc", Line: 5, Complexity: 8, Coverage: 10, CRAP: 57.76},
			}},
			opts: FormatOptions{Threshold: 50},
			checks: checkGithubFormatterOutput(
				checkGithubOutputContains("file=/project/main.go"),
				checkGithubOutputContains("line=10"),
				checkGithubOutputContains("BadFunc"),
				checkGithubOutputContains("file=/project/utils.go"),
				checkGithubOutputContains("line=5"),
				checkGithubOutputContains("UglyFunc"),
				checkOutputLineCount(2),
			),
		},
		{
			name: "success_base_dir_rewrites_path",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/tmp/project/main.go", Package: "myapp", FuncName: "Process", Line: 5, Complexity: 10, Coverage: 0, CRAP: 100},
			}},
			opts: FormatOptions{Threshold: 50, BaseDir: "/tmp/project"},
			checks: checkGithubFormatterOutput(
				checkGithubOutputContains("file=main.go"),
				checkGithubOutputNotContains("file=/tmp/project/main.go"),
				checkGithubOutputContains("line=5"),
				checkGithubOutputContains("Process"),
			),
		},
		{
			name: "success_base_dir_no_rewrite_when_no_match",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/other/project/main.go", Package: "myapp", FuncName: "Process", Line: 5, Complexity: 10, Coverage: 0, CRAP: 100},
			}},
			opts: FormatOptions{Threshold: 50, BaseDir: "/tmp/project"},
			checks: checkGithubFormatterOutput(
				checkGithubOutputContains("file=../../other/project/main.go"),
				checkGithubOutputContains("line=5"),
				checkGithubOutputContains("Process"),
			),
		},
		{
			name: "success_threshold_boundary_exactly_at_threshold",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Exact", Line: 1, Complexity: 10, Coverage: 100, CRAP: 10},
			}},
			opts: FormatOptions{Threshold: 10},
			checks: checkGithubFormatterOutput(
				checkGithubOutputNotContains("::warning"),
				checkOutputEmpty(),
			),
		},
		{
			name: "success_threshold_boundary_one_over",
			entries: scan.Entries{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Over", Line: 1, Complexity: 2, Coverage: 100, CRAP: 11},
			}},
			opts: FormatOptions{Threshold: 10},
			checks: checkGithubFormatterOutput(
				checkGithubOutputContains("::warning"),
				checkGithubOutputContains("Over"),
				checkGithubOutputContains("CRAP score 11.0"),
				checkGithubOutputContains("exceeds threshold 10"),
			),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &GithubFormatter{}
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

func TestGithubFormatter_Format_relative_path_conversion(t *testing.T) {
	tests := []struct {
		name      string
		entryFile string
		baseDir   string
		wantInOut string
		notInOut  string
	}{
		{
			name:      "relative_path_not_changed_when_no_base_dir",
			entryFile: "./main.go",
			baseDir:   "",
			wantInOut: "file=./main.go",
			notInOut:  "",
		},
		{
			name:      "base_dir_matches_absolute_path",
			entryFile: "/home/user/project/src/main.go",
			baseDir:   "/home/user/project",
			wantInOut: "file=src/main.go",
			notInOut:  "file=/home/user/project/src/main.go",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := &GithubFormatter{}
			buf := &bytes.Buffer{}
			entries := &scan.Entries{List: []score.CRAPEntry{
				{File: tt.entryFile, Package: "myapp", FuncName: "Foo", Line: 1, Complexity: 10, Coverage: 0, CRAP: 100},
			}}
			opts := FormatOptions{
				Threshold: 50,
				BaseDir:   tt.baseDir,
				Writer:    buf,
			}
			err := f.Format(entries, opts)
			require.NoError(t, err)
			got := buf.String()
			assert.Contains(t, got, tt.wantInOut)
			if tt.notInOut != "" {
				assert.NotContains(t, got, tt.notInOut)
			}
		})
	}
}

func TestGithubFormatter_Format_output_format_details(t *testing.T) {
	f := &GithubFormatter{}
	buf := &bytes.Buffer{}
	entries := &scan.Entries{List: []score.CRAPEntry{
		{File: "/project/main.go", Package: "myapp", FuncName: "Complex", Line: 42, Complexity: 7, Coverage: 30, CRAP: 123.46},
	}}
	opts := FormatOptions{
		Threshold: 50,
		Writer:    buf,
	}
	err := f.Format(entries, opts)
	require.NoError(t, err)

	got := buf.String()
	assert.Contains(t, got, "::warning")
	assert.Contains(t, got, "file=/project/main.go")
	assert.Contains(t, got, "line=42")
	assert.Contains(t, got, "Complex")
	assert.Contains(t, got, "CRAP score 123.5")
	assert.Contains(t, got, "CC=7")
	assert.Contains(t, got, "cov=30.0%")
	assert.Contains(t, got, "exceeds threshold 50")

	lines := strings.Split(strings.TrimSpace(got), "\n")
	assert.Len(t, lines, 1)
}

func TestGithubFormatter_Format_returns_nil_error(t *testing.T) {
	f := &GithubFormatter{}
	buf := &bytes.Buffer{}
	entries := &scan.Entries{List: []score.CRAPEntry{
		{File: "/project/main.go", Package: "myapp", FuncName: "Foo", Line: 1, Complexity: 10, Coverage: 0, CRAP: 100},
	}}
	opts := FormatOptions{
		Threshold: 50,
		Writer:    buf,
	}
	err := f.Format(entries, opts)
	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Foo")

	err = f.Format(nil, opts)
	assert.Error(t, err)

	err = f.Format(&scan.Entries{List: []score.CRAPEntry{}}, opts)
	assert.NoError(t, err)
}

func TestGithubFormatter_Format_coverage_untrusted_below_threshold(t *testing.T) {
	f := &GithubFormatter{}
	buf := &bytes.Buffer{}
	entries := &scan.Entries{List: []score.CRAPEntry{
		{
			File:              "/project/main.go",
			Package:           "myapp",
			FuncName:          "GoodFunction",
			Line:              42,
			Complexity:        2,
			Coverage:          95.0,
			CRAP:              3.0,
			CoverageUntrusted: true,
			MutationScore:     0.6,
		},
	}}
	opts := FormatOptions{
		Threshold: 30,
		Writer:    buf,
	}
	err := f.Format(entries, opts)
	require.NoError(t, err)

	got := buf.String()
	assert.Contains(t, got, "::warning")
	assert.Contains(t, got, "coverage not reliable")
	assert.Contains(t, got, "mutation score: 60.0%")
	assert.Contains(t, got, "GoodFunction")
}

func TestGithubFormatter_Format_coverage_untrusted_above_threshold(t *testing.T) {
	f := &GithubFormatter{}
	buf := &bytes.Buffer{}
	entries := &scan.Entries{List: []score.CRAPEntry{
		{
			File:              "/project/main.go",
			Package:           "myapp",
			FuncName:          "BadFunction",
			Line:              10,
			Complexity:        10,
			Coverage:          0,
			CRAP:              110.0,
			CoverageUntrusted: true,
			MutationScore:     0.3,
		},
	}}
	opts := FormatOptions{
		Threshold: 30,
		Writer:    buf,
	}
	err := f.Format(entries, opts)
	require.NoError(t, err)

	got := buf.String()
	lines := strings.Split(strings.TrimSpace(got), "\n")
	assert.Len(t, lines, 2)
	assert.Contains(t, got, "CRAP score 110.0")
	assert.Contains(t, got, "coverage not reliable")
}

func TestGithubFormatter_Format_multiple_unreliable_below_threshold(t *testing.T) {
	f := &GithubFormatter{}
	buf := &bytes.Buffer{}
	entries := &scan.Entries{List: []score.CRAPEntry{
		{
			File:              "/project/a.go",
			Package:           "myapp",
			FuncName:          "FuncA",
			Line:              5,
			Complexity:        2,
			Coverage:          90.0,
			CRAP:              5.0,
			CoverageUntrusted: true,
			MutationScore:     0.5,
		},
		{
			File:              "/project/b.go",
			Package:           "myapp",
			FuncName:          "FuncB",
			Line:              15,
			Complexity:        3,
			Coverage:          85.0,
			CRAP:              8.0,
			CoverageUntrusted: true,
			MutationScore:     0.7,
		},
	}}
	opts := FormatOptions{
		Threshold: 30,
		Writer:    buf,
	}
	err := f.Format(entries, opts)
	require.NoError(t, err)

	got := buf.String()
	assert.Contains(t, got, "FuncA")
	assert.Contains(t, got, "FuncB")
	assert.Contains(t, got, "coverage not reliable")
}

func TestGithubFormatter_Format_crap_warning_exact_mutation_score(t *testing.T) {
	// ARITH :53 — e.MutationScore*100 in formatGithubCRAPWarning appended text.
	// Mutant changes *100, producing wrong percentage on the CRAP warning line.
	f := &GithubFormatter{}
	buf := &bytes.Buffer{}
	entries := &scan.Entries{List: []score.CRAPEntry{
		{
			File: "/project/main.go", Package: "myapp",
			FuncName:          "BadFunc",
			Line:              10,
			Complexity:        10,
			Coverage:          0,
			CRAP:              100,
			CoverageUntrusted: true,
			MutationScore:     0.75,
		},
	}}
	opts := FormatOptions{Threshold: 30, Writer: buf}
	err := f.Format(entries, opts)
	require.NoError(t, err)
	got := buf.String()
	lines := strings.Split(strings.TrimSpace(got), "\n")
	require.Len(t, lines, 2)
	// Second line is the CRAP warning which appends mutation score
	assert.Contains(t, lines[1], "CRAP score")
	assert.Contains(t, lines[1], "mutation score: 75.0%")
}

func TestGithubFormatter_nil_entries(t *testing.T) {
	formatter := &GithubFormatter{}
	var buf strings.Builder
	err := formatter.Format(nil, FormatOptions{Writer: &buf})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestGithubFormatter_untrusted_below_threshold(t *testing.T) {
	formatter := &GithubFormatter{}
	var buf strings.Builder
	entries := &scan.Entries{List: []score.CRAPEntry{
		{CRAP: 20.0, Coverage: 50.0, CoverageUntrusted: true, FuncName: "untrusted",
			MutationScore: 0.5, File: "file.go", Line: 10},
	}}
	err := formatter.Format(entries, FormatOptions{Writer: &buf, Threshold: 30.0})
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "coverage not reliable")
	assert.Contains(t, output, "50.0%")
}

func TestGithubFormatter_untrusted_above_threshold(t *testing.T) {
	formatter := &GithubFormatter{}
	var buf strings.Builder
	entries := &scan.Entries{List: []score.CRAPEntry{
		{CRAP: 50.0, Coverage: 50.0, CoverageUntrusted: true, FuncName: "untrusted",
			MutationScore: 0.5, File: "file.go", Line: 10},
	}}
	err := formatter.Format(entries, FormatOptions{Writer: &buf, Threshold: 30.0})
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "CRAP score")
	assert.Contains(t, output, "coverage not reliable")
}

func TestGithubFormatter_threshold_exactly_met(t *testing.T) {
	formatter := &GithubFormatter{}
	var buf strings.Builder
	entries := &scan.Entries{List: []score.CRAPEntry{
		{CRAP: 30.0, Coverage: 50.0, CoverageUntrusted: false, FuncName: "exact", File: "f.go", Line: 1},
	}}
	err := formatter.Format(entries, FormatOptions{Writer: &buf, Threshold: 30.0})
	assert.NoError(t, err)
	assert.Empty(t, strings.TrimSpace(buf.String()))
}

func TestGithubFormatter_one_above_threshold(t *testing.T) {
	formatter := &GithubFormatter{}
	var buf strings.Builder
	entries := &scan.Entries{List: []score.CRAPEntry{
		{CRAP: 30.01, Coverage: 50.0, CoverageUntrusted: false, FuncName: "above", File: "f.go", Line: 1},
	}}
	err := formatter.Format(entries, FormatOptions{Writer: &buf, Threshold: 30.0})
	assert.NoError(t, err)
	assert.NotEmpty(t, strings.TrimSpace(buf.String()))
}
