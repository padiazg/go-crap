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
	lines := strings.SplitSeq(output, "\n")
	for line := range lines {
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

func TestPRCommentFormatter_Format_many_total_few_crappy_no_panic(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}
	entries := &scan.Entries{List: make([]score.CRAPEntry, 30)}
	for i := range 30 {
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

func TestPRCommentFormatter_Format_25_entries_few_crappy_no_panic(t *testing.T) {
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

func TestPRCommentFormatter_Format_summary_nil_backward_compat(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &scan.Entries{List: []score.CRAPEntry{
		{File: "/project/main.go", Package: "myapp", FuncName: "Foo", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1},
	}}

	opts := FormatOptions{
		Threshold: 30,
		Writer:    buf,
		Summary:   nil,
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	output := buf.String()
	assert.NotContains(t, output, "Combined CRAP")
	assert.NotContains(t, output, "Average CRAP")
	assert.Contains(t, output, "1 function(s) analyzed")
}

func TestPRCommentFormatter_Format_summary_non_nil(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &scan.Entries{List: []score.CRAPEntry{
		{File: "/project/main.go", Package: "myapp", FuncName: "Foo", Line: 1, Complexity: 1, Coverage: 100, CRAP: 1},
		{File: "/project/main.go", Package: "myapp", FuncName: "Bar", Line: 10, Complexity: 10, Coverage: 0, CRAP: 100},
	}}

	opts := FormatOptions{
		Threshold: 30,
		Writer:    buf,
		Summary:   &Summary{Combined: 101, Average: 50.5, TotalFuncs: 2, Exceeded: 1},
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Combined CRAP: 101.00")
	assert.Contains(t, output, "Average CRAP: 50.50")
	assert.Contains(t, output, "2 function(s) analyzed")
}

func TestPRCommentFormatter_Format_summary_empty_entries(t *testing.T) {
	f := &PRCommentFormatter{}
	buf := &bytes.Buffer{}

	entries := &scan.Entries{List: []score.CRAPEntry{}}

	opts := FormatOptions{
		Threshold: 30,
		Writer:    buf,
		Summary:   &Summary{Combined: 0, Average: 0, TotalFuncs: 0, Exceeded: 0},
	}

	err := f.Format(entries, opts)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Combined CRAP: 0.00")
	assert.Contains(t, output, "Average CRAP: 0.00")
	assert.Contains(t, output, "0 function(s) analyzed")
}

type checkRegressionsOutputFn func(*testing.T, string)

func checkRegressionsOutputContains(want string) checkRegressionsOutputFn {
	return func(t *testing.T, got string) {
		t.Helper()
		assert.Containsf(t, got, want, "output should contain %q", want)
	}
}

func checkRegressionsOutputNotContains(want string) checkRegressionsOutputFn {
	return func(t *testing.T, got string) {
		t.Helper()
		assert.NotContainsf(t, got, want, "output should not contain %q", want)
	}
}

func checkRegressionsOutputOrder(before, after string) checkRegressionsOutputFn {
	return func(t *testing.T, got string) {
		t.Helper()
		bi := strings.Index(got, before)
		ai := strings.Index(got, after)
		if assert.GreaterOrEqualf(t, bi, 0, "output should contain %q", before) {
			if assert.GreaterOrEqualf(t, ai, 0, "output should contain %q", after) {
				assert.Lessf(t, bi, ai, "%q should appear before %q in output", before, after)
			}
		}
	}
}

func checkRegressionsOutputEmpty() checkRegressionsOutputFn {
	return func(t *testing.T, got string) {
		t.Helper()
		assert.Empty(t, got, "output should be empty")
	}
}

func TestPRCommentFormatter_writeRegressionsSection(t *testing.T) {
	tests := []struct {
		name          string
		w             *bytes.Buffer
		sorted        []score.CRAPEntry
		baseDir       string
		ignoreCovered bool
		checks        []checkRegressionsOutputFn
	}{
		{
			name:   "no_regressions_empty_sorted",
			sorted: []score.CRAPEntry{},
			checks: []checkRegressionsOutputFn{checkRegressionsOutputEmpty()},
		},
		{
			name: "no_regressions_baseline_negative",
			sorted: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "NewFunc", Line: 1, BaselineCRAP: -1, EffectiveCRAP: 10},
				{File: "/project/main.go", Package: "myapp", FuncName: "AlsoNew", Line: 5, BaselineCRAP: -1, EffectiveCRAP: 20},
			},
			checks: []checkRegressionsOutputFn{checkRegressionsOutputEmpty()},
		},
		{
			name: "no_regressions_delta_below_tolerance",
			sorted: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Almost", Line: 1, BaselineCRAP: 0, EffectiveCRAP: 0.009},
			},
			checks: []checkRegressionsOutputFn{checkRegressionsOutputEmpty()},
		},
		{
			name: "single_regression",
			sorted: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Degraded", Line: 10, BaselineCRAP: 0, EffectiveCRAP: 10},
			},
			checks: []checkRegressionsOutputFn{checkRegressionsOutputContains("## \U0001f534 Regressions")},
		},
		{
			name: "multiple_regressions_sorted_by_delta_desc",
			sorted: []score.CRAPEntry{
				{File: "/project/a.go", Package: "myapp", FuncName: "SmallDelta", Line: 1, BaselineCRAP: 0, EffectiveCRAP: 5},
				{File: "/project/b.go", Package: "myapp", FuncName: "BigDelta", Line: 2, BaselineCRAP: 0, EffectiveCRAP: 15},
				{File: "/project/c.go", Package: "myapp", FuncName: "MidDelta", Line: 3, BaselineCRAP: 0, EffectiveCRAP: 10},
			},
			checks: []checkRegressionsOutputFn{
				checkRegressionsOutputContains("## \U0001f534 Regressions"),
				checkRegressionsOutputOrder("BigDelta", "MidDelta"),
				checkRegressionsOutputOrder("MidDelta", "SmallDelta"),
			},
		},
		{
			name: "delta_at_tolerance_boundary_above",
			sorted: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "JustAbove", Line: 1, BaselineCRAP: 0, EffectiveCRAP: 0.011},
			},
			checks: []checkRegressionsOutputFn{checkRegressionsOutputContains("## \U0001f534 Regressions")},
		},
		{
			name: "delta_at_tolerance_boundary_below",
			sorted: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "JustBelow", Line: 1, BaselineCRAP: 0, EffectiveCRAP: 0.009},
			},
			checks: []checkRegressionsOutputFn{checkRegressionsOutputEmpty()},
		},
		{
			name: "ignore_covered_true_all_covered_no_regressions",
			sorted: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "FullyCovered", Line: 1, BaselineCRAP: 0, EffectiveCRAP: 20, Coverage: 100},
			},
			ignoreCovered: true,
			checks:        []checkRegressionsOutputFn{checkRegressionsOutputEmpty()},
		},
		{
			name: "ignore_covered_true_uncovered_regression",
			sorted: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "PartiallyCovered", Line: 1, BaselineCRAP: 0, EffectiveCRAP: 20, Coverage: 50},
			},
			ignoreCovered: true,
			checks: []checkRegressionsOutputFn{
				checkRegressionsOutputContains("## \U0001f534 Regressions"),
				checkRegressionsOutputContains("PartiallyCovered"),
				checkRegressionsOutputNotContains("## Ignored"),
			},
		},
		{
			name: "ignore_covered_mixed_regressed_and_ignored",
			sorted: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "IgnoredFunc", Line: 1, BaselineCRAP: 0, EffectiveCRAP: 20, Coverage: 100},
				{File: "/project/main.go", Package: "myapp", FuncName: "RegressedFunc", Line: 2, BaselineCRAP: 0, EffectiveCRAP: 10, Coverage: 50},
			},
			ignoreCovered: true,
			checks: []checkRegressionsOutputFn{
				checkRegressionsOutputContains("## \U0001f534 Regressions"),
				checkRegressionsOutputContains("RegressedFunc"),
				checkRegressionsOutputContains("## Ignored (fully covered)"),
				checkRegressionsOutputContains("IgnoredFunc"),
			},
		},
		{
			name: "ignore_covered_false_covered_regression",
			sorted: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "FullyCovered", Line: 1, BaselineCRAP: 0, EffectiveCRAP: 20, Coverage: 100},
			},
			ignoreCovered: false,
			checks: []checkRegressionsOutputFn{
				checkRegressionsOutputContains("## \U0001f534 Regressions"),
				checkRegressionsOutputContains("FullyCovered"),
				checkRegressionsOutputNotContains("## Ignored"),
			},
		},
		{
			name: "baseline_crap_zero",
			sorted: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "ZeroBaseline", Line: 1, BaselineCRAP: 0, EffectiveCRAP: 5},
			},
			checks: []checkRegressionsOutputFn{checkRegressionsOutputContains("## \U0001f534 Regressions")},
		},
		{
			name: "negative_delta_not_shown",
			sorted: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Improved", Line: 1, BaselineCRAP: 10, EffectiveCRAP: 5},
			},
			checks: []checkRegressionsOutputFn{checkRegressionsOutputEmpty()},
		},
		{
			name: "with_base_dir_relativized",
			sorted: []score.CRAPEntry{
				{File: "/tmp/project/main.go", Package: "myapp", FuncName: "Relocated", Line: 15, BaselineCRAP: 0, EffectiveCRAP: 10},
			},
			baseDir: "/tmp/project",
			checks: []checkRegressionsOutputFn{
				checkRegressionsOutputContains("## \U0001f534 Regressions"),
				checkRegressionsOutputContains("`main.go:15`"),
				checkRegressionsOutputNotContains("`/tmp/project/main.go:15`"),
			},
		},
		{
			name: "ignore_covered_tilde_in_ignored_section",
			sorted: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "Regressed", Line: 1, BaselineCRAP: 0, EffectiveCRAP: 10, Coverage: 50},
				{File: "/project/main.go", Package: "myapp", FuncName: "CoveredDegraded", Line: 2, BaselineCRAP: 20, EffectiveCRAP: 30, Coverage: 99.95},
			},
			ignoreCovered: true,
			checks: []checkRegressionsOutputFn{
				checkRegressionsOutputContains("## \U0001f534 Regressions"),
				checkRegressionsOutputContains("Regressed"),
				checkRegressionsOutputContains("## Ignored (fully covered)"),
				checkRegressionsOutputContains("CoveredDegraded"),
				checkRegressionsOutputContains("~"),
				checkRegressionsOutputContains("+10.0 ~"),
			},
		},
		{
			name: "delta_at_tolerance_boundary_exact",
			sorted: []score.CRAPEntry{
				{File: "/project/main.go", Package: "myapp", FuncName: "ExactlyAt", Line: 1, BaselineCRAP: 0, EffectiveCRAP: 0.01},
			},
			checks: []checkRegressionsOutputFn{checkRegressionsOutputEmpty()},
		},
		{
			name: "multiple_regressions_with_negative_deltas",
			sorted: []score.CRAPEntry{
				{File: "/project/a.go", Package: "myapp", FuncName: "Improved", Line: 1, BaselineCRAP: 10, EffectiveCRAP: 5},
				{File: "/project/b.go", Package: "myapp", FuncName: "Regressed", Line: 2, BaselineCRAP: 0, EffectiveCRAP: 15},
				{File: "/project/c.go", Package: "myapp", FuncName: "WorseRegressed", Line: 3, BaselineCRAP: 0, EffectiveCRAP: 25},
			},
			checks: []checkRegressionsOutputFn{
				checkRegressionsOutputContains("## \U0001f534 Regressions"),
				checkRegressionsOutputContains("WorseRegressed"),
				checkRegressionsOutputContains("Regressed"),
				checkRegressionsOutputNotContains("Improved"),
				checkRegressionsOutputOrder("WorseRegressed", "Regressed"),
			},
		},
		{
			name: "sort_delta_with_nonzero_baselines",
			sorted: []score.CRAPEntry{
				{File: "/project/a.go", Package: "myapp", FuncName: "HighBaselineLowDelta", Line: 1, BaselineCRAP: 0, EffectiveCRAP: 25},
				{File: "/project/b.go", Package: "myapp", FuncName: "LowBaselineHighDelta", Line: 2, BaselineCRAP: 10, EffectiveCRAP: 30},
			},
			checks: []checkRegressionsOutputFn{
				checkRegressionsOutputContains("## \U0001f534 Regressions"),
				checkRegressionsOutputOrder("HighBaselineLowDelta", "LowBaselineHighDelta"),
				checkRegressionsOutputContains("+25.0"),
				checkRegressionsOutputContains("+20.0"),
			},
		},
		{
			name: "sort_delta_with_nonzero_i_baseline",
			sorted: []score.CRAPEntry{
				{File: "/project/a.go", Package: "myapp", FuncName: "HighBaselineFirst", Line: 1, BaselineCRAP: 5, EffectiveCRAP: 30},
				{File: "/project/b.go", Package: "myapp", FuncName: "LowBaselineSwapper", Line: 2, BaselineCRAP: 0, EffectiveCRAP: 28},
			},
			checks: []checkRegressionsOutputFn{
				checkRegressionsOutputContains("## \U0001f534 Regressions"),
				checkRegressionsOutputOrder("LowBaselineSwapper", "HighBaselineFirst"),
				checkRegressionsOutputContains("+28.0"),
				checkRegressionsOutputContains("+25.0"),
			},
		},
		{
			name: "sort_stable_with_equal_deltas",
			sorted: []score.CRAPEntry{
				{File: "/project/a.go", Package: "myapp", FuncName: "FirstEqual", Line: 1, BaselineCRAP: 10, EffectiveCRAP: 20},
				{File: "/project/b.go", Package: "myapp", FuncName: "SecondEqual", Line: 2, BaselineCRAP: 0, EffectiveCRAP: 10},
			},
			checks: []checkRegressionsOutputFn{
				checkRegressionsOutputContains("## \U0001f534 Regressions"),
				checkRegressionsOutputOrder("FirstEqual", "SecondEqual"),
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f := &PRCommentFormatter{}
			if tt.w == nil {
				tt.w = &bytes.Buffer{}
			}
			f.writeRegressionsSection(tt.w, tt.sorted, tt.baseDir, tt.ignoreCovered)
			for _, c := range tt.checks {
				c(t, tt.w.String())
			}
		})
	}
}

