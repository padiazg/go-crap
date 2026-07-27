package report

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/padiazg/go-crap/internal/scan"
	"github.com/padiazg/go-crap/internal/score"
)

// Baseline represents a previous go-crap JSON report used for comparison.
type Baseline struct {
	entries map[string]BaselineEntry
	Summary Summary
}

// BaselineEntry holds CRAP data for a single function from a baseline report.
type BaselineEntry struct {
	CRAP          float64
	EffectiveCRAP float64
	Complexity    int
	Coverage      float64
	FuncName      string
	File          string
	Line          int
}

// LoadBaseline loads a baseline from a JSON report file.
func LoadBaseline(path string) (*Baseline, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("baseline: %w", err)
	}

	var report struct {
		Schema  string       `json:"$schema"`
		Version string       `json:"version"`
		Entries []JSONEntry  `json:"entries"`
	}

	if err := json.Unmarshal(data, &report); err != nil {
		return nil, fmt.Errorf("baseline: invalid JSON: %w", err)
	}

	b := &Baseline{
		entries: make(map[string]BaselineEntry),
	}

	for _, e := range report.Entries {
		entry := BaselineEntry{
			CRAP:          e.CRAP,
			EffectiveCRAP: e.EffectiveCRAP,
			Complexity:    e.Cyclomatic,
			Coverage:      coverageOrDefault(e.Coverage),
			FuncName:      e.Function,
			File:          e.File,
			Line:          e.Line,
		}
		key := entry.File + ":" + entry.FuncName
		b.entries[key] = entry
	}

	b.Summary = b.computeSummary()

	return b, nil
}

func coverageOrDefault(f *float64) float64 {
	if f == nil {
		return 0
	}
	return *f
}

func (b *Baseline) computeSummary() Summary {
	var combined float64
	for _, e := range b.entries {
		combined += e.EffectiveCRAP
	}
	total := len(b.entries)
	var avg float64
	if total > 0 {
		avg = combined / float64(total)
	}
	return Summary{
		Combined:   combined,
		Average:    avg,
		TotalFuncs: total,
	}
}

// Lookup returns the baseline entry for a given file:funcname key.
func (b *Baseline) Lookup(file, funcname string) (BaselineEntry, bool) {
	key := file + ":" + funcname
	entry, ok := b.entries[key]
	return entry, ok
}

// BaselineEntries returns all baseline entries for iteration.
func (b *Baseline) BaselineEntries() []BaselineEntry {
	result := make([]BaselineEntry, 0, len(b.entries))
	for _, e := range b.entries {
		result = append(result, e)
	}
	return result
}

// AnnotateWithBaseline sets BaselineCRAP on each entry based on the baseline.
func AnnotateWithBaseline(entries []score.CRAPEntry, baseline *Baseline) []score.CRAPEntry {
	if baseline == nil {
		return entries
	}
	result := make([]score.CRAPEntry, len(entries))
	for i, e := range entries {
		result[i] = e
		if bl, ok := baseline.Lookup(e.File, e.FuncName); ok {
			result[i].BaselineCRAP = bl.EffectiveCRAP
		} else {
			result[i].BaselineCRAP = -1
		}
	}
	return result
}

// ComputeSummaryWithBaseline computes aggregate CRAP stats and compares against baseline.
func ComputeSummaryWithBaseline(entries *scan.Entries, threshold float64, baseline *Baseline) Summary {
	s := ComputeSummary(entries, threshold)

	if baseline != nil {
		s.BaselineCombined = baseline.Summary.Combined
		s.BaselineAverage = baseline.Summary.Average
		s.DeltaCombined = s.Combined - baseline.Summary.Combined
		s.DeltaAverage = s.Average - baseline.Summary.Average
	}

	return s
}
