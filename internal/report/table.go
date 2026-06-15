package report

import (
	"fmt"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/padiazg/go-crap/internal/score"
)

// TableFormatter outputs CRAP entries as a human-readable table.
type TableFormatter struct{}

func (f *TableFormatter) Format(entries *score.EntryList, opts FormatOptions) error {
	if entries == nil {
		return fmt.Errorf("Format: entries list shouldn't be nil")
	}

	sort.Slice(entries.List, func(i, j int) bool {
		effectiveI := entries.List[i].EffectiveScore()
		effectiveJ := entries.List[j].EffectiveScore()
		if effectiveI != effectiveJ {
			return effectiveI > effectiveJ
		}
		return entries.List[i].MutationScore < entries.List[j].MutationScore
	})

	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.AppendHeader(table.Row{"", "CRAP", "CC", "Coverage", "Function", "Location"})

	failed := 0
	halfThreshold := opts.Threshold / 2.0

	for _, e := range entries.List {
		effectiveCRAP := e.EffectiveScore()

		status := StatusSymbol(effectiveCRAP, opts.Threshold, halfThreshold)
		covBar := coverageBar(e.Coverage)

		if effectiveCRAP > opts.Threshold {
			failed++
		}

		loc := fmt.Sprintf("%s:%d", e.File, e.Line)
		if base := opts.BaseDir; base != "" {
			if rel := RelativizePath(e.File, base); rel != e.File {
				loc = fmt.Sprintf("%s:%d", rel, e.Line)
			}
		}

		covStr := fmt.Sprintf("%.1f%%", e.Coverage)
		if e.CoverageUntrusted {
			covStr += " \xe2\x9a\xa0"
		}

		t.AppendRow(table.Row{
			status,
			fmt.Sprintf("%.2f", effectiveCRAP),
			e.Complexity,
			fmt.Sprintf("%s %s", covBar, covStr),
			e.FuncName,
			loc,
		})
	}

	fmt.Fprintf(opts.Writer, "\n")
	fmt.Fprint(opts.Writer, t.Render())
	fmt.Fprintf(opts.Writer, "\n")

	total := len(entries.List)
	if total > 0 {
		fmt.Fprintf(opts.Writer, "%d/%d function(s) exceed threshold CRAP %.0f.\n", failed, total, opts.Threshold)
	}
	return nil
}

func StatusSymbol(crap, threshold, halfThreshold float64) string {
	switch {
	case crap > threshold:
		return "✗"
	case crap > halfThreshold:
		return "▲"
	default:
		return "✓"
	}
}

func coverageBar(pct float64) string {
	filled := int(pct / 10)
	empty := 10 - filled
	var bar strings.Builder
	for range filled {
		bar.WriteString("█")
	}
	for range empty {
		bar.WriteString("░")
	}
	return bar.String()
}
