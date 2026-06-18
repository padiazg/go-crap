package complexity

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"regexp"
	"testing"

	"github.com/padiazg/go-crap/pkg/logger"
	"github.com/stretchr/testify/assert"
)

func TestComplexity_Simple(t *testing.T) {
	src := `
package test
func simple() int { return 42 }
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		got := Complexity(fn)
		if got != 1 {
			t.Errorf("simple() = %d, want 1", got)
		}
	}
}

func TestComplexity_WithIf(t *testing.T) {
	src := `
package test
func withIf(x int) int {
	if x > 0 { return 1 }
	return 0
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		got := Complexity(fn)
		if got != 2 {
			t.Errorf("withIf() = %d, want 2", got)
		}
	}
}

func TestComplexity_BinaryAnd(t *testing.T) {
	src := `
package test
func withAnd(a, b bool) bool {
	return a && b
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		got := Complexity(fn)
		if got != 2 {
			t.Errorf("withAnd() = %d, want 2", got)
		}
	}
}

func TestComplexity_ForLoop(t *testing.T) {
	src := `
package test
func withLoop(n int) {
	for i := 0; i < n; i++ {
	}
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		got := Complexity(fn)
		if got != 2 {
			t.Errorf("withLoop() = %d, want 2", got)
		}
	}
}

func TestAnalyze(t *testing.T) {
	workDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	projectRoot := filepath.Join(workDir, "..", "..")
	cleanPath := filepath.Clean(projectRoot)
	stats := Analyze([]string{cleanPath}, nil, nil)
	if len(stats) == 0 {
		t.Error("expected some stats, got none")
	}
}

func TestComplexity_CaseClause_default(t *testing.T) {
	src := `package test
func withDefault(x int) int {
	switch x {
	default:
		return 0
	}
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		got := Complexity(fn)
		if got != 1 {
			t.Errorf("withDefault() = %d, want 1", got)
		}
	}
}

func TestComplexity_CaseClause_with_cases(t *testing.T) {
	src := `package test
func withCases(x int) int {
	switch x {
	case 1:
		return 1
	case 2:
		return 2
	}
	return 0
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		got := Complexity(fn)
		if got != 3 {
			t.Errorf("withCases() = %d, want 3", got)
		}
	}
}

func TestComplexity_CommClause_default(t *testing.T) {
	src := `package test
func withSelect(ch chan int) int {
	select {
	default:
		return 0
	}
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		got := Complexity(fn)
		if got != 1 {
			t.Errorf("withSelect() = %d, want 1", got)
		}
	}
}

func TestComplexity_CommClause_non_default(t *testing.T) {
	src := `package test
func withSelect(ch chan int) int {
	select {
	case <-ch:
		return 1
	default:
		return 0
	}
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		got := Complexity(fn)
		if got != 2 {
			t.Errorf("withSelect() = %d, want 2", got)
		}
	}
}

func TestComplexity_BinaryOr(t *testing.T) {
	src := `package test
func withOr(a, b bool) bool {
	return a || b
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		got := Complexity(fn)
		if got != 2 {
			t.Errorf("withOr() = %d, want 2", got)
		}
	}
}

func TestComplexity_Combo(t *testing.T) {
	src := `package test
func complexFunc(a, b, c bool) int {
	if a && b || c {
		return 1
	}
	for i := 0; i < 10; i++ {
		if i > 5 {
			return 2
		}
	}
	return 0
}
`
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, "test.go", src, 0)
	if err != nil {
		t.Fatal(err)
	}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		got := Complexity(fn)
		if got != 6 {
			t.Errorf("complexFunc() = %d, want 6", got)
		}
	}
}

func Test_shouldSkipDir(t *testing.T) {
	assert.True(t, shouldSkipDir(".hidden"))
	assert.True(t, shouldSkipDir("_private"))
	assert.True(t, shouldSkipDir("vendor"))
	assert.True(t, shouldSkipDir("testdata"))
	assert.False(t, shouldSkipDir("pkg"))
	assert.False(t, shouldSkipDir("internal"))
}

func Test_receiverName_nil(t *testing.T) {
	got := receiverName(nil)
	assert.Equal(t, "", got)
}

func Test_receiverName_empty_list(t *testing.T) {
	recv := &ast.FieldList{List: []*ast.Field{}}
	got := receiverName(recv)
	assert.Equal(t, "", got)
}

func Test_receiverName_ptr(t *testing.T) {
	recv := &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.StarExpr{X: &ast.Ident{Name: "MyType"}}},
		},
	}
	got := receiverName(recv)
	assert.Equal(t, "*MyType", got)
}

func Test_receiverName_value(t *testing.T) {
	recv := &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.Ident{Name: "MyType"}},
		},
	}
	got := receiverName(recv)
	assert.Equal(t, "MyType", got)
}

func Test_receiverName_selector(t *testing.T) {
	recv := &ast.FieldList{
		List: []*ast.Field{
			{Type: &ast.SelectorExpr{X: &ast.Ident{Name: "pkg"}, Sel: &ast.Ident{Name: "Type"}}},
		},
	}
	got := receiverName(recv)
	assert.Equal(t, "pkg.Type", got)
}

func TestAnalyze_with_exclude(t *testing.T) {
	workDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	projectRoot := filepath.Join(workDir, "..", "..")
	cleanPath := filepath.Clean(projectRoot)
	regex, _ := regexp.Compile("TestAnalyze|TestComplexity")
	stats := Analyze([]string{cleanPath}, regex, nil)
	for _, s := range stats {
		if s.FuncName == "TestAnalyze" || s.FuncName == "TestComplexity" {
			t.Errorf("should have excluded %s", s.FuncName)
		}
	}
}

type newAnalyzeFn func(*testing.T, *analyzeData)

var checknewAnalyze = func(fns ...newAnalyzeFn) []newAnalyzeFn { return fns }

func Test_newAnalyze(t *testing.T) {
	tests := []struct {
		name    string
		paths   []string
		exclude *regexp.Regexp
		l       logger.Logger
		checks  []newAnalyzeFn
	}{
		{
			name:    "nil logger substitutes dummylogger to avoid panic",
			paths:   []string{"test_trailing\\"},
			exclude: nil,
			l:       nil,
			checks: checknewAnalyze(
				func(t *testing.T, a *analyzeData) {
					t.Helper()
					// With nil logger, real code substitutes dummylogger.
					// Path ending with '\' triggers filepath.Glob error,
					// which calls a.logger.Debug(). Dummylogger handles it;
					// nil logger would panic. This kills the mutation
					// COND_NEG at line 34: l == nil -> l != nil
					assert.NotPanics(t, func() {
						a.analyzeGoFiles("test_trailing\\")
					}, "nil logger should be replaced with no-op dummylogger")
				},
			),
		},
		{
			name:   "TODO: success case",
			checks: checknewAnalyze(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := newAnalyze(tt.paths, tt.exclude, tt.l)
			for _, c := range tt.checks {
				c(t, r)
			}
		})
	}
}
