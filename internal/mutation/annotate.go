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

	endLineIdx := buildEndLineIndex(merged)
	mutantsByFile := buildMutantsByFile(report.Mutants)

	for i := range entries {
		e := &entries[i]

		if e.Skipped || e.Coverage == 0 {
			e.EffectiveCRAP = e.CRAP
			continue
		}

		key := mergeKey(e.File, e.FuncName, e.Receiver)
		endLine := resolveEndLine(endLineIdx, key, e.Line)

		mutants := mutantsByFile[buildMutantFileSuffix(e.File)]
		if len(mutants) == 0 {
			e.EffectiveCRAP = e.CRAP
			continue
		}

		killed, lived, livedMutants := classifyMutants(mutants, e.Line, endLine)
		annotateEntry(e, killed, lived, livedMutants)
	}

	return entries
}

func buildEndLineIndex(merged []merge.MergedEntry) map[string]int {
	endLineIdx := make(map[string]int, len(merged))
	for _, m := range merged {
		key := mergeKey(m.File, m.FuncName, m.Receiver)
		endLineIdx[key] = m.EndLine
	}
	return endLineIdx
}

func resolveEndLine(endLineIdx map[string]int, key string, startLine int) int {
	endLine := endLineIdx[key]
	if endLine < startLine {
		endLine = startLine + 100
	}
	return endLine
}

func classifyMutants(mutants []Mutant, startLine, endLine int) (killed, lived int, livedMutants []Mutant) {
	for _, m := range mutants {
		if m.Line >= startLine && m.Line <= endLine {
			switch m.Status {
			case StatusKilled:
				killed++
			case StatusLived:
				lived++
				livedMutants = append(livedMutants, m)
			}
		}
	}
	return
}

func annotateEntry(e *score.CRAPEntry, killed, lived int, livedMutants []Mutant) {
	if killed == 0 && lived == 0 {
		e.MutationScore = -1
		e.EffectiveCRAP = e.CRAP
		return
	}

	if lived > 0 {
		e.CoverageUntrusted = true
		e.MutationScore = float64(killed) / float64(killed+lived)
		e.EffectiveCRAP = score.CRAP(e.Complexity, 0)
		e.MutationDetails = buildMutationDetails(livedMutants)

		return
	}

	e.CoverageUntrusted = false
	e.MutationScore = 1
	e.EffectiveCRAP = e.CRAP
}

func buildMutationDetails(livedMutants []Mutant) []score.MutationDetail {
	details := make([]score.MutationDetail, 0, len(livedMutants))
	for _, m := range livedMutants {
		details = append(details, score.MutationDetail{
			MutantType:      m.Type,
			MutatorName:     m.MutatorName,
			File:            m.File,
			Line:            m.Line,
			Status:          string(m.Status),
			OriginalText:    m.OriginalCode,
			ReplacementText: m.ReplacementCode,
		})
	}

	return details
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

	switch {
	case len(filtered) >= 3:
		return strings.Join(filtered[len(filtered)-3:], "/")
	case len(filtered) == 1:
		return filtered[0]
	default:
		return strings.Join(filtered, "/")
	}
}
