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
