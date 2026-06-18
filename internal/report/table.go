package report

import (
	"fmt"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/padiazg/go-crap/internal/scan"
	"github.com/padiazg/go-crap/internal/score"
)

// TableFormatter outputs CRAP entries as a human-readable table.
type TableFormatter struct{}

func (f *TableFormatter) Format(entries *scan.Entries, opts FormatOptions) error {
	if entries == nil {
		return fmt.Errorf("Format: entries list shouldn't be nil")
	}

	// sort.Slice(entries.List, func(i, j int) bool {
	// 	effectiveI := entries.List[i].EffectiveScore()
	// 	effectiveJ := entries.List[j].EffectiveScore()
	// 	if effectiveI != effectiveJ {
	// 		return effectiveI > effectiveJ
	// 	}
	// 	return entries.List[i].MutationScore < entries.List[j].MutationScore
	// })

	sorted := entries.ForTable()

	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.AppendHeader(table.Row{"", "CRAP", "CC", "Coverage", "Function", "Location"})

	failed := 0
	halfThreshold := opts.Threshold / 2.0

	for _, e := range sorted {
		if e.EffectiveCRAP > opts.Threshold {
			failed++
		}
		t.AppendRow(f.formatTableRow(e, opts, halfThreshold))
	}

	fmt.Fprintf(opts.Writer, "\n")
	fmt.Fprint(opts.Writer, t.Render())
	fmt.Fprintf(opts.Writer, "\n")

	total := len(sorted)
	if total > 0 {
		fmt.Fprintf(opts.Writer, "%d/%d function(s) exceed threshold CRAP %.0f.\n", failed, total, opts.Threshold)
	}

	return nil
}

func (f *TableFormatter) formatTableRow(e score.CRAPEntry, opts FormatOptions, halfThreshold float64) table.Row {
	effectiveCRAP := e.EffectiveScore()
	status := StatusSymbol(effectiveCRAP, opts.Threshold, halfThreshold)
	covBar := coverageBar(e.Coverage)
	loc := f.formatLocation(e, opts.BaseDir)
	covStr := f.formatCoverageString(e)

	return table.Row{
		status,
		fmt.Sprintf("%.2f", effectiveCRAP),
		e.Complexity,
		fmt.Sprintf("%s %s", covBar, covStr),
		e.FuncName,
		loc,
	}
}

func (f *TableFormatter) formatLocation(e score.CRAPEntry, baseDir string) string {
	loc := fmt.Sprintf("%s:%d", e.File, e.Line)
	if baseDir != "" {
		if rel := RelativizePath(e.File, baseDir); rel != e.File {
			loc = fmt.Sprintf("%s:%d", rel, e.Line)
		}
	}
	return loc
}

func (f *TableFormatter) formatCoverageString(e score.CRAPEntry) string {
	covStr := fmt.Sprintf("%.1f%%", e.Coverage)
	if e.CoverageUntrusted {
		covStr += " \xe2\x9a\xa0"
	}
	return covStr
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