func Test_formatPRDelta(t *testing.T) {
	tests := []struct {
		name          string
		e             score.CRAPEntry
		ignoreCovered bool
		want          string
	}{
		{
			name:          "new_function",
			e:             score.CRAPEntry{FuncName: "New", BaselineCRAP: -1, EffectiveCRAP: 10},
			ignoreCovered: false,
			want:          "[NEW]",
		},
		{
			name:          "negative_baseline_new",
			e:             score.CRAPEntry{FuncName: "AlsoNew", BaselineCRAP: -0.5, EffectiveCRAP: 10},
			ignoreCovered: false,
			want:          "[NEW]",
		},
		{
			name:          "regression",
			e:             score.CRAPEntry{FuncName: "Regressed", BaselineCRAP: 0, EffectiveCRAP: 10},
			ignoreCovered: false,
			want:          "+10.0 \U0001f534",
		},
		{
			name:          "improvement",
			e:             score.CRAPEntry{FuncName: "Improved", BaselineCRAP: 10, EffectiveCRAP: 0},
			ignoreCovered: false,
			want:          "-10.0 \U0001f7e2",
		},
		{
			name:          "no_change_equal_values",
			e:             score.CRAPEntry{FuncName: "Same", BaselineCRAP: 5, EffectiveCRAP: 5},
			ignoreCovered: false,
			want:          "-",
		},
		{
			name:          "zero_delta",
			e:             score.CRAPEntry{FuncName: "Zero", BaselineCRAP: 0, EffectiveCRAP: 0},
			ignoreCovered: true,
			want:          "-",
		},
		{
			name:          "delta_at_upper_tolerance",
			e:             score.CRAPEntry{FuncName: "Upper", BaselineCRAP: 0, EffectiveCRAP: 0.01},
			ignoreCovered: false,
			want:          "-",
		},
		{
			name:          "delta_at_lower_tolerance",
			e:             score.CRAPEntry{FuncName: "Lower", BaselineCRAP: 0, EffectiveCRAP: -0.01},
			ignoreCovered: false,
			want:          "-",
		},
		{
			name:          "delta_just_above_tolerance",
			e:             score.CRAPEntry{FuncName: "Above", BaselineCRAP: 0, EffectiveCRAP: 0.011},
			ignoreCovered: false,
			want:          "+0.0 \U0001f534",
		},
		{
			name:          "delta_just_below_tolerance",
			e:             score.CRAPEntry{FuncName: "Below", BaselineCRAP: 0, EffectiveCRAP: -0.011},
			ignoreCovered: false,
			want:          "-0.0 \U0001f7e2",
		},
		{
			name:          "regression_ignored_fully_covered",
			e:             score.CRAPEntry{FuncName: "CoveredReg", BaselineCRAP: 0, EffectiveCRAP: 10, Coverage: 100},
			ignoreCovered: true,
			want:          "+10.0 ~",
		},
		{
			name:          "improvement_ignored_fully_covered",
			e:             score.CRAPEntry{FuncName: "CoveredImp", BaselineCRAP: 10, EffectiveCRAP: 0, Coverage: 100},
			ignoreCovered: true,
			want:          "-10.0 ~",
		},
		{
			name:          "regression_not_ignored_below_coverage",
			e:             score.CRAPEntry{FuncName: "AlmostCovered", BaselineCRAP: 0, EffectiveCRAP: 10, Coverage: 99.94},
			ignoreCovered: true,
			want:          "+10.0 \U0001f534",
		},
		{
			name:          "improvement_not_ignored_below_coverage",
			e:             score.CRAPEntry{FuncName: "AlmostCovered2", BaselineCRAP: 10, EffectiveCRAP: 0, Coverage: 99.94},
			ignoreCovered: true,
			want:          "-10.0 \U0001f7e2",
		},
		{
			name:          "regression_fully_covered_ignore_false",
			e:             score.CRAPEntry{FuncName: "CoveredNoIgnore", BaselineCRAP: 0, EffectiveCRAP: 10, Coverage: 100},
			ignoreCovered: false,
			want:          "+10.0 \U0001f534",
		},
		{
			name:          "improvement_fully_covered_ignore_false",
			e:             score.CRAPEntry{FuncName: "CoveredNoIgnore2", BaselineCRAP: 10, EffectiveCRAP: 0, Coverage: 100},
			ignoreCovered: false,
			want:          "-10.0 \U0001f7e2",
		},
		{
			name:          "large_regression",
			e:             score.CRAPEntry{FuncName: "HugeRegression", BaselineCRAP: 0, EffectiveCRAP: 1000},
			ignoreCovered: false,
			want:          "+1000.0 \U0001f534",
		},
		{
			name:          "large_improvement",
			e:             score.CRAPEntry{FuncName: "HugeImprovement", BaselineCRAP: 1000, EffectiveCRAP: 0},
			ignoreCovered: false,
			want:          "-1000.0 \U0001f7e2",
		},
		{
			name:          "ignore_covered_with_negative_delta_coverage_high",
			e:             score.CRAPEntry{FuncName: "NegDeltaCovered", BaselineCRAP: 100, EffectiveCRAP: 50, Coverage: 100},
			ignoreCovered: true,
			want:          "-50.0 ~",
		},
		{
			name:          "improvement_ignore_covered_exactly_at_99.95",
			e:             score.CRAPEntry{FuncName: "ImpExactBoundary", BaselineCRAP: 100, EffectiveCRAP: 50, Coverage: 99.95},
			ignoreCovered: true,
			want:          "-50.0 ~",
		},
		{
			name:          "improvement_ignore_covered_just_below_99.95",
			e:             score.CRAPEntry{FuncName: "ImpBelowBoundary", BaselineCRAP: 100, EffectiveCRAP: 50, Coverage: 99.94},
			ignoreCovered: true,
			want:          "-50.0 \U0001f7e2",
		},
		{
			name:          "regression_ignore_covered_exactly_at_99.95",
			e:             score.CRAPEntry{FuncName: "ExactBoundary", BaselineCRAP: 0, EffectiveCRAP: 5, Coverage: 99.95},
			ignoreCovered: true,
			want:          "+5.0 ~",
		},
		{
			name:          "regression_ignore_covered_just_below_99.95",
			e:             score.CRAPEntry{FuncName: "BelowBoundary", BaselineCRAP: 0, EffectiveCRAP: 5, Coverage: 99.94},
			ignoreCovered: true,
			want:          "+5.0 \U0001f534",
		},
		{
			name:          "baseline_equals_effective_ignore_covered",
			e:             score.CRAPEntry{FuncName: "SameIgnore", BaselineCRAP: 50, EffectiveCRAP: 50, Coverage: 80},
			ignoreCovered: true,
			want:          "-",
		},
		{
			name:          "new_function_with_high_coverage",
			e:             score.CRAPEntry{FuncName: "NewCovered", BaselineCRAP: -1, EffectiveCRAP: 10, Coverage: 100},
			ignoreCovered: true,
			want:          "[NEW]",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r := formatPRDelta(tt.e, tt.ignoreCovered)
			assert.Equal(t, tt.want, r)
		})
	}
}

