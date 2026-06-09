package report

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/padiazg/go-crap/internal/score"
)

type checkPRCommentFormatterOutputFn func(*testing.T, string)

var checkPRCommentFormatterOutput = func(fns ...checkPRCommentFormatterOutputFn) []checkPRCommentFormatterOutputFn {
	return fns
}

func checkPRCommentOutputContains(want string) checkPRCommentFormatterOutputFn {
	return func(t *testing.T, got string) {
		t.Helper()
		assert.Containsf(t, got, want, "output should contain %q", want)
	}
}

func checkPRCommentOutputNotContains(want string) checkPRCommentFormatterOutputFn {
	return func(t *testing.T, got string) {
		t.Helper()
		assert.NotContainsf(t, got, want, "output should not contain %q", want)
	}
}

func TestPRCommentFormatter_Format(t *testing.T) {
	tests := []struct {
		name    string
		entries *score.EntryList
		opts    FormatOptions
		checks  []checkPRCommentFormatterOutputFn
	}{
		{
			name:    "success_empty_entries",
			entries: &score.EntryList{List: []score.CRAPEntry{}},
			checks: checkPRCommentFormatterOutput(
				checkPRCommentOutputContains("<!-- go-crap-report -->"),
				checkPRCommentOutputContains("## No crappy functions"),
				checkPRCommentOutputContains("0 function(s) analyzed"),
			),
		},
		{
			name: "success_no_crappy_functions",
			entries: &score.EntryList{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Good", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1},
				{File: "/project/main.go", Package: "myapp", FuncName: "OK", Line: 10, Complexity: 2, Coverage: 80, CRAP: 5},
			}},
			opts: FormatOptions{Threshold: 200},
			checks: checkPRCommentFormatterOutput(
				checkPRCommentOutputContains("## No crappy functions"),
				checkPRCommentOutputContains("2 function(s) analyzed"),
				checkPRCommentOutputNotContains("```"),
			),
		},
		{
			name: "success_with_crappy_functions",
			entries: &score.EntryList{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Bad", Line: 42, Complexity: 10, Coverage: 0, CRAP: 110},
				{File: "/project/main.go", Package: "myapp", FuncName: "Good", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1},
			}},
			opts: FormatOptions{Threshold: 30},
			checks: checkPRCommentFormatterOutput(
				checkPRCommentOutputContains("## 1 crappy function(s)"),
				checkPRCommentOutputContains("2 function(s) analyzed"),
				checkPRCommentOutputContains("| ✗ |"),
				checkPRCommentOutputContains("| 110.00 |"),
				checkPRCommentOutputContains("| `Bad` |"),
			),
		},
		{
			name: "success_sorted_by_crap_desc",
			entries: &score.EntryList{List: []score.CRAPEntry{
				{File: "/project/a.go", Package: "myapp", FuncName: "Low", Line: 1, Complexity: 3, Coverage: 0, CRAP: 12},
				{File: "/project/b.go", Package: "myapp", FuncName: "High", Line: 2, Complexity: 10, Coverage: 0, CRAP: 110},
				{File: "/project/c.go", Package: "myapp", FuncName: "Mid", Line: 3, Complexity: 5, Coverage: 10, CRAP: 45},
			}},
			opts: FormatOptions{Threshold: 10},
			checks: checkPRCommentFormatterOutput(
				checkPRCommentOutputContains("High"),
				checkPRCommentOutputContains("Mid"),
				checkPRCommentOutputContains("Low"),
			),
		},
		{
			name: "success_threshold_boundary",
			entries: &score.EntryList{List: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Exact", Line: 1, Complexity: 10, Coverage: 100, CRAP: 10},
				{File: "/project/main.go", Package: "myapp", FuncName: "Over", Line: 2, Complexity: 11, Coverage: 100, CRAP: 11},
			}},
			opts: FormatOptions{Threshold: 10},
			checks: checkPRCommentFormatterOutput(
				checkPRCommentOutputContains("## 1 crappy function(s)"),
				checkPRCommentOutputContains("Over"),
				checkPRCommentOutputNotContains("Exact"),
			),
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f := &PRCommentFormatter{}
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

func TestPRCommentFormatter_Format_truncation_at_25(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &score.EntryList{List: make([]score.CRAPEntry, 30)}
	for i := range 30 {
		entries.List[i] = score.CRAPEntry{
			File:       "/project/main.go",
			Package:    "myapp",
			FuncName:   func(i int) string {
				return "Func" + string(rune('A'+i))
			}(i),
			Line:       i + 1,
			Complexity: 10,
			Coverage:   0,
			CRAP:       float64(100 - i),
		}
	}

	opts := FormatOptions{
		Threshold: 0,
		Writer:    buf,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "…and 5 more")
}

