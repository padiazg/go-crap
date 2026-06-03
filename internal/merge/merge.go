package merge

import (
	"path/filepath"
	"strings"

	"github.com/padiazg/go-crap/internal/complexity"
	"github.com/padiazg/go-crap/internal/coverage"
)

type MergedEntry struct {
	Coverage   *float64
	File       string
	FuncName   string
	Package    string
	Complexity int
	Line       int
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
	parts := strings.Split(path, string(filepath.Separator))
	if len(parts) >= 2 {
		return strings.Join(parts[len(parts)-2:], string(filepath.Separator))
	}
	return filepath.Base(path)
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
	replacer := strings.NewReplacer("(", "", ")", "")
	return replacer.Replace(name)
}

func Merge(coverages []coverage.ModuleCoverage, stats []complexity.Stat) []MergedEntry {
	idx := buildIndex(coverages)
	var entries []MergedEntry
	for _, stat := range stats {
		fnName := normalizeFuncName(stat.FuncName)
		var coverage *float64
		if fns, ok := idx.lookup(stat.Pos.Filename); ok {
			for _, fn := range fns {
				if normalizeFuncName(fn.Name) == fnName {
					cov := fn.Coverage
					coverage = &cov
					break
				}
			}
		}
		entries = append(entries, MergedEntry{
			File:       stat.Pos.Filename,
			Package:    stat.PkgName,
			FuncName:   stat.FuncName,
			Line:       stat.Pos.Line,
			Complexity: stat.Complexity,
			Coverage:   coverage,
		})
	}
	return entries
}
