package scan

import (
	"fmt"
	"sort"

	"github.com/padiazg/go-crap/internal/merge"
	"github.com/padiazg/go-crap/internal/mutation"
	"github.com/padiazg/go-crap/internal/score"
)

// Entries is a list of CRAP entries.
type Entries struct {
	options *Options
	List    []score.CRAPEntry
}

func NewEntries(options *Options, merged []merge.MergedEntry, policy score.MissingPolicy) (*Entries, error) {
	entries := &Entries{
		List:    score.Score(merged, policy),
		options: options,
	}

	if err := entries.applyMutationAnnotations(merged); err != nil {
		return nil, fmt.Errorf("NewEntries: %w", err)
	}

	entries.applyFilters()

	return entries, nil
}

func (entries *Entries) ThresholdExceeded(threshold float64) bool {
	for _, e := range entries.List {
		if e.EffectiveCRAP > threshold {
			return true
		}
	}

	return false
}

func (entries *Entries) applyMutationAnnotations(merged []merge.MergedEntry) error {
	if entries.options.MutationReport == "" {
		return nil
	}

	mutReport, err := mutation.ParseReport(entries.options.MutationReport)
	if err != nil {
		return fmt.Errorf("ApplyMutationAnnotations: %w", err)
	}

	entries.List = mutation.Annotate(entries.List, mutReport, merged)

	return nil
}

func (entries *Entries) sort() {
	sort.Slice(entries.List, func(i, j int) bool {
		return effectiveCRAP(entries.List[i]) > effectiveCRAP(entries.List[j])
	})
}

func (entries *Entries) applyFilters() {
	entries.sort()

	if entries.shouldApplyMinFilter() {
		entries.filterByMinCRAP()
	}

	if entries.shouldApplyTopFilter() {
		entries.filterByTop()
	}
}

func (entries *Entries) shouldApplyMinFilter() bool {
	return entries.options.Min > 0
}

func (entries *Entries) shouldApplyTopFilter() bool {
	return entries.options.Top > 0 && entries.options.Top < len(entries.List)
}

func (entries *Entries) filterByMinCRAP() {
	var filtered []score.CRAPEntry
	for _, e := range entries.List {
		if e.CoverageUntrusted || effectiveCRAP(e) >= entries.options.Min {
			filtered = append(filtered, e)
		}
	}

	entries.List = filtered
}

func (entries *Entries) filterByTop() {
	var result []score.CRAPEntry

	for _, e := range entries.List {
		if e.CoverageUntrusted {
			result = append(result, e)
		}
	}

	var kept int
	for _, e := range entries.List {
		if !e.CoverageUntrusted && kept < entries.options.Top {
			result = append(result, e)
			kept++
		}
	}

	entries.List = result
}

func (entries *Entries) ForPRComment() []score.CRAPEntry {
	sorted := make([]score.CRAPEntry, len(entries.List))
	copy(sorted, entries.List)
	for i := range sorted {
		sorted[i].EffectiveCRAP = sorted[i].EffectiveScore()
	}

	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].EffectiveCRAP > sorted[j].EffectiveCRAP
	})

	return sorted
}

func (entries *Entries) ForTable() []score.CRAPEntry {
	sorted := make([]score.CRAPEntry, len(entries.List))
	copy(sorted, entries.List)

	for i := range sorted {
		sorted[i].EffectiveCRAP = sorted[i].EffectiveScore()
	}

	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].EffectiveCRAP != sorted[j].EffectiveCRAP {
			return sorted[i].EffectiveCRAP > sorted[j].EffectiveCRAP
		}

		return sorted[i].MutationScore < sorted[j].MutationScore
	})

	return sorted
}
