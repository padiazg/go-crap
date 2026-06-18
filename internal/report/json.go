package report

import (
	"encoding/json"
	"fmt"

	"github.com/padiazg/go-crap/internal/scan"
	"github.com/padiazg/go-crap/internal/score"
)

type Report struct {
	Schema  string      `json:"$schema"`
	Version string      `json:"version"`
	Entries []JSONEntry `json:"entries"`
}

type JSONEntry struct {
	Coverage          *float64             `json:"coverage,omitempty"`
	File              string               `json:"file"`
	Function          string               `json:"function"`
	Package           string               `json:"package"`
	Receiver          string               `json:"receiver,omitempty"`
	MutationDetails   []JSONMutationDetail `json:"mutation_details,omitempty"`
	CRAP              float64              `json:"crap"`
	Cyclomatic        int                  `json:"cyclomatic"`
	EffectiveCRAP     float64              `json:"effective_crap"`
	Line              int                  `json:"line"`
	MutationScore     float64              `json:"mutation_score"`
	CoverageUntrusted bool                 `json:"coverage_untrusted"`
	CoverageWarning   string               `json:"coverage_warning,omitempty"`
}

type JSONMutationDetail struct {
	File            string `json:"file"`
	MutatorName     string `json:"mutator_name,omitempty"`
	OriginalText    string `json:"original_text,omitempty"`
	ReplacementText string `json:"replacement_text,omitempty"`
	Status          string `json:"status"`
	Type            string `json:"type"`
	Line            int    `json:"line"`
}

// JSONFormatter outputs CRAP entries as JSON.
type JSONFormatter struct {
	jsonMarshalIndent func(v any, prefix, indent string) ([]byte, error)
}

func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{
		jsonMarshalIndent: json.MarshalIndent,
	}
}

func (f *JSONFormatter) Format(entries *scan.Entries, opts FormatOptions) error {
	if entries == nil {
		return fmt.Errorf("Format: entries list shouldn't be nil")
	}

	report := Report{
		Schema:  "https://raw.githubusercontent.com/padiazg/go-crap/main/schemas/report-v1.json",
		Version: "1.0.0",
		Entries: make([]JSONEntry, 0, len(entries.List)),
	}

	for _, e := range entries.List {
		report.Entries = append(report.Entries, f.convertToJSONEntry(e, opts))
	}

	data, err := f.jsonMarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal JSON: %w", err)
	}

	fmt.Fprintln(opts.Writer, string(data))
	return nil
}

func (f *JSONFormatter) convertToJSONEntry(e score.CRAPEntry, opts FormatOptions) JSONEntry {
	file := e.File
	if base := opts.BaseDir; base != "" {
		if rel := RelativizePath(e.File, base); rel != e.File {
			file = rel
		}
	}

	entry := JSONEntry{
		File:              file,
		Package:           e.Package,
		Function:          e.FuncName,
		Receiver:          e.Receiver,
		Line:              e.Line,
		Cyclomatic:        e.Complexity,
		CRAP:              e.CRAP,
		EffectiveCRAP:     e.EffectiveCRAP,
		MutationScore:     e.MutationScore,
		CoverageUntrusted: e.CoverageUntrusted,
		CoverageWarning:   e.CoverageWarning,
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

	if e.CoverageWarning == "" {
		entry.Coverage = &e.Coverage
	}
	return entry
}