func TestPRCommentFormatter_Format_html_marker_present(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &score.EntryList{List: []score.CRAPEntry{
		{File: "/project/main.go", Package: "myapp", FuncName: "Bad", Line: 1, Complexity: 10, Coverage: 0, CRAP: 100},
	}}

	opts := FormatOptions{
		Threshold: 30,
		Writer:    buf,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	lines := strings.Split(buf.String(), "\n")
	assert.Equal(t, "<!-- go-crap-report -->", lines[0])
}

func TestPRCommentFormatter_Format_summary_line(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &score.EntryList{List: []score.CRAPEntry{
		{File: "/project/main.go", Package: "myapp", FuncName: "A", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1},
		{File: "/project/main.go", Package: "myapp", FuncName: "B", Line: 2, Complexity: 2, Coverage: 80, CRAP: 6.4},
		{File: "/project/main.go", Package: "myapp", FuncName: "C", Line: 3, Complexity: 3, Coverage: 60, CRAP: 14.4},
	}}

	opts := FormatOptions{
		Threshold: 30,
		Writer:    buf,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "3 function(s) analyzed")
	assert.Contains(t, output, "threshold 30")
}

func TestPRCommentFormatter_Format_status_icons(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &score.EntryList{List: []score.CRAPEntry{
		{File: "/project/main.go", Package: "myapp", FuncName: "Bad", Line: 10, Complexity: 10, Coverage: 0, CRAP: 110},
	}}

	opts := FormatOptions{
		Threshold: 30,
		Writer:    buf,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "✗")
}

func TestPRCommentFormatter_Format_base_dir_relativize(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &score.EntryList{List: []score.CRAPEntry{
		{File: "/tmp/project/main.go", Package: "myapp", FuncName: "Process", Line: 5, Complexity: 1, Coverage: 0, CRAP: 1},
	}}

	opts := FormatOptions{
		Threshold: 0,
		BaseDir:   "/tmp/project",
		Writer:    buf,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "`main.go:5`")
	assert.NotContains(t, output, "`/tmp/project/main.go:5`")
}

func TestPRCommentFormatter_Format_nil_entries(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	err := f.Format(nil, FormatOptions{Writer: buf})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "nil")
}

func TestPRCommentFormatter_Format_no_table_when_no_crappy(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &score.EntryList{List: []score.CRAPEntry{
		{File: "/project/main.go", Package: "myapp", FuncName: "Good", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1},
	}}

	opts := FormatOptions{
		Threshold: 200,
		Writer:    buf,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "## No crappy functions")
	assert.NotContains(t, output, "|")
}

func TestPRCommentFormatter_Format_unreliable_coverage_without_threshold_violation(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &score.EntryList{List: []score.CRAPEntry{
		{
			File:              "/project/main.go",
			Package:           "myapp",
			FuncName:          "GoodFunction",
			Line:              10,
			Complexity:        2,
			Coverage:          95.0,
			CRAP:              5.0,
			CoverageUntrusted: true,
			MutationScore:     0.6,
		},
		{
			File:              "/project/main.go",
			Package:           "myapp",
			FuncName:          "AlsoGood",
			Line:              20,
			Complexity:        1,
			Coverage:          100.0,
			CRAP:              1.0,
			CoverageUntrusted: false,
		},
	}}

	opts := FormatOptions{
		Threshold: 30,
		Writer:    buf,
		Detailed:  true,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "## No crappy functions")
	assert.Contains(t, output, "## \u26a0\ufe0f Unreliable Coverage")
	assert.Contains(t, output, "GoodFunction")
	assert.Contains(t, output, "60.0%")
}

func TestPRCommentFormatter_Format_unreliable_with_crappy_functions(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &score.EntryList{List: []score.CRAPEntry{
		{
			File:              "/project/main.go",
			Package:           "myapp",
			FuncName:          "BadFunction",
			Line:              10,
			Complexity:        10,
			Coverage:          0,
			CRAP:              110.0,
			CoverageUntrusted: false,
		},
		{
			File:              "/project/main.go",
			Package:           "myapp",
			FuncName:          "UnreliableGood",
			Line:              20,
			Complexity:        3,
			Coverage:          90.0,
			CRAP:              10.0,
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

	output := buf.String()
	assert.Contains(t, output, "## 1 crappy function(s)")
	assert.Contains(t, output, "BadFunction")
	assert.Contains(t, output, "## \u26a0\ufe0f Unreliable Coverage")
	assert.Contains(t, output, "UnreliableGood")
	assert.Contains(t, output, "| ✗ |")
	assert.Contains(t, output, "Mutation Score")
}

func TestPRCommentFormatter_Format_detailed_mutations(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &score.EntryList{List: []score.CRAPEntry{
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
				{MutantType: "ARITHMETIC", Line: 18, Status: "LIVED", OriginalText: "a + b", ReplacementText: "a - b"},
			},
		},
		{
			File:              "/project/main.go",
			Package:           "myapp",
			FuncName:          "AlsoBad",
			Line:              20,
			Complexity:        3,
			Coverage:          80.0,
			CRAP:              15.0,
			CoverageUntrusted: true,
			MutationScore:     0.75,
			MutationDetails: []score.MutationDetail{
				{MutantType: "CONTROL_FLOW", Line: 22, Status: "LIVED"},
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

	output := buf.String()
	assert.Contains(t, output, "Survived Mutants")
	assert.Contains(t, output, "`CONDITIONALS_BOUNDARY`@L15")
	assert.Contains(t, output, "`a < b` → `a >= b`")
	assert.Contains(t, output, "`ARITHMETIC`@L18")
	assert.Contains(t, output, "`a + b` → `a - b`")
	assert.Contains(t, output, "`CONTROL_FLOW`@L22")
	assert.Contains(t, output, "| Function | CRAP | Effective CRAP | Mutation Score | Survived Mutants |")
}

func TestPRCommentFormatter_Format_no_detailed_by_default(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &score.EntryList{List: []score.CRAPEntry{
		{
			File:              "/project/main.go",
			Package:           "myapp",
			FuncName:          "Bad",
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

	output := buf.String()
	assert.NotContains(t, output, "Survived Mutants")
	assert.NotContains(t, output, "CONDITIONALS_BOUNDARY")
	assert.NotContains(t, output, "a < b")
	assert.Contains(t, output, "| Function | CRAP | Effective CRAP | Mutation Score |")
}
