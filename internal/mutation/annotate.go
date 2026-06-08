package mutation

import (
	"strings"

	"github.com/padiazg/go-crap/internal/merge"
	"github.com/padiazg/go-crap/internal/score"
)

func Annotate(entries []score.CRAPEntry, report *Report, merged []merge.MergedEntry) []score.CRAPEntry {
	if report == nil {
		for i := range entries {
			entries[i].EffectiveCRAP = entries[i].CRAP
		}
		return entries
	}

	// Build endLine index from merged entries
	endLineIdx := make(map[string]int, len(merged))
	for _, m := range merged {
		key := mergeKey(m.File, m.FuncName, m.Receiver)
		endLineIdx[key] = m.EndLine
	}

	// Build mutants index by file suffix
	mutantsByFile := buildMutantsByFile(report.Mutants)

	for i := range entries {
		e := &entries[i]

		if e.Skipped || e.Coverage == 0 {
			e.EffectiveCRAP = e.CRAP
			continue
		}

		key := mergeKey(e.File, e.FuncName, e.Receiver)
		endLine := endLineIdx[key]
		if endLine < e.Line {
			endLine = e.Line + 100
		}

		mutants := mutantsByFile[buildMutantFileSuffix(e.File)]
		if len(mutants) == 0 {
			e.EffectiveCRAP = e.CRAP
			continue
		}

		var killed, lived int
		var livedMutants []Mutant
		for _, m := range mutants {
			if m.Line >= e.Line && m.Line <= endLine {
				switch m.Status {
				case StatusKilled:
					killed++
				case StatusLived:
					lived++
					livedMutants = append(livedMutants, m)
				}
			}
		}

		if lived > 0 {
			e.CoverageUntrusted = true
			e.MutationScore = float64(killed) / float64(killed+lived)
			e.EffectiveCRAP = score.CRAP(e.Complexity, 0)
			if len(livedMutants) > 0 {
				e.MutationDetails = make([]score.MutationDetail, 0, len(livedMutants))
				for _, m := range livedMutants {
					e.MutationDetails = append(e.MutationDetails, score.MutationDetail{
						MutantType:    m.Type,
						MutatorName:   m.MutatorName,
						File:          m.File,
						Line:          m.Line,
						Status:        string(m.Status),
						OriginalText:  m.OriginalCode,
						ReplacementText: m.ReplacementCode,
					})
				}
			}
		} else {
			e.CoverageUntrusted = false
			total := killed + lived
			if total > 0 {
				e.MutationScore = float64(killed) / float64(total)
			} else {
				e.MutationScore = -1
			}
			e.EffectiveCRAP = e.CRAP
		}
	}

	return entries
}

func mergeKey(file, funcName, receiver string) string {
	if receiver != "" {
		return file + "::" + receiver + "." + funcName
	}
	return file + "::" + funcName
}

func buildMutantsByFile(mutants []Mutant) map[string][]Mutant {
	result := make(map[string][]Mutant)
	for _, m := range mutants {
		result[buildMutantFileSuffix(m.File)] = append(result[buildMutantFileSuffix(m.File)], m)
	}
	return result
}

func buildMutantFileSuffix(path string) string {
	var parts []string
	if strings.ContainsRune(path, '\\') {
		parts = strings.Split(path, "\\")
	} else {
		parts = strings.Split(path, "/")
	}
	filtered := make([]string, 0, len(parts))
	for _, p := range parts {
		if p != "" {
			filtered = append(filtered, p)
		}
	}
	if len(filtered) >= 3 {
		return strings.Join(filtered[len(filtered)-3:], "/")
	}
	if len(filtered) == 1 {
		return filtered[0]
	}
	return strings.Join(filtered, "/")
}
