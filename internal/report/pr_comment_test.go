package report

import (
	"bytes"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/padiazg/go-crap/internal/scan"
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
		entries *scan.Entries
		opts    FormatOptions
		checks  []checkPRCommentFormatterOutputFn
	}{
		{
			name:    "success_empty_entries",
			entries: &scan.Entries{List: []score.CRAPEntry{}},
			checks: checkPRCommentFormatterOutput(
				checkPRCommentOutputContains("<!-- go-crap-report -->"),
				checkPRCommentOutputContains("## No crappy functions"),
				checkPRCommentOutputContains("0 function(s) analyzed"),
			),
		},
		{
			name: "success_no_crappy_functions",
			entries: &scan.Entries{List: []score.CRAPEntry{
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
			entries: &scan.Entries{List: []score.CRAPEntry{
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
			entries: &scan.Entries{List: []score.CRAPEntry{
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
			entries: &scan.Entries{List: []score.CRAPEntry{
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

	entries := &scan.Entries{List: make([]score.CRAPEntry, 30)}
	for i := range 30 {
		entries.List[i] = score.CRAPEntry{
			File:    "/project/main.go",
			Package: "myapp",
			FuncName: func(i int) string {
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

	entries := &scan.Entries{List: []score.CRAPEntry{
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

	entries := &scan.Entries{List: []score.CRAPEntry{
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

	entries := &scan.Entries{List: []score.CRAPEntry{
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

	entries := &scan.Entries{List: []score.CRAPEntry{
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

	entries := &scan.Entries{List: []score.CRAPEntry{
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

	entries := &scan.Entries{List: []score.CRAPEntry{
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

	entries := &scan.Entries{List: []score.CRAPEntry{
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

	entries := &scan.Entries{List: []score.CRAPEntry{
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

func TestPRCommentFormatter_Format_sort_stability_with_equal_effective_crap(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &scan.Entries{List: []score.CRAPEntry{
		{File: "/project/a.go", Package: "myapp", FuncName: "EqualA", Line: 1, Complexity: 5, Coverage: 0, CRAP: 40, EffectiveCRAP: 40},
		{File: "/project/b.go", Package: "myapp", FuncName: "EqualB", Line: 2, Complexity: 5, Coverage: 0, CRAP: 30, EffectiveCRAP: 40},
		{File: "/project/c.go", Package: "myapp", FuncName: "JustBelow", Line: 3, Complexity: 3, Coverage: 0, CRAP: 20, EffectiveCRAP: 20},
	}}

	opts := FormatOptions{
		Threshold: 10,
		Writer:    buf,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	output := buf.String()
	posA := strings.Index(output, "EqualA")
	posB := strings.Index(output, "EqualB")
	posC := strings.Index(output, "JustBelow")
	require.GreaterOrEqual(t, posA, 0, "EqualA should appear in output")
	require.GreaterOrEqual(t, posB, 0, "EqualB should appear in output")
	require.GreaterOrEqual(t, posC, 0, "JustBelow should appear in output")
	require.Greater(t, posC, posA, "JustBelow (EffectiveCRAP=20) should appear after EqualA and EqualB (EffectiveCRAP=40)")
}

func TestPRCommentFormatter_Format_boundary_25_entries_exactly(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &scan.Entries{List: make([]score.CRAPEntry, 25)}
	for i := range 25 {
		entries.List[i] = score.CRAPEntry{
			File:       "/project/main.go",
			Package:    "myapp",
			FuncName:   fmt.Sprintf("Func%d", i),
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
	assert.NotContains(t, output, "and .* more", "exactly 25 entries should not show truncation message")
	assert.Contains(t, output, "Func0")
	assert.Contains(t, output, "Func24")
}

func TestPRCommentFormatter_Format_boundary_26_entries_truncated(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &scan.Entries{List: make([]score.CRAPEntry, 26)}
	for i := range 26 {
		entries.List[i] = score.CRAPEntry{
			File:       "/project/main.go",
			Package:    "myapp",
			FuncName:   fmt.Sprintf("Func%d", i),
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
	assert.Contains(t, output, "and 1 more", "26 entries should show truncation with exactly 1 more")
	assert.NotContains(t, output, "Func25")
	assert.Contains(t, output, "Func0")
}

func TestPRCommentFormatter_Format_status_symbol_all_above_threshold(t *testing.T) {
	// All entries above threshold should show "✗" regardless of halfThreshold
	// This verifies the ARITHMETIC_BASE mutant at halfThreshold calc doesn't
	// affect output (all entries in crappy table are above threshold, so
	// StatusSymbol always returns "✗" for them)
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &scan.Entries{List: []score.CRAPEntry{
		{File: "/project/main.go", Package: "myapp", FuncName: "JustOver", Line: 1, Complexity: 6, Coverage: 0, CRAP: 21},
		{File: "/project/main.go", Package: "myapp", FuncName: "FarOver", Line: 2, Complexity: 15, Coverage: 0, CRAP: 225},
	}}

	opts := FormatOptions{
		Threshold: 20,
		Writer:    buf,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	output := buf.String()
	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if strings.Contains(line, "JustOver") || strings.Contains(line, "FarOver") {
			assert.Contains(t, line, "✗", "Entries above threshold should show ✗")
			assert.NotContains(t, line, "▲", "Entries above threshold should NOT show ▲")
			assert.NotContains(t, line, "✓", "Entries above threshold should NOT show ✓")
		}
	}
}

func Test_formatMutantsStr_empty_details(t *testing.T) {
	got := formatMutantsStr(nil)
	assert.Empty(t, got)

	got = formatMutantsStr([]score.MutationDetail{})
	assert.Empty(t, got)
}

func Test_formatMutantsStr_single_detail_no_text(t *testing.T) {
	details := []score.MutationDetail{
		{MutantType: "CONDITIONALS_BOUNDARY", Line: 10, Status: "LIVED"},
	}
	got := formatMutantsStr(details)
	assert.Contains(t, got, "`CONDITIONALS_BOUNDARY`@L10")
	assert.NotContains(t, got, "\n    `")
}

func Test_formatMutantsStr_multiple_details_with_text(t *testing.T) {
	details := []score.MutationDetail{
		{MutantType: "ARITHMETIC", Line: 5, Status: "LIVED", OriginalText: "a + b", ReplacementText: "a - b"},
		{MutantType: "CONDITIONALS_NEGATION", Line: 8, Status: "LIVED", OriginalText: "x == y", ReplacementText: "x != y"},
		{MutantType: "INVERT_NEGATIVES", Line: 12, Status: "LIVED"},
	}
	got := formatMutantsStr(details)
	assert.Contains(t, got, "`ARITHMETIC`@L5")
	assert.Contains(t, got, "`a + b` → `a - b`")
	assert.Contains(t, got, ", ")
	assert.Contains(t, got, "`CONDITIONALS_NEGATION`@L8")
	assert.Contains(t, got, "`x == y` → `x != y`")
	assert.Contains(t, got, "`INVERT_NEGATIVES`@L12")
}

func Test_formatMutantsStr_detail_with_empty_text(t *testing.T) {
	details := []score.MutationDetail{
		{MutantType: "CONDITIONALS_BOUNDARY", Line: 10, Status: "LIVED", OriginalText: "", ReplacementText: ""},
	}
	got := formatMutantsStr(details)
	assert.Contains(t, got, "`CONDITIONALS_BOUNDARY`@L10")
	assert.NotContains(t, got, "→")
}

func TestPRCommentFormatter_Format_detailed_unreliable_with_mutation_details(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &scan.Entries{List: []score.CRAPEntry{
		{
			File:              "/project/main.go",
			Package:           "myapp",
			FuncName:          "UnreliableFunc",
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
		Threshold: 30,
		Writer:    buf,
		Detailed:  true,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "## \u26a0\ufe0f Unreliable Coverage")
	assert.Contains(t, output, "UnreliableFunc")
	assert.Contains(t, output, "`CONDITIONALS_BOUNDARY`@L15")
	assert.Contains(t, output, "`a < b` → `a >= b`")
}

func TestPRCommentFormatter_Format_25_total_no_truncation_message(t *testing.T) {
	// COND_BOUND :47 — with exactly 25 total entries, no "…and" truncation
	// message should appear. Mutant >= would print "…and 0 more".
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}
	entries := &scan.Entries{List: make([]score.CRAPEntry, 25)}
	for i := range 25 {
		entries.List[i] = score.CRAPEntry{
			File: "/project/main.go", Package: "myapp",
			FuncName:   fmt.Sprintf("Func%d", i),
			Line:       i + 1,
			Complexity: 10,
			Coverage:   0,
			CRAP:       float64(100 - i),
		}
	}
	opts := FormatOptions{Threshold: 0, Writer: buf}
	err := f.Format(entries, opts)
	require.NoError(t, err)
	output := buf.String()
	assert.NotContains(t, output, "…and")
}

func TestPRCommentFormatter_Format_unreliable_non_detailed_exact_mutation_score(t *testing.T) {
	// ARITH :76 — e.MutationScore*100 in non-detailed unreliable section.
	// Mutant would change *100 to /100 or +100, producing wrong percentage.
	f := &PRCommentFormatter{}
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
			MutationScore:     0.75,
		},
	}}
	opts := FormatOptions{Threshold: 30, Writer: buf, Detailed: false}
	err := f.Format(entries, opts)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "75.0%")
}

func Test_formatMutantsStr_no_leading_comma(t *testing.T) {
	// COND_BOUND :87 — i > 0 changed to i >= 0 would add leading comma.
	// COND_NEG :87 — negation would remove commas between items.
	details := []score.MutationDetail{
		{MutantType: "FIRST", Line: 1},
		{MutantType: "SECOND", Line: 2},
	}
	got := formatMutantsStr(details)
	assert.False(t, strings.HasPrefix(got, ","), "should not start with comma")
	assert.True(t, strings.HasPrefix(got, "`"), "should start with backtick")
	assert.Contains(t, got, ", `SECOND`")
}

func TestPRCommentFormatter_Format_25_entries_few_crappy_no_panic(t *testing.T) {
	// COND_BOUND :122 — len(entries.List) > maxPRCommentRows changed to >=
	// would try crappy[:25] with only 3 items → panic.
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}
	entries := &scan.Entries{List: make([]score.CRAPEntry, 25)}
	for i := range 25 {
		crap := 5.0
		if i < 3 {
			crap = 100.0
		}
		entries.List[i] = score.CRAPEntry{
			File: "/project/main.go", Package: "myapp",
			FuncName:   fmt.Sprintf("Func%d", i),
			Line:       i + 1,
			Complexity: 10,
			Coverage:   0,
			CRAP:       crap,
		}
	}
	opts := FormatOptions{Threshold: 30, Writer: buf}
	err := f.Format(entries, opts)
	require.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "3 crappy function(s)")
}

func TestPRCommentFormatter_Format_sort_equal_crap_no_panic(t *testing.T) {
	// COND_BOUND :114 — > changed to >= creates broken comparator for equal
	// EffectiveCRAP values. Go 1.22+ panics on broken comparators.
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}
	entries := &scan.Entries{List: make([]score.CRAPEntry, 20)}
	for i := range 20 {
		entries.List[i] = score.CRAPEntry{
			File: "/project/main.go", Package: "myapp",
			FuncName:   fmt.Sprintf("Func%d", i),
			Line:       i + 1,
			Complexity: 5,
			Coverage:   0,
			CRAP:       30,
		}
	}
	opts := FormatOptions{Threshold: 10, Writer: buf}
	err := f.Format(entries, opts)
	require.NoError(t, err)
}

func TestPRCommentFormatter_Format_detailed_unreliable_without_text_fields(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &scan.Entries{List: []score.CRAPEntry{
		{
			File:              "/project/main.go",
			Package:           "myapp",
			FuncName:          "UnreliableFunc",
			Line:              10,
			Complexity:        5,
			Coverage:          90.0,
			CRAP:              15.0,
			CoverageUntrusted: true,
			MutationScore:     0.5,
			MutationDetails: []score.MutationDetail{
				{MutantType: "CONTROL_FLOW", Line: 15, Status: "LIVED"},
			},
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
	assert.Contains(t, output, "## \u26a0\ufe0f Unreliable Coverage")
	assert.Contains(t, output, "UnreliableFunc")
	assert.Contains(t, output, "`CONTROL_FLOW`@L15")
	assert.NotContains(t, output, "→")
}
