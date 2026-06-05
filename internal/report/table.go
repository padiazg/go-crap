package report

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/padiazg/go-crap/internal/score"
)

type TableFormatter struct{}

func (f *TableFormatter) Format(entries *score.EntryList, opts FormatOptions) error {
	if entries == nil {
		return fmt.Errorf("Format: entries list shouldn't be nil")
	}

	sort.Slice(entries.List, func(i, j int) bool {
		return entries.List[i].CRAP > entries.List[j].CRAP
	})

	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.AppendHeader(table.Row{"", "CRAP", "CC", "Coverage", "Function", "Location"})

	failed := 0
	halfThreshold := opts.Threshold / 2.0

	for _, e := range entries.List {
		status := StatusSymbol(e.CRAP, opts.Threshold, halfThreshold)
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
