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
	Coverage        *float64 `json:"coverage"`
	CoverageUntrusted bool   `json:"coverage_untrusted"`
	File            string   `json:"file"`
	Function        string   `json:"function"`
	Receiver        string   `json:"receiver,omitempty"`
	Package         string   `json:"package"`
	CRAP            float64  `json:"crap"`
	EffectiveCRAP   float64  `json:"effective_crap"`
	MutationScore   float64  `json:"mutation_score"`
	Cyclomatic      int      `json:"cyclomatic"`
	Line            int      `json:"line"`
	MutationDetails []JSONMutationDetail `json:"mutation_details,omitempty"`
}

type JSONMutationDetail struct {
	Type            string `json:"type"`
	MutatorName     string `json:"mutator_name,omitempty"`
	File            string `json:"file"`
	Line            int    `json:"line"`
	Status          string `json:"status"`
	OriginalText    string `json:"original_text,omitempty"`
	ReplacementText string `json:"replacement_text,omitempty"`
}

type JSONFormatter struct {
	jsonMarshalIndent func(v any, prefix, indent string) ([]byte, error)
}

func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{
		jsonMarshalIndent: json.MarshalIndent,
	}
}

func (f *JSONFormatter) Format(entries *score.EntryList, opts FormatOptions) error {
	if entries == nil {
		return fmt.Errorf("Format: entries list shouldn't be nil")
	}

	report := Report{
		Schema:  "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		Version: "1.0.0",
		Entries: make([]JSONEntry, 0, len(entries.List)),
	}

	for _, e := range entries.List {
		file := e.File
		if base := opts.BaseDir; base != "" {
			if absBase, err := filepath.Abs(base); err == nil {
				if rel, err := filepath.Rel(absBase, e.File); err == nil && rel != e.File {
					file = rel
				}
			}
		}
		entry := JSONEntry{
			File:            file,
			Package:         e.Package,
			Function:        e.FuncName,
			Receiver:        e.Receiver,
			Line:            e.Line,
			Cyclomatic:      e.Complexity,
			CRAP:            e.CRAP,
			EffectiveCRAP:   e.EffectiveCRAP,
			MutationScore:   e.MutationScore,
			CoverageUntrusted: e.CoverageUntrusted,
		}
		if opts.Detailed && len(e.MutationDetails) > 0 {
			entry.MutationDetails = make([]JSONMutationDetail, 0, len(e.MutationDetails))
			for _, md := range e.MutationDetails {
				entry.MutationDetails = append(entry.MutationDetails, JSONMutationDetail{
					Type:            md.MutantType,
					MutatorName:     md.MutatorName,
					File:            md.File,
					Line:            md.Line,
					Status:          md.Status,
					OriginalText:    md.OriginalText,
					ReplacementText: md.ReplacementText,
				})
			}
		}
		entry.Coverage = &e.Coverage
		report.Entries = append(report.Entries, entry)
	}

	data, err := f.jsonMarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	fmt.Fprintln(opts.Writer, string(data))
	return nil
}
