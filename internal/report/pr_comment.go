package report

import (
	"fmt"
	"io"
	"strings"

	"github.com/padiazg/go-crap/internal/scan"
	"github.com/padiazg/go-crap/internal/score"
)

const maxPRCommentRows = 25
const deltaTolerance = 0.01

func (f *PRCommentFormatter) writePRHeader(w io.Writer, sorted []score.CRAPEntry, crappy []score.CRAPEntry, threshold float64, summary *Summary, baseline *Baseline) {
	fmt.Fprintln(w, "<!-- go-crap-report -->")
	fmt.Fprintln(w)

	if len(crappy) == 0 {
		fmt.Fprintln(w, "## No crappy functions")
	} else {
		fmt.Fprintf(w, "## %d crappy function(s)\n", len(crappy))
	}

	fmt.Fprintf(w, "\n%d function(s) analyzed · threshold %.0f", len(sorted), threshold)
	if summary != nil {
		fmt.Fprintf(w, "\n")
		fmt.Fprint(w, "\n**CRAP Summary:** ")
		fmt.Fprintf(w, "Combined CRAP: %.2f", summary.Combined)
		if baseline != nil && summary.DeltaCombined != 0 {
			symbol := "+"
			if summary.DeltaCombined < 0 {
				symbol = ""
			}
			fmt.Fprintf(w, " (%s%.2f vs baseline)", symbol, summary.DeltaCombined)
		}
		fmt.Fprint(w, " · ")
		fmt.Fprintf(w, "Average CRAP: %.2f", summary.Average)
		if baseline != nil && summary.DeltaAverage != 0 {
			symbol := "+"
			if summary.DeltaAverage < 0 {
				symbol = ""
			}
			fmt.Fprintf(w, " (%s%.2f vs baseline)", symbol, summary.DeltaAverage)
		}
		fmt.Fprintln(w)
		fmt.Fprintln(w)

		f.writeBadge(w, summary)
		fmt.Fprintln(w)
	} else {
		fmt.Fprintln(w)
	}
}

func (f *PRCommentFormatter) writeBadge(w io.Writer, summary *Summary) {
	if summary == nil || summary.Exceeded == 0 {
		fmt.Fprint(w, "[OK] All good")
	} else if summary.TotalFuncs > 0 && summary.Exceeded <= summary.TotalFuncs/2 {
		fmt.Fprint(w, "[!!] Minor changes")
	} else {
		fmt.Fprint(w, "[ERROR] Regressions detected")
	}
}

func (f *PRCommentFormatter) writeCrappyTable(w io.Writer, crappy []score.CRAPEntry, total int, baseDir string, baseline *Baseline) {
	if len(crappy) == 0 {
		return
	}

	if baseline != nil {
		fmt.Fprintln(w, "| | CRAP | CC | Cov % | \u0394 | Function | Location |")
		fmt.Fprintln(w, "|---|---:|---:|---:|---:|---|---|")

		for _, e := range crappy {
			loc := formatPRLocation(e, baseDir)
			covStr := fmt.Sprintf("%.1f%%", e.Coverage)
			if e.CoverageUntrusted {
				covStr += " \xe2\x9a\xa0"
			}
			deltaStr := formatPRDelta(e)
			fmt.Fprintf(w, "| \xe2\x9c\x97 | %.2f | %d | %s | %s | `%s` | %s |\n",
				e.EffectiveCRAP, e.Complexity, covStr, deltaStr, e.FuncName, loc)
		}
	} else {
		fmt.Fprintln(w, "| | CRAP | CC | Cov % | Function | Location |")
		fmt.Fprintln(w, "|---|---:|---:|---:|---|---|")

		for _, e := range crappy {
			loc := formatPRLocation(e, baseDir)
			covStr := fmt.Sprintf("%.1f%%", e.Coverage)
			if e.CoverageUntrusted {
				covStr += " \xe2\x9a\xa0"
			}
			fmt.Fprintf(w, "| \xe2\x9c\x97 | %.2f | %d | %s | `%s` | %s |\n",
				e.EffectiveCRAP, e.Complexity, covStr, e.FuncName, loc)
		}
	}

	if total > maxPRCommentRows {
		fmt.Fprintf(w, "\n\xe2\x80\xa6and %d more\n", total-maxPRCommentRows)
	}

	fmt.Fprintln(w)
}

func formatPRDelta(e score.CRAPEntry) string {
	if e.BaselineCRAP < 0 {
		return "[NEW]"
	}
	delta := e.EffectiveCRAP - e.BaselineCRAP
	if delta > deltaTolerance {
		return fmt.Sprintf("+%.1f \U0001f534", delta)
	}
	if delta < -deltaTolerance {
		return fmt.Sprintf("%.1f \U0001f7e2", delta)
	}
	return "-"
}

func (f *PRCommentFormatter) writeNewFunctionsSection(w io.Writer, sorted []score.CRAPEntry, baseDir string) {
	newFuncs := filterNewFunctions(sorted)
	if len(newFuncs) == 0 {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "## New Functions")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Function | CRAP | CC | Location |")
	fmt.Fprintln(w, "|---|---:|---:|---|")

	for _, e := range newFuncs {
		loc := formatPRLocation(e, baseDir)
		fmt.Fprintf(w, "| `%s` | %.2f | %d | %s |\n", e.FuncName, e.EffectiveCRAP, e.Complexity, loc)
	}

	fmt.Fprintln(w)
}

