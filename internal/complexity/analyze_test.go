package complexity

import (
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"testing"
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
