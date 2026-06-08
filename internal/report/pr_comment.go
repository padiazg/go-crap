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
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].CRAP > sorted[j].CRAP
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
		status := StatusSymbol(e.CRAP, opts.Threshold, halfThreshold)
		loc := formatPRLocation(e, opts.BaseDir)
		fmt.Fprintf(opts.Writer, "| %s | %.2f | %d | %.1f%% | `%s` | %s |\n",
			status, e.CRAP, e.Complexity, e.Coverage, e.FuncName, loc)
	}

	if len(entries.List) > maxPRCommentRows {
		fmt.Fprintf(opts.Writer, "\n…and %d more\n", len(entries.List)-maxPRCommentRows)
	}

	return nil
}

func filterAboveThreshold(entries []score.CRAPEntry, threshold float64) []score.CRAPEntry {
	result := make([]score.CRAPEntry, 0)
	for _, e := range entries {
		if e.CRAP > threshold {
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