func (f *PRCommentFormatter) writeRegressionsSection(w io.Writer, sorted []score.CRAPEntry, baseDir string) {
	regressed := filterRegressions(sorted)
	if len(regressed) == 0 {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "## \U0001f534 Regressions")
	fmt.Fprintln(w)

	sortByDeltaDesc(regressed)
	fmt.Fprintln(w, "| Function | CRAP | \u0394 | Location |")
	fmt.Fprintln(w, "|---|---:|---:|---|")

	for _, e := range regressed {
		loc := formatPRLocation(e, baseDir)
		delta := e.EffectiveCRAP - e.BaselineCRAP
		fmt.Fprintf(w, "| `%s` | %.2f | +%.1f | %s |\n", e.FuncName, e.EffectiveCRAP, delta, loc)
	}

	fmt.Fprintln(w)
}

func (f *PRCommentFormatter) writeSummaryTable(w io.Writer, summary *Summary, baseline *Baseline, totalFuncs, exceeded int) {
	if baseline == nil || summary == nil {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "## Summary")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "| Metric | Baseline | Current | \u0394 |")
	fmt.Fprintln(w, "|---|---:|---:|---|")

	combinedDeltaStr := formatDeltaStr(summary.DeltaCombined)
	avgDeltaStr := formatDeltaStr(summary.DeltaAverage)
	exceededDelta := exceeded - summary.Exceeded
	var exceededDeltaStr string
	if exceededDelta != 0 {
		if exceededDelta > 0 {
			exceededDeltaStr = fmt.Sprintf("+%d \U0001f534", exceededDelta)
		} else {
			exceededDeltaStr = fmt.Sprintf("%d \U0001f7e2", exceededDelta)
		}
	}

	fmt.Fprintf(w, "| Combined CRAP | %.1f | %.1f | %s |\n",
		summary.BaselineCombined, summary.Combined, combinedDeltaStr)
	fmt.Fprintf(w, "| Average CRAP | %.1f | %.1f | %s |\n",
		summary.BaselineAverage, summary.Average, avgDeltaStr)
	fmt.Fprintf(w, "| Functions exceeding threshold | %d | %d | %s |\n",
		baseline.Summary.TotalFuncs, exceeded, exceededDeltaStr)
	fmt.Fprintf(w, "| Total functions | %d | %d | |\n",
		baseline.Summary.TotalFuncs, totalFuncs)

	fmt.Fprintln(w)
}

func formatDeltaStr(delta float64) string {
	if delta == 0 {
		return "-"
	}
	if delta > 0 {
		return fmt.Sprintf("+%.1f \U0001f534", delta)
	}
	return fmt.Sprintf("%.1f \U0001f7e2", delta)
}

func sortByDeltaDesc(entries []score.CRAPEntry) {
	for i := 0; i < len(entries); i++ {
		for j := i + 1; j < len(entries); j++ {
			di := entries[i].EffectiveCRAP - entries[i].BaselineCRAP
			dj := entries[j].EffectiveCRAP - entries[j].BaselineCRAP
			if dj > di {
				entries[i], entries[j] = entries[j], entries[i]
			}
		}
	}
}

func filterNewFunctions(entries []score.CRAPEntry) []score.CRAPEntry {
	result := make([]score.CRAPEntry, 0)
	for _, e := range entries {
		if e.BaselineCRAP < 0 {
			result = append(result, e)
		}
	}
	return result
}

func filterRegressions(entries []score.CRAPEntry) []score.CRAPEntry {
	result := make([]score.CRAPEntry, 0)
	for _, e := range entries {
		if e.BaselineCRAP >= 0 && e.EffectiveCRAP-e.BaselineCRAP > deltaTolerance {
			result = append(result, e)
		}
	}
	return result
}

// PRCommentFormatter outputs CRAP entries as a GitHub PR comment.
type PRCommentFormatter struct{}

func (f *PRCommentFormatter) Format(entries *scan.Entries, opts FormatOptions) error {
	if entries == nil {
		return fmt.Errorf("Format: entries list shouldn't be nil")
	}

	sorted := entries.ForPRComment()

	crappy := filterAboveThreshold(sorted, opts.Threshold)

	f.writePRHeader(opts.Writer, sorted, crappy, opts.Threshold, opts.Summary, opts.Baseline)

	if len(crappy) > maxPRCommentRows {
		crappy = crappy[:maxPRCommentRows]
	}

	f.writeCrappyTable(opts.Writer, crappy, len(entries.List), opts.BaseDir, opts.Baseline)

	if opts.Baseline != nil {
		f.writeNewFunctionsSection(opts.Writer, sorted, opts.BaseDir)
		f.writeRegressionsSection(opts.Writer, sorted, opts.BaseDir)
		f.writeSummaryTable(opts.Writer, opts.Summary, opts.Baseline, len(entries.List), 0)
	}

	unreliable := filterUnreliableCoverage(sorted)
	f.writeUnreliableSection(opts.Writer, unreliable, opts.Detailed)

	unavailable := filterUnavailableCoverage(sorted)
	f.writeUnavailableSection(opts.Writer, unavailable)

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

func filterUnavailableCoverage(entries []score.CRAPEntry) []score.CRAPEntry {
	result := make([]score.CRAPEntry, 0)
	for _, e := range entries {
		if e.CoverageWarning != "" {
			result = append(result, e)
		}
	}
	return result
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

func (f *PRCommentFormatter) writeUnavailableSection(w io.Writer, unavailable []score.CRAPEntry) {
	if len(unavailable) == 0 {
		return
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "## \u26a0\ufe0f Coverage Unavailable")
	fmt.Fprintln(w)

	fmt.Fprintln(w, "| Function | Location | Reason |")
	fmt.Fprintln(w, "|---|---|---|")
	for _, e := range unavailable {
		loc := formatPRLocation(e, "")
		fmt.Fprintf(w, "| `%s` | %s | %s |\n", e.FuncName, loc, e.CoverageWarning)
	}
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
