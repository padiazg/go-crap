package report

import (
	"fmt"

	"github.com/padiazg/go-crap/internal/scan"
	"github.com/padiazg/go-crap/internal/score"
)

// GithubFormatter outputs CRAP entries as GitHub Actions warnings.
type GithubFormatter struct{}

func (f *GithubFormatter) Format(entries *scan.Entries, opts FormatOptions) error {
	if entries == nil {
		return fmt.Errorf("Format: entries list shouldn't be nil")
	}

	for _, e := range entries.List {
		effectiveCRAP := e.EffectiveScore()
		file := resolveGithubFile(e, opts.BaseDir)

		if e.CoverageWarning != "" {
			fmt.Fprintf(opts.Writer, "::warning file=%s,line=%d::%s\n", file, e.Line, e.CoverageWarning)
		}

		if e.CoverageUntrusted {
			msg := formatGithubUntrustedWarning(file, e)
			fmt.Fprintf(opts.Writer, "::warning file=%s,line=%d::%s\n", file, e.Line, msg)
		}

		if effectiveCRAP > opts.Threshold {
			msg := formatGithubCRAPWarning(file, e, effectiveCRAP, opts.Threshold)
			fmt.Fprintf(opts.Writer, "::warning file=%s,line=%d::%s\n", file, e.Line, msg)
		}
	}

	if opts.Summary != nil && opts.Baseline != nil {
		fmt.Fprintf(opts.Writer, "::notice::CRAP Summary: combined %.1f", opts.Summary.Combined)
		if opts.Summary.DeltaCombined != 0 {
			symbol := "+"
			if opts.Summary.DeltaCombined < 0 {
				symbol = ""
			}
			fmt.Fprintf(opts.Writer, "(%s%.1f vs baseline)", symbol, opts.Summary.DeltaCombined)
		}
		fmt.Fprintf(opts.Writer, ", average %.1f", opts.Summary.Average)
		if opts.Summary.DeltaAverage != 0 {
			symbol := "+"
			if opts.Summary.DeltaAverage < 0 {
				symbol = ""
			}
			fmt.Fprintf(opts.Writer, "(%s%.1f vs baseline)", symbol, opts.Summary.DeltaAverage)
		}
		fmt.Fprintf(opts.Writer, ", %d/%d functions exceed threshold\n", opts.Summary.Exceeded, opts.Summary.TotalFuncs)
	}

	return nil
}

func resolveGithubFile(e score.CRAPEntry, baseDir string) string {
	file := e.File
	if baseDir != "" {
		if rel := RelativizePath(e.File, baseDir); rel != e.File {
			file = rel
		}
	}
	return file
}

func formatGithubUntrustedWarning(file string, e score.CRAPEntry) string {
	return fmt.Sprintf("%s:%d %s [coverage not reliable (mutation score: %.1f%%)]",
		file, e.Line, e.FuncName, e.MutationScore*100)
}

func formatGithubCRAPWarning(file string, e score.CRAPEntry, effectiveCRAP, threshold float64) string {
	msg := fmt.Sprintf("%s:%d %s CRAP score %.1f (CC=%d, cov=%.1f%%) exceeds threshold %.0f",
		file, e.Line, e.FuncName, effectiveCRAP, e.Complexity, e.Coverage, threshold)
	if e.CoverageUntrusted {
		msg += fmt.Sprintf(" [coverage not reliable (mutation score: %.1f%%)]", e.MutationScore*100)
	}
	return msg
}
