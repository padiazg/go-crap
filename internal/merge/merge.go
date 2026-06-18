package merge

import (
	"fmt"
	"strings"

	"github.com/padiazg/go-crap/internal/complexity"
	"github.com/padiazg/go-crap/internal/coverage"
)

// MergedEntry combines complexity analysis with coverage data for a single function.
type MergedEntry struct {
	Coverage          *float64
	CoverageWarning   string
	File              string
	FuncName          string
	Package           string
	Receiver          string
	Complexity        int
	EndLine           int
	Line              int
}

type pathIndex struct {
	byAbsolute map[string][]coverage.FunctionCoverage
	bySuffix   map[string][]coverage.FunctionCoverage
}

func buildIndex(coverages []coverage.ModuleCoverage) *pathIndex {
	idx := &pathIndex{
		byAbsolute: make(map[string][]coverage.FunctionCoverage),
		bySuffix:   make(map[string][]coverage.FunctionCoverage),
	}
	for _, mc := range coverages {
		for _, fn := range mc.Functions {
			absPath := fn.File
			idx.byAbsolute[absPath] = append(idx.byAbsolute[absPath], fn)
			suffix := buildSuffix(absPath)
			idx.bySuffix[suffix] = append(idx.bySuffix[suffix], fn)
		}
	}
	return idx
}

func buildSuffix(path string) string {
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

func (idx *pathIndex) lookup(absPath string) ([]coverage.FunctionCoverage, bool) {
	if fns, ok := idx.byAbsolute[absPath]; ok {
		return fns, true
	}
	suffix := buildSuffix(absPath)
	if fns, ok := idx.bySuffix[suffix]; ok {
		return fns, true
	}
	return nil, false
}

func normalizeFuncName(name string) string {
	// (*Type).Method → Method
	if _, after, ok := strings.Cut(name, ")."); ok {
		return after
	}
	// Type.Method or *Type.Method → Method
	if _, after, ok := strings.Cut(name, "."); ok {
		return after
	}
	return name
}

// Merge combines complexity statistics and coverage data into unified entries.
func Merge(coverages []coverage.ModuleCoverage, stats []complexity.Stat) []MergedEntry {
	idx := buildIndex(coverages)
	erroredModules := make(map[string]string, len(coverages))
	for _, mc := range coverages {
		if mc.Error != nil {
			erroredModules[mc.Dir] = mc.Error.Error()
		}
	}

	var entries []MergedEntry
	for _, stat := range stats {
		fnName := normalizeFuncName(stat.FuncName)
		var coverage *float64
		var covWarn string
		if fns, ok := idx.lookup(stat.Pos.Filename); ok {
			for _, fn := range fns {
				if normalizeFuncName(fn.Name) == fnName {
					cov := fn.Coverage
					coverage = &cov
					break
				}
			}
		}
		if coverage == nil {
			for modDir, errMsg := range erroredModules {
				if strings.HasPrefix(stat.Pos.Filename, modDir) {
					covWarn = fmt.Sprintf("coverage unavailable for %s: %s", modDir, errMsg)
					break
				}
			}
		}

		name := stat.FuncName
		if stat.Receiver != "" {
			name = stat.Receiver + "." + name
		}
		entries = append(entries, MergedEntry{
			CoverageWarning: covWarn,
			File:            stat.Pos.Filename,
			EndLine:         stat.EndLine,
			Package:         stat.PkgName,
			FuncName:        name,
			Receiver:        stat.Receiver,
			Line:            stat.Pos.Line,
			Complexity:      stat.Complexity,
			Coverage:        coverage,
		})
	}
	return entries
}
