package complexity

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

type Stat struct {
	Pos        token.Position
	FuncName   string
	PkgName    string
	Complexity int
}

type Stats []Stat

func Analyze(paths []string, ignore *regexp.Regexp) []Stat {
	var stats Stats
	for _, path := range paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}
		stats = analyzeDir(absPath, stats, ignore)
	}
	return stats
}

func analyzeDir(dir string, stats Stats, ignore *regexp.Regexp) Stats {
	entries, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return stats
	}
	for _, entry := range entries {
		stats = analyzeFile(entry, stats, ignore)
	}
	dirs, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return stats
	}
	for _, dirEntry := range dirs {
		info, err := os.Stat(dirEntry)
		if err != nil || !info.IsDir() {
			continue
		}
		base := info.Name()
		if strings.HasPrefix(base, ".") || strings.HasPrefix(base, "_") || base == "vendor" || base == "testdata" {
			continue
		}
		stats = analyzeDir(dirEntry, stats, ignore)
	}
	return stats
}

func analyzeFile(file string, stats Stats, ignore *regexp.Regexp) Stats {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return stats
	}
	stats = AnalyzeASTFile(f, fset, stats, ignore)
	return stats
}

func AnalyzeASTFile(f *ast.File, fset *token.FileSet, stats Stats, ignore *regexp.Regexp) Stats {
	for _, decl := range f.Decls {
		fnDecl, ok := decl.(*ast.FuncDecl)
		if !ok || fnDecl.Recv != nil {
			continue
		}
		directives := parseDirectives(fnDecl.Doc)
		if directives.ignore {
			continue
		}
		name := fnDecl.Name.Name
		if ignore != nil && ignore.MatchString(name) {
			continue
		}
		complexity := Complexity(fnDecl)
		stats = append(stats, Stat{
			PkgName:    f.Name.Name,
			FuncName:   name,
			Complexity: complexity,
			Pos:        fset.Position(fnDecl.Pos()),
		})
	}
	return stats
}
