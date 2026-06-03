package report

import (
	"fmt"
	"path/filepath"

	"github.com/padiazg/go-crap/internal/score"
)

type GithubFormatter struct{}

func (f *GithubFormatter) Format(entries []score.CRAPEntry, opts FormatOptions) error {
	for _, e := range entries {
		file := e.File
		if base := opts.BaseDir; base != "" {
			if absBase, err := filepath.Abs(base); err == nil {
				if rel, err := filepath.Rel(absBase, e.File); err == nil && rel != e.File {
					file = rel
				}
			}
		}
		if e.CRAP > opts.Threshold {
			fmt.Fprintf(opts.Writer,
				"::warning file=%s,line=%d::CRAP score %.1f (CC=%d, cov=%.1f%%) exceeds threshold %.0f\n",
				file, e.Line, e.CRAP, e.Complexity, e.Coverage, opts.Threshold,
			)
		}
	}
	return nil
}