type checkPRCommentFormatterwriteSummaryTableFn func(*testing.T, string)

var checkPRCommentFormatterwriteSummaryTable = func(fns ...checkPRCommentFormatterwriteSummaryTableFn) []checkPRCommentFormatterwriteSummaryTableFn {
	return fns
}

func TestPRCommentFormatter_writeSummaryTable(t *testing.T) {
	checkOutputContains := func(want string) checkPRCommentFormatterwriteSummaryTableFn {
		return func(t *testing.T, got string) {
			t.Helper()
			assert.Containsf(t, got, want, "output should contain %q", want)
		}
	}

	tests := []struct {
		name       string
		summary    *Summary
		baseline   *Baseline
		totalFuncs int
		exceeded   int
		checks     []checkPRCommentFormatterwriteSummaryTableFn
		before     func(*PRCommentFormatter)
	}{
		{
			name: "no_changes",
			summary: &Summary{
				Combined:         50,
				Average:          25,
				Exceeded:         5,
				BaselineCombined: 50,
				BaselineAverage:  25,
			},
			baseline:   &Baseline{Summary: Summary{TotalFuncs: 10}},
			totalFuncs: 10,
			exceeded:   5,
			checks: checkPRCommentFormatterwriteSummaryTable(
				checkOutputContains("## Summary"),
				checkOutputContains("| Metric | Baseline | Current | Δ |"),
				checkOutputContains("| Combined CRAP | 50.0 | 50.0 | - |"),
				checkOutputContains("| Average CRAP | 25.0 | 25.0 | - |"),
				checkOutputContains("| Functions exceeding threshold | 10 | 5 |  |"),
				checkOutputContains("| Total functions | 10 | 10 | |"),
			),
		},
		{
			name: "positive_deltas",
			summary: &Summary{
				Combined:         150,
				Average:          75,
				Exceeded:         3,
				BaselineCombined: 100,
				BaselineAverage:  50,
				DeltaCombined:    50,
				DeltaAverage:     25,
			},
			baseline:   &Baseline{Summary: Summary{TotalFuncs: 8}},
			totalFuncs: 10,
			exceeded:   5,
			checks: checkPRCommentFormatterwriteSummaryTable(
				checkOutputContains("| Combined CRAP | 100.0 | 150.0 | +50.0 🔴 |"),
				checkOutputContains("| Average CRAP | 50.0 | 75.0 | +25.0 🔴 |"),
				checkOutputContains("| Functions exceeding threshold | 8 | 5 | +2 🔴 |"),
				checkOutputContains("| Total functions | 8 | 10 | |"),
			),
		},
		{
			name: "negative_deltas",
			summary: &Summary{
				Combined:         50,
				Average:          25,
				Exceeded:         1,
				BaselineCombined: 100,
				BaselineAverage:  50,
				DeltaCombined:    -50,
				DeltaAverage:     -25,
			},
			baseline:   &Baseline{Summary: Summary{TotalFuncs: 10}},
			totalFuncs: 5,
			exceeded:   1,
			checks: checkPRCommentFormatterwriteSummaryTable(
				checkOutputContains("| Combined CRAP | 100.0 | 50.0 | -50.0 🟢 |"),
				checkOutputContains("| Average CRAP | 50.0 | 25.0 | -25.0 🟢 |"),
				checkOutputContains("| Total functions | 10 | 5 | |"),
			),
		},
		{
			name: "exceeded_worsened",
			summary: &Summary{
				Combined:         50,
				Average:          25,
				Exceeded:         3,
				BaselineCombined: 50,
				BaselineAverage:  25,
			},
			baseline:   &Baseline{Summary: Summary{TotalFuncs: 10}},
			totalFuncs: 12,
			exceeded:   8,
			checks: checkPRCommentFormatterwriteSummaryTable(
				checkOutputContains("| Combined CRAP | 50.0 | 50.0 | - |"),
				checkOutputContains("| Average CRAP | 25.0 | 25.0 | - |"),
				checkOutputContains("| Functions exceeding threshold | 10 | 8 | +5 🔴 |"),
				checkOutputContains("| Total functions | 10 | 12 | |"),
			),
		},
		{
			name: "exceeded_improved",
			summary: &Summary{
				Combined:         50,
				Average:          25,
				Exceeded:         5,
				BaselineCombined: 50,
				BaselineAverage:  25,
			},
			baseline:   &Baseline{Summary: Summary{TotalFuncs: 10}},
			totalFuncs: 10,
			exceeded:   2,
			checks: checkPRCommentFormatterwriteSummaryTable(
				checkOutputContains("| Functions exceeding threshold | 10 | 2 | -3 🟢 |"),
			),
		},
		{
			name: "mixed_deltas",
			summary: &Summary{
				Combined:         200,
				Average:          20,
				Exceeded:         2,
				BaselineCombined: 100,
				BaselineAverage:  40,
				DeltaCombined:    100,
				DeltaAverage:     -20,
			},
			baseline:   &Baseline{Summary: Summary{TotalFuncs: 15}},
			totalFuncs: 20,
			exceeded:   5,
			checks: checkPRCommentFormatterwriteSummaryTable(
				checkOutputContains("| Combined CRAP | 100.0 | 200.0 | +100.0 🔴 |"),
				checkOutputContains("| Average CRAP | 40.0 | 20.0 | -20.0 🟢 |"),
				checkOutputContains("| Functions exceeding threshold | 15 | 5 | +3 🔴 |"),
				checkOutputContains("| Total functions | 15 | 20 | |"),
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s := &PRCommentFormatter{}
			if tt.before != nil {
				tt.before(s)
			}

			buf := &bytes.Buffer{}
			s.writeSummaryTable(buf, tt.summary, tt.baseline, tt.totalFuncs, tt.exceeded)

			for _, c := range tt.checks {
				c(t, buf.String())
			}
		})
	}
}

