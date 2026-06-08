package report

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/padiazg/go-crap/internal/score"
)

const maxPRCommentRows = 25

type PRCommentFormatter struct{}

func (f *PRCommentFormatter) Format(entries *score.EntryList, opts FormatOptions) error {
	if entries == nil {
		return fmt.Errorf("Format: entries list shouldn't be nil")
	}

	sorted := make([]score.CRAPEntry, len(entries.List))
	copy(sorted, entries.List)
	for i := range sorted {
		if sorted[i].EffectiveCRAP == 0 {
			sorted[i].EffectiveCRAP = sorted[i].CRAP
		}
	}
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].EffectiveCRAP > sorted[j].EffectiveCRAP
	})

	halfThreshold := opts.Threshold / 2.0
	crappy := filterAboveThreshold(sorted, opts.Threshold)

	fmt.Fprintln(opts.Writer, "<!-- go-crap-report -->")
	fmt.Fprintln(opts.Writer)

	if len(crappy) == 0 {
		fmt.Fprintln(opts.Writer, "## No crappy functions")
	} else {
		fmt.Fprintf(opts.Writer, "## %d crappy function(s)\n", len(crappy))
	}

	fmt.Fprintf(opts.Writer, "\n%d function(s) analyzed · threshold %.0f\n\n", len(sorted), opts.Threshold)

	if len(crappy) == 0 {
		return nil
	}

	if len(crappy) > maxPRCommentRows {
		crappy = crappy[:maxPRCommentRows]
	}

	fmt.Fprintln(opts.Writer, "| | CRAP | CC | Cov % | Function | Location |")
	fmt.Fprintln(opts.Writer, "|---|---:|---:|---:|---|---|")

	for _, e := range crappy {
		status := StatusSymbol(e.EffectiveCRAP, opts.Threshold, halfThreshold)
		loc := formatPRLocation(e, opts.BaseDir)
		covStr := fmt.Sprintf("%.1f%%", e.Coverage)
		if e.CoverageUntrusted {
			covStr += " \xe2\x9a\xa0"
		}
		fmt.Fprintf(opts.Writer, "| %s | %.2f | %d | %s | `%s` | %s |\n",
			status, e.EffectiveCRAP, e.Complexity, covStr, e.FuncName, loc)
	}

	if len(entries.List) > maxPRCommentRows {
		fmt.Fprintf(opts.Writer, "\n…and %d more\n", len(entries.List)-maxPRCommentRows)
	}

	unreliable := filterUnreliableCoverage(sorted)
	if len(unreliable) > 0 {
		fmt.Fprintln(opts.Writer)
		fmt.Fprintln(opts.Writer, "## \u26a0\ufe0f Unreliable Coverage")
		fmt.Fprintln(opts.Writer)

		if opts.Detailed {
			fmt.Fprintln(opts.Writer, "| Function | CRAP | Effective CRAP | Mutation Score | Survived Mutants |")
			fmt.Fprintln(opts.Writer, "|---|---:|---:|---:|---|")
			for _, e := range unreliable {
				mutantsStr := ""
				if len(e.MutationDetails) > 0 {
					for i, md := range e.MutationDetails {
						if i > 0 {
							mutantsStr += ", "
						}
						mutantsStr += fmt.Sprintf("`%s`@L%d", md.MutantType, md.Line)
						if md.OriginalText != "" && md.ReplacementText != "" {
							mutantsStr += fmt.Sprintf("\n    `%s` → `%s`", md.OriginalText, md.ReplacementText)
						}
					}
				}
				fmt.Fprintf(opts.Writer, "| `%s` | %.2f | %.2f | %.1f%% | %s |\n",
					e.FuncName, e.CRAP, e.EffectiveCRAP, e.MutationScore*100, mutantsStr)
			}
		} else {
			fmt.Fprintln(opts.Writer, "| Function | CRAP | Effective CRAP | Mutation Score |")
			fmt.Fprintln(opts.Writer, "|---|---:|---:|---:|")
			for _, e := range unreliable {
				fmt.Fprintf(opts.Writer, "| `%s` | %.2f | %.2f | %.1f%% |\n",
					e.FuncName, e.CRAP, e.EffectiveCRAP, e.MutationScore*100)
			}
		}
	}

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
		if absBase, err := filepath.Abs(baseDir); err == nil {
			if rel, err := filepath.Rel(absBase, e.File); err == nil && rel != e.File {
				loc = fmt.Sprintf("`%s:%d`", rel, e.Line)
			}
		}
	}
	return loc
}
