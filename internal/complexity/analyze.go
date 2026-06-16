package complexity

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/padiazg/go-crap/pkg/dummylogger"
	"github.com/padiazg/go-crap/pkg/logger"
)

// Stat holds complexity analysis data for a single function.
type Stat struct {
	Pos        token.Position
	FuncName   string
	PkgName    string
	Receiver   string
	Complexity int
	EndLine    int
}

type analyzeData struct {
	exclude *regexp.Regexp
	logger  logger.Logger
	paths   []string
	stats   []Stat
}

func newAnalyze(paths []string, exclude *regexp.Regexp, l logger.Logger) *analyzeData {
	if l == nil {
		l = dummylogger.New(nil)
	}

	return &analyzeData{
		exclude: exclude,
		paths:   paths,
		stats:   make([]Stat, 0),
		logger:  l,
	}
}

func (a *analyzeData) Analyze() {
	for _, path := range a.paths {
		absPath, err := filepath.Abs(path)
		if err != nil {
			a.logger.Debug("complexity analyze: could not resolve absolute path", "path", path, "error", err.Error())
			continue
		}

		a.analyzeDir(absPath)
	}
}

// Analyze walks directories, parses Go files, and returns cyclomatic complexity statistics for each function.
func Analyze(paths []string, exclude *regexp.Regexp, l logger.Logger) []Stat {
	a := newAnalyze(paths, exclude, l)
	a.Analyze()
	return a.stats
}

func (a *analyzeData) analyzeDir(dir string) {
	a.analyzeGoFiles(dir)
	a.walkSubdirs(dir)
}

func (a *analyzeData) analyzeGoFiles(dir string) {
	entries, err := filepath.Glob(filepath.Join(dir, "*.go"))
	if err != nil {
		a.logger.Debug("complexity analyze: glob error", "dir", dir, "error", err.Error())
		return
	}

	for _, entry := range entries {
		if a.exclude != nil && a.exclude.MatchString(entry) {
			continue
		}

		a.analyzeFile(entry)
	}
}

func (a *analyzeData) walkSubdirs(dir string) {
	dirs, err := filepath.Glob(filepath.Join(dir, "*"))
	if err != nil {
		a.logger.Debug("complexity analyze: glob error for subdirs", "dir", dir, "error", err.Error())
		return
	}

	for _, dirEntry := range dirs {
		info, err := os.Stat(dirEntry)
		if err != nil {
			a.logger.Debug("complexity analyze: stat error", "dir", dirEntry, "error", err.Error())
			continue
		}
		if !info.IsDir() {
			continue
		}

		if shouldSkipDir(info.Name()) {
			continue
		}

		a.analyzeDir(dirEntry)
	}
}

func shouldSkipDir(name string) bool {
	return strings.HasPrefix(name, ".") || strings.HasPrefix(name, "_") || name == "vendor" || name == "testdata"
}

func (a *analyzeData) analyzeFile(file string) {
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, file, nil, parser.ParseComments|parser.AllErrors)
	if err != nil {
		a.logger.Debug("complexity analyze: parse error", "file", file, "error", err.Error())
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
		a.stats = append(a.stats, Stat{
			PkgName:    f.Name.Name,
			FuncName:   name,
			Receiver:   receiverName(fnDecl.Recv),
			Complexity: complexity,
			Pos:        fset.Position(fnDecl.Pos()),
			EndLine:    fset.Position(fnDecl.End()).Line,
		})
	}
}