func TestPRCommentFormatter_writeCrappyTable_baseline_delta_column(t *testing.T) {
	tests := []struct {
		name     string
		baseline *Baseline
		wantIn   string
		notIn    string
	}{
		{
			name:     "with_baseline_shows_delta_column",
			baseline: &Baseline{},
			wantIn:   "| Δ |",
			notIn:    "",
		},
		{
			name:     "without_baseline_no_delta_column",
			baseline: nil,
			wantIn:   "",
			notIn:    "| Δ |",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f := &PRCommentFormatter{}
			buf := &bytes.Buffer{}
			crappy := []score.CRAPEntry{
				{File: "/project/a.go", Package: "myapp", FuncName: "Bad", Line: 1, Complexity: 10, Coverage: 0, EffectiveCRAP: 100, BaselineCRAP: 10},
			}
			f.writeCrappyTable(buf, crappy, 1, "", tt.baseline, false)
			got := buf.String()
			if tt.wantIn != "" {
				assert.Contains(t, got, tt.wantIn)
			}
			if tt.notIn != "" {
				assert.NotContains(t, got, tt.notIn)
			}
			assert.Contains(t, got, "Bad")
		})
	}
}

func TestPRCommentFormatter_writeBadge(t *testing.T) {
	tests := []struct {
		name    string
		summary *Summary
		want    string
	}{
		{
			name:    "nil_summary_all_good",
			summary: nil,
			want:    "[OK] All good",
		},
		{
			name:    "zero_exceeded_all_good",
			summary: &Summary{Exceeded: 0, TotalFuncs: 10},
			want:    "[OK] All good",
		},
		{
			name:    "minor_changes",
			summary: &Summary{Exceeded: 2, TotalFuncs: 10},
			want:    "[!!] Minor changes",
		},
		{
			name:    "minor_changes_exactly_half",
			summary: &Summary{Exceeded: 5, TotalFuncs: 10},
			want:    "[!!] Minor changes",
		},
		{
			name:    "all_exceeded_small_total_error",
			summary: &Summary{Exceeded: 2, TotalFuncs: 2},
			want:    "[ERROR] Regressions detected",
		},
		{
			name:    "regressions_detected",
			summary: &Summary{Exceeded: 6, TotalFuncs: 10},
			want:    "[ERROR] Regressions detected",
		},
		{
			name:    "exceeded_with_zero_total",
			summary: &Summary{Exceeded: 1, TotalFuncs: 0},
			want:    "[ERROR] Regressions detected",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f := &PRCommentFormatter{}
			buf := &bytes.Buffer{}
			f.writeBadge(buf, tt.summary)
			assert.Equal(t, tt.want, buf.String())
		})
	}
}

