package report

import (
	"io"

	"github.com/padiazg/go-crap/internal/score"
)

type Formatter interface {
	Format(entries *score.EntryList, opts FormatOptions) error
}

type FormatOptions struct {
	Writer    io.Writer
	BaseDir   string
	Threshold float64
}
