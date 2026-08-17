package report

import (
	"encoding/json"
	"fmt"

	"github.com/padiazg/go-crap/internal/scan"
	"github.com/padiazg/go-crap/internal/score"
)

type Report struct {
	Schema  string         `json:"$schema"`
	Version string         `json:"version"`
	Summary *SummaryObject `json:"summary,omitempty"`
	Entries []JSONEntry    `json:"entries"`
}

type SummaryObject struct {
	BaselineAverage  *float64 `json:"baseline_average,omitempty"`
	BaselineCombined *float64 `json:"baseline_combined,omitempty"`
	DeltaAverage     *float64 `json:"delta_average,omitempty"`
	DeltaCombined    *float64 `json:"delta_combined,omitempty"`
	Average          float64  `json:"average"`
	Combined         float64  `json:"combined"`
	Exceeded         int      `json:"exceeded"`
	TotalFuncs       int      `json:"total_funcs"`
}

type JSONEntry struct {
	BaselineCRAP      *float64             `json:"baseline_crap,omitempty"`
	Coverage          *float64             `json:"coverage,omitempty"`
	Delta             *float64             `json:"delta,omitempty"`
	CoverageWarning   string               `json:"coverage_warning,omitempty"`
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
		Schema:  "https://raw.githubusercontent.com/padiazg/go-crap/master/schemas/report-v1.json",
		Version: "1.1.0",
		Entries: make([]JSONEntry, 0, len(entries.List)),
	}

	if opts.Summary != nil {
		report.Summary = &SummaryObject{
			Combined:   opts.Summary.Combined,
			Average:    opts.Summary.Average,
			TotalFuncs: opts.Summary.TotalFuncs,
			Exceeded:   opts.Summary.Exceeded,
		}
		if opts.Baseline != nil {
			report.Summary.BaselineCombined = &opts.Summary.BaselineCombined
			report.Summary.BaselineAverage = &opts.Summary.BaselineAverage
			report.Summary.DeltaCombined = &opts.Summary.DeltaCombined
			report.Summary.DeltaAverage = &opts.Summary.DeltaAverage
		}
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

	if e.BaselineCRAP >= 0 {
		baselineCRAP := e.BaselineCRAP
		entry.BaselineCRAP = &baselineCRAP
		delta := e.EffectiveCRAP - e.BaselineCRAP
		entry.Delta = &delta
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