func TestPRCommentFormatter_writePRHeader(t *testing.T) {
	tests := []struct {
		name      string
		sorted    []score.CRAPEntry
		crappy    []score.CRAPEntry
		threshold float64
		summary   *Summary
		baseline  *Baseline
		checks    []checkPRCommentFormatterOutputFn
	}{
		{
			name: "no_crappy",
			checks: checkPRCommentFormatterOutput(
				checkPRCommentOutputContains("## No crappy functions"),
			),
		},
		{
			name: "crappy_count",
			crappy: []score.CRAPEntry{
				{File: "/project/a.go", FuncName: "A", Line: 1},
				{File: "/project/b.go", FuncName: "B", Line: 2},
			},
			checks: checkPRCommentFormatterOutput(
				checkPRCommentOutputContains("## 2 crappy function(s)"),
			),
		},
		{
			name:     "summary_without_baseline_no_delta",
			summary:  &Summary{Combined: 100.5, Average: 50.25},
			baseline: nil,
			checks: checkPRCommentFormatterOutput(
				checkPRCommentOutputContains("Combined CRAP: 100.50"),
				checkPRCommentOutputContains("Average CRAP: 50.25"),
				checkPRCommentOutputNotContains("vs baseline"),
			),
		},
		{
			name: "summary_with_baseline_positive_delta",
			summary: &Summary{
				Combined:      100.5,
				Average:       50.25,
				DeltaCombined: 5.5,
				DeltaAverage:  2.25,
			},
			baseline: &Baseline{},
			checks: checkPRCommentFormatterOutput(
				checkPRCommentOutputContains("Combined CRAP: 100.50 (+5.50 vs baseline)"),
				checkPRCommentOutputContains("Average CRAP: 50.25 (+2.25 vs baseline)"),
			),
		},
		{
			name: "summary_with_baseline_negative_delta",
			summary: &Summary{
				Combined:      100.5,
				Average:       50.25,
				DeltaCombined: -5.5,
				DeltaAverage:  -2.25,
			},
			baseline: &Baseline{},
			checks: checkPRCommentFormatterOutput(
				checkPRCommentOutputContains("Combined CRAP: 100.50 (-5.50 vs baseline)"),
				checkPRCommentOutputContains("Average CRAP: 50.25 (-2.25 vs baseline)"),
			),
		},
		{
			name: "summary_with_baseline_zero_delta_no_suffix",
			summary: &Summary{
				Combined:      100.5,
				Average:       50.25,
				DeltaCombined: 0,
				DeltaAverage:  0,
			},
			baseline: &Baseline{},
			checks: checkPRCommentFormatterOutput(
				checkPRCommentOutputContains("Combined CRAP: 100.50"),
				checkPRCommentOutputContains("Average CRAP: 50.25"),
				checkPRCommentOutputNotContains("vs baseline"),
			),
		},
		{
			name:     "no_summary_no_crash",
			summary:  nil,
			baseline: &Baseline{},
			checks: checkPRCommentFormatterOutput(
				checkPRCommentOutputContains("function(s) analyzed"),
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			f := &PRCommentFormatter{}
			buf := &bytes.Buffer{}
			f.writePRHeader(buf, tt.sorted, tt.crappy, tt.threshold, tt.summary, tt.baseline)
			for _, c := range tt.checks {
				c(t, buf.String())
			}
		})
	}
}
