package report

import (
	"fmt"

	"github.com/padiazg/go-crap/internal/score"
)

type GithubFormatter struct{}

func (f *GithubFormatter) Format(entries *score.EntryList, opts FormatOptions) error {
	if entries == nil {
		return fmt.Errorf("Format: entries list shouldn't be nil")
	}

	for _, e := range entries.List {
		effectiveCRAP := e.EffectiveScore()

		file := e.File
		if base := opts.BaseDir; base != "" {
			if rel := RelativizePath(e.File, base); rel != e.File {
				file = rel
			}
		}

		if e.CoverageUntrusted {
			msg := fmt.Sprintf("%s:%d %s [coverage not reliable (mutation score: %.1f%%)]",
				file, e.Line, e.FuncName, e.MutationScore*100)
			fmt.Fprintf(opts.Writer, "::warning file=%s,line=%d::%s\n", file, e.Line, msg)
		}

		if effectiveCRAP > opts.Threshold {
			msg := fmt.Sprintf("%s:%d %s CRAP score %.1f (CC=%d, cov=%.1f%%) exceeds threshold %.0f",
				file, e.Line, e.FuncName, effectiveCRAP, e.Complexity, e.Coverage, opts.Threshold)
			if e.CoverageUntrusted {
				msg += fmt.Sprintf(" [coverage not reliable (mutation score: %.1f%%)]", e.MutationScore*100)
			}
			fmt.Fprintf(opts.Writer, "::warning file=%s,line=%d::%s\n", file, e.Line, msg)
		}
	}
	return nil
}
