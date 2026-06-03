package report

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/padiazg/go-crap/internal/score"
)

type TableFormatter struct{}

func (f *TableFormatter) Format(entries []score.CRAPEntry, opts FormatOptions) error {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CRAP > entries[j].CRAP
	})

	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.AppendHeader(table.Row{"", "CRAP", "CC", "Coverage", "Function", "Location"})

	failed := 0
	halfThreshold := opts.Threshold / 2.0

	for _, e := range entries {
		status := statusSymbol(e.CRAP, opts.Threshold, halfThreshold)
		covBar := coverageBar(e.Coverage)

		if e.CRAP > opts.Threshold {
			failed++
		}

		loc := fmt.Sprintf("%s:%d", e.File, e.Line)
		if base := opts.BaseDir; base != "" {
			if absBase, err := filepath.Abs(base); err == nil {
				if rel, err := filepath.Rel(absBase, e.File); err == nil && rel != e.File {
					loc = fmt.Sprintf("%s:%d", rel, e.Line)
				}
			}
		}

		t.AppendRow(table.Row{
			status,
			fmt.Sprintf("%.2f", e.CRAP),
			e.Complexity,
			fmt.Sprintf("%s %.1f%%", covBar, e.Coverage),
			e.FuncName,
			loc,
		})
	}

	fmt.Fprintf(opts.Writer, "\n")
	fmt.Fprint(opts.Writer, t.Render())
	fmt.Fprintf(opts.Writer, "\n")

	total := len(entries)
	if total > 0 {
		fmt.Fprintf(opts.Writer, "%d/%d function(s) exceed threshold CRAP %.0f.\n", failed, total, opts.Threshold)
	}
	return nil
}

func statusSymbol(crap, threshold, halfThreshold float64) string {
	if crap > threshold {
		return "✗"
	}
	if crap > halfThreshold {
		return "▲"
	}
	return "✓"
}

func coverageBar(pct float64) string {
	filled := int(pct / 10)
	empty := 10 - filled
	bar := ""
	for i := 0; i < filled; i++ {
		bar += "█"
	}
	for i := 0; i < empty; i++ {
		bar += "░"
	}
	return bar
}
