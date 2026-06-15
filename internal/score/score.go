package score

import (
	"github.com/padiazg/go-crap/internal/merge"
)

type MissingPolicy int

const (
	MissingPessimistic MissingPolicy = iota
	MissingOptimistic
	MissingSkip
)

type MutationDetail struct {
	File            string
	MutantType      string
	MutatorName     string
	OriginalText    string
	ReplacementText string
	Status          string
	Line            int
}

type CRAPEntry struct {
	File              string
	FuncName          string
	Package           string
	Receiver          string
	MutationDetails   []MutationDetail
	Complexity        int
	Coverage          float64
	CRAP              float64
	EffectiveCRAP     float64
	Line              int
	MutationScore     float64
	CoverageUntrusted bool
	Skipped           bool
}

type EntryList struct {
	List []CRAPEntry
}

func (el *EntryList) ThresholdExceeded(threshold float64) bool {
	for _, e := range el.List {
		if e.EffectiveCRAP > threshold {
			return true
		}
	}

	return false
}

func CRAP(complexity int, coverage float64) float64 {
	comp := float64(complexity)
	cov := coverage / 100.0
	return comp*comp*(1-cov)*(1-cov)*(1-cov) + comp
}

func Score(entries []merge.MergedEntry, policy MissingPolicy) []CRAPEntry {
	result := make([]CRAPEntry, 0, len(entries))
	for _, e := range entries {
		var cov float64
		if e.Coverage == nil {
			switch policy {
			case MissingPessimistic:
				cov = 0.0
			case MissingOptimistic:
				cov = 100.0
			case MissingSkip:
				result = append(result, CRAPEntry{
					File:          e.File,
					Package:       e.Package,
					FuncName:      e.FuncName,
					Receiver:      e.Receiver,
					Line:          e.Line,
					Complexity:    e.Complexity,
					Coverage:      0,
					CRAP:          float64(e.Complexity),
					Skipped:       true,
					EffectiveCRAP: float64(e.Complexity),
				})
				continue
			}
		} else {
			cov = *e.Coverage
		}
		result = append(result, CRAPEntry{
			File:          e.File,
			Package:       e.Package,
			FuncName:      e.FuncName,
			Receiver:      e.Receiver,
			Line:          e.Line,
			Complexity:    e.Complexity,
			Coverage:      cov,
			CRAP:          CRAP(e.Complexity, cov),
			EffectiveCRAP: CRAP(e.Complexity, cov),
		})
	}
	return result
}
