package report

import (
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/padiazg/go-crap/internal/score"
)

type Report struct {
	Schema  string      `json:"$schema"`
	Version string      `json:"version"`
	Entries []JSONEntry `json:"entries"`
}

type JSONEntry struct {
	Coverage   *float64 `json:"coverage"`
	File       string   `json:"file"`
	Function   string   `json:"function"`
	Package    string   `json:"package"`
	CRAP       float64  `json:"crap"`
	Cyclomatic int      `json:"cyclomatic"`
	Line       int      `json:"line"`
}

type JSONFormatter struct{}

func (f *JSONFormatter) Format(entries []score.CRAPEntry, opts FormatOptions) error {
	report := Report{
		Schema:  "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		Version: "1.0.0",
		Entries: make([]JSONEntry, 0, len(entries)),
	}

	for _, e := range entries {
		file := e.File
		if base := opts.BaseDir; base != "" {
			if absBase, err := filepath.Abs(base); err == nil {
				if rel, err := filepath.Rel(absBase, e.File); err == nil && rel != e.File {
					file = rel
				}
			}
		}
		entry := JSONEntry{
			File:       file,
			Package:    e.Package,
			Function:   e.FuncName,
			Line:       e.Line,
			Cyclomatic: e.Complexity,
			CRAP:       e.CRAP,
		}
		if e.Coverage > 0 || e.Coverage == 0 {
			entry.Coverage = &e.Coverage
		}
		report.Entries = append(report.Entries, entry)
	}

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	fmt.Fprintln(opts.Writer, string(data))
	return nil
}
