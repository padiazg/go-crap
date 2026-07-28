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

const tableDeltaTolerance = 0.01

func isUnchanged(e score.CRAPEntry) bool {
	if e.BaselineCRAP < 0 {
		return false
	}
	delta := e.EffectiveCRAP - e.BaselineCRAP
	return delta >= -tableDeltaTolerance && delta <= tableDeltaTolerance
}

func anyChanged(sorted []score.CRAPEntry) bool {
	for _, e := range sorted {
		if !isUnchanged(e) {
			return true
		}
	}
	return false
}

func (f *TableFormatter) Format(entries *scan.Entries, opts FormatOptions) error {
	if entries == nil {
		return fmt.Errorf("Format: entries list shouldn't be nil")
	}

	sorted := entries.ForTable()

	if opts.Baseline != nil && !opts.ShowUnchanged {
		if !anyChanged(sorted) {
			fmt.Fprintln(opts.Writer, "\nNo changes since baseline.")
			f.printSummary(opts)
			return nil
		}
		filtered := make([]score.CRAPEntry, 0, len(sorted))
		for _, e := range sorted {
			if !isUnchanged(e) {
				filtered = append(filtered, e)
			}
		}
		sorted = filtered
	}

	var header table.Row
	if opts.Baseline != nil {
		header = table.Row{"", "CRAP", "CC", "Coverage", "Δ", "Function", "Location"}
	} else {
		header = table.Row{"", "CRAP", "CC", "Coverage", "Function", "Location"}
	}

	t := table.NewWriter()
	t.SetStyle(table.StyleLight)
	t.AppendHeader(header)

	failed := 0
	halfThreshold := opts.Threshold / 2.0
	var warningSet map[string]bool
	var warningSeen bool

	for _, e := range sorted {
		if e.EffectiveCRAP > opts.Threshold {
			failed++
		}
		t.AppendRow(f.formatTableRow(e, opts, halfThreshold))
		if e.CoverageWarning != "" && !warningSeen {
			warningSet = make(map[string]bool)
			warningSeen = true
		}
	}

	fmt.Fprintf(opts.Writer, "\n")
	fmt.Fprint(opts.Writer, t.Render())
	fmt.Fprintf(opts.Writer, "\n")

	total := len(sorted)
	if total > 0 {
		fmt.Fprintf(opts.Writer, "%d/%d function(s) exceed threshold CRAP %.0f.\n", failed, total, opts.Threshold)
	}

	f.printSummary(opts)

	if warningSeen {
		fmt.Fprintln(opts.Writer)
		for _, e := range sorted {
			warn := e.CoverageWarning
			if warn == "" {
				continue
			}
			if warningSet[warn] {
				continue
			}
			warningSet[warn] = true
			fmt.Fprintf(opts.Writer, "coverage unavailable for %s\n", warn)
		}
	}

	return nil
}

func (f *TableFormatter) printSummary(opts FormatOptions) {
	if opts.Summary == nil {
		return
	}
	if opts.Baseline != nil {
		fmt.Fprintf(opts.Writer, "Combined CRAP: %.2f (Δ%.2f vs baseline) | Average CRAP: %.2f (Δ%.2f vs baseline)\n",
			opts.Summary.Combined, opts.Summary.DeltaCombined,
			opts.Summary.Average, opts.Summary.DeltaAverage)
	} else {
		fmt.Fprintf(opts.Writer, "Combined CRAP: %.2f | Average CRAP: %.2f\n", opts.Summary.Combined, opts.Summary.Average)
	}
}

func (f *TableFormatter) formatTableRow(e score.CRAPEntry, opts FormatOptions, halfThreshold float64) table.Row {
	effectiveCRAP := e.EffectiveScore()
	status := StatusSymbol(effectiveCRAP, opts.Threshold, halfThreshold)
	covBar := coverageBar(e.Coverage)
	loc := f.formatLocation(e, opts.BaseDir)
	covStr := f.formatCoverageString(e)

	row := table.Row{
		status,
		fmt.Sprintf("%.2f", effectiveCRAP),
		e.Complexity,
		fmt.Sprintf("%s %s", covBar, covStr),
	}

	if opts.Baseline != nil {
		row = append(row, f.formatDelta(e, opts))
	}

	row = append(row, e.FuncName, loc)

	return row
}

func (f *TableFormatter) formatDelta(e score.CRAPEntry, opts FormatOptions) string {
	if e.BaselineCRAP < 0 {
		return "new"
	}
	delta := e.EffectiveCRAP - e.BaselineCRAP
	if delta > deltaTolerance {
		if opts.IgnoreCovered && e.Coverage >= 99.95 {
			return fmt.Sprintf("+%.1f ~", delta)
		}
		return fmt.Sprintf("+%.1f \u2191", delta)
	}
	if delta < -deltaTolerance {
		if opts.IgnoreCovered && e.Coverage >= 99.95 {
			return fmt.Sprintf("%.1f ~", delta)
		}
		return fmt.Sprintf("%.1f \u2193", delta)
	}
	return "-"
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
	if e.CoverageWarning != "" {
		return "N/A \xe2\x9a\xa1"
	}
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
