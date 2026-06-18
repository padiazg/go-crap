package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/padiazg/go-crap/internal/scan"
	"github.com/padiazg/go-crap/internal/score"
)

func (f *PRCommentFormatter) writePRHeader(w io.Writer, sorted []score.CRAPEntry, crappy []score.CRAPEntry, threshold float64) {
	fmt.Fprintln(w, "<!-- go-crap-report -->")
	fmt.Fprintln(w)

	if len(crappy) == 0 {
		fmt.Fprintln(w, "## No crappy functions")
	} else {
		fmt.Fprintf(w, "## %d crappy function(s)\n", len(crappy))
	}

	fmt.Fprintf(w, "\n%d function(s) analyzed · threshold %.0f\n\n", len(sorted), threshold)
}

func (f *PRCommentFormatter) writeCrappyTable(w io.Writer, crappy []score.CRAPEntry, total int, baseDir string) {
	if len(crappy) == 0 {
		return
	}

	fmt.Fprintln(w, "| | CRAP | CC | Cov % | Function | Location |")
	fmt.Fprintln(w, "|---|---:|---:|---:|---|---|")

	for _, e := range crappy {
		loc := formatPRLocation(e, baseDir)
		covStr := fmt.Sprintf("%.1f%%", e.Coverage)
		if e.CoverageUntrusted {
			covStr += " \xe2\x9a\xa0"
		}
		fmt.Fprintf(w, "| ✗ | %.2f | %d | %s | `%s` | %s |\n",
			e.EffectiveCRAP, e.Complexity, covStr, e.FuncName, loc)
	}

	if total > maxPRCommentRows {
		fmt.Fprintf(w, "\n…and %d more\n", total-maxPRCommentRows)
	}

	fmt.Fprintln(w)
}

func (f *PRCommentFormatter) writeUnreliableSection(w io.Writer, unreliable []score.CRAPEntry, detailed bool) {
	if len(unreliable) == 0 {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "## \u26a0\ufe0f Unreliable Coverage")
	fmt.Fprintln(w)

	if detailed {
		fmt.Fprintln(w, "| Function | CRAP | Effective CRAP | Mutation Score | Survived Mutants |")
		fmt.Fprintln(w, "|---|---:|---:|---:|---|")
		for _, e := range unreliable {
			mutantsStr := formatMutantsStr(e.MutationDetails)
			fmt.Fprintf(w, "| `%s` | %.2f | %.2f | %.1f%% | %s |\n",
				e.FuncName, e.CRAP, e.EffectiveCRAP, e.MutationScore*100, mutantsStr)
		}
	} else {
		fmt.Fprintln(w, "| Function | CRAP | Effective CRAP | Mutation Score |")
		fmt.Fprintln(w, "|---|---:|---:|---:|")
		for _, e := range unreliable {
			fmt.Fprintf(w, "| `%s` | %.2f | %.2f | %.1f%% |\n",
				e.FuncName, e.CRAP, e.EffectiveCRAP, e.MutationScore*100)
		}
	}
}

func formatMutantsStr(details []score.MutationDetail) string {
	if len(details) == 0 {
		return ""
	}
	var mutantsStr strings.Builder
	for i, md := range details {
		if i > 0 {
			mutantsStr.WriteString(", ")
		}
		fmt.Fprintf(&mutantsStr, "`%s`@L%d", md.MutantType, md.Line)
		if md.OriginalText != "" && md.ReplacementText != "" {
			fmt.Fprintf(&mutantsStr, "\n    `%s` → `%s`", md.OriginalText, md.ReplacementText)
		}
	}
	return mutantsStr.String()
}

const maxPRCommentRows = 25

// PRCommentFormatter outputs CRAP entries as a GitHub PR comment.
type PRCommentFormatter struct{}

func (f *PRCommentFormatter) Format(entries *scan.Entries, opts FormatOptions) error {
	if entries == nil {
		return fmt.Errorf("Format: entries list shouldn't be nil")
	}

	sorted := entries.ForPRComment()

	crappy := filterAboveThreshold(sorted, opts.Threshold)

	f.writePRHeader(opts.Writer, sorted, crappy, opts.Threshold)

	if len(crappy) > maxPRCommentRows {
		crappy = crappy[:maxPRCommentRows]
	}

	f.writeCrappyTable(opts.Writer, crappy, len(entries.List), opts.BaseDir)

	unreliable := filterUnreliableCoverage(sorted)
	f.writeUnreliableSection(opts.Writer, unreliable, opts.Detailed)

	return nil
}

func filterAboveThreshold(entries []score.CRAPEntry, threshold float64) []score.CRAPEntry {
	result := make([]score.CRAPEntry, 0)
	for _, e := range entries {
		if e.EffectiveCRAP > threshold {
			result = append(result, e)
		}
	}
	return result
}

func filterUnreliableCoverage(entries []score.CRAPEntry) []score.CRAPEntry {
	result := make([]score.CRAPEntry, 0)
	for _, e := range entries {
		if e.CoverageUntrusted {
			result = append(result, e)
		}
	}
	return result
}

func formatPRLocation(e score.CRAPEntry, baseDir string) string {
	loc := fmt.Sprintf("`%s:%d`", e.File, e.Line)
	if baseDir != "" {
		if rel := RelativizePath(e.File, baseDir); rel != e.File {
			loc = fmt.Sprintf("`%s:%d`", rel, e.Line)
		}
	}
	return loc
}
