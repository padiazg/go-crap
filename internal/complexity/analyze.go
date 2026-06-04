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
	Receiver   string
	PkgName    string
	Complexity int
}

type analyzeData struct {
	exclude *regexp.Regexp
	paths   []string
	Stats   []Stat
}

func newAnalyze(paths []string, exclude *regexp.Regexp) *analyzeData {
	return &analyzeData{
		exclude: exclude,
		paths:   paths,
		Stats:   make([]Stat, 0),
	}
}

func (a *analyzeData) Analyze() {
	for _, path := range a.paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			continue
		}

		a.analyzeDir(absPath)
	}
}

func Analyze(paths []string, exclude *regexp.Regexp) []Stat {
	a := newAnalyze(paths, exclude)
	a.Analyze()
	return a.Stats
}

func (a *analyzeData) analyzeDir(dir string) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		return
	}

	for _, entry := range entries {
		if a.exclude != nil && a.exclude.MatchString(entry) {
			continue
		}

		a.analyzeFile(entry)
	}

	dirs, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		return
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

		a.analyzeDir(dirEntry)
	}
}

func (a *analyzeData) analyzeFile(file string) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ParseComments|parser.AllErrors)
	if err != nil {
		return
	}

	a.analyzeASTFile(f, fset)
}

func receiverName(recv *ast.FieldList) string {
	if recv == nil || len(recv.List) == 0 {
		return ""
	}
	field := recv.List[0]
	if field == nil || field.Type == nil {
		return ""
	}

	var recvType string
	switch t := field.Type.(type) {
	case *ast.StarExpr:
		recvType = "*" + exprString(t.X)
	case *ast.Ident:
		recvType = t.Name
	case *ast.SelectorExpr:
		recvType = exprString(t.X) + "." + t.Sel.Name
	default:
		recvType = exprString(t)
	}

	return recvType
}

func exprString(e ast.Expr) string {
	switch v := e.(type) {
	case *ast.Ident:
		return v.Name
	case *ast.SelectorExpr:
		return exprString(v.X) + "." + v.Sel.Name
	case *ast.IndexExpr:
		return exprString(v.X) + "[" + exprString(v.Index) + "]"
	case *ast.SliceExpr:
		return exprString(v.X) + "[:]"
	case *ast.StarExpr:
		return "*" + exprString(v.X)
	default:
		return "<unknown>"
	}
}

func (a *analyzeData) analyzeASTFile(f *ast.File, fset *token.FileSet) {
	for _, decl := range f.Decls {
		fnDecl, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}

		directives := parseDirectives(fnDecl.Doc)
		if directives.exclude {
			continue
		}

		name := fnDecl.Name.Name
		if a.exclude != nil && a.exclude.MatchString(name) {
			continue
		}

		complexity := Complexity(fnDecl)
		a.Stats = append(a.Stats, Stat{
			PkgName:    f.Name.Name,
			FuncName:   name,
			Receiver:   receiverName(fnDecl.Recv),
			Complexity: complexity,
			Pos:        fset.Position(fnDecl.Pos()),
		})
	}
}
