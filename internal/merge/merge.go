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
	Receiver   string
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
	if idx := strings.Index(name, ")."); idx != -1 {
		return name[idx+2:]
	}
	if idx := strings.Index(name, "."); idx != -1 {
		recv := name[:idx]
		if strings.HasPrefix(recv, "*") || strings.Contains(recv, ".") {
			return name[idx+1:]
		}
	}
	return name
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
					if fn.Coverage > 0 {
						cov := fn.Coverage
						coverage = &cov
					}
					break
				}
			}
		}

		name := stat.FuncName
	if stat.Receiver != "" {
		name = stat.Receiver + "." + name
	}
	entries = append(entries, MergedEntry{
		File:       stat.Pos.Filename,
		Package:    stat.PkgName,
		FuncName:   name,
		Receiver:   stat.Receiver,
		Line:       stat.Pos.Line,
		Complexity: stat.Complexity,
		Coverage:   coverage,
	})
	}
	return entries
}
