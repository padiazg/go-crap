package coverage

import (
	"fmt"
	"go/ast"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParseCoverOutput(t *testing.T) {
	input := `github.com/padiazg/go-crap/internal/merge/merge.go:45: Merge 87.5%
github.com/padiazg/go-crap/internal/merge/merge.go:12: buildIndex 45.2%
total: (statements) 82.3%
`
	functions, err := parseCoverOutput(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(functions) != 2 {
		t.Fatalf("expected 2 functions, got %d", len(functions))
	}
	if functions[0].Name != "Merge" {
		t.Errorf("expected Merge, got %s", functions[0].Name)
	}
	if functions[0].Coverage != 87.5 {
		t.Errorf("expected 87.5 coverage, got %f", functions[0].Coverage)
	}
	if functions[0].Line != 45 {
		t.Errorf("expected line 45, got %d", functions[0].Line)
	}
}

func TestParseCoverOutput_SkipsTotal(t *testing.T) {
	input := `total: (statements) 82.3%
`
	functions, err := parseCoverOutput(strings.NewReader(input))
	if err != nil {
		t.Fatal(err)
	}
	if len(functions) != 0 {
		t.Errorf("expected 0 functions, got %d", len(functions))
	}
}

func TestParseCoverProfile(t *testing.T) {
	functions, err := parseCoverProfile(
		"../testdata/cover.out",
		"../testdata",
		"github.com/padiazg/go-crap/internal/testdata",
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(functions) == 0 {
		t.Fatal("expected functions, got none")
	}
	funcMap := make(map[string]float64)
	for _, f := range functions {
		funcMap[f.Name] = f.Coverage
	}
	if cov, ok := funcMap["simple"]; !ok {
		t.Error("expected simple function")
	} else if cov != 100.0 {
		t.Errorf("expected simple 100%% coverage, got %f", cov)
	}
	if cov, ok := funcMap["veryComplex"]; !ok {
		t.Error("expected veryComplex function")
	} else if cov != 0.0 {
		t.Errorf("expected veryComplex 0%% coverage, got %f", cov)
	}
}

func Test_extractRecvName(t *testing.T) {
	tests := []struct {
		name     string
		recv     *ast.FieldList
		want     string
		wantStr  string
		hasField bool
	}{
		{
			name:     "nil_receiver",
			recv:     nil,
			want:     "",
			hasField: false,
		},
		{
			name:     "empty_field_list",
			recv:     &ast.FieldList{},
			want:     "",
			hasField: false,
		},
		{
			name:     "value_receiver_ident",
			recv:     &ast.FieldList{List: []*ast.Field{{Type: &ast.Ident{Name: "User"}}}},
			want:     "User",
			wantStr:  "User",
			hasField: true,
		},
		{
			name:     "pointer_receiver_star_ident",
			recv:     &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: &ast.Ident{Name: "User"}}}}},
			want:     "*User",
			wantStr:  "*User",
			hasField: true,
		},
		{
			name:     "selector_expr_receiver",
			recv:     &ast.FieldList{List: []*ast.Field{{Type: &ast.SelectorExpr{X: &ast.Ident{Name: "pkg"}}}}},
			want:     "pkg",
			wantStr:  "pkg",
			hasField: true,
		},
		{
			name:     "pointer_to_selector_expr_receiver",
			recv:     &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: &ast.SelectorExpr{X: &ast.Ident{Name: "pkg"}}}}}},
			want:     "*pkg",
			wantStr:  "*pkg",
			hasField: true,
		},
		{
			name:     "star_expr_non_ident_X",
			recv:     &ast.FieldList{List: []*ast.Field{{Type: &ast.StarExpr{X: &ast.SelectorExpr{X: &ast.SelectorExpr{X: &ast.Ident{Name: "x"}}}}}}},
			want:     "",
			wantStr:  "",
			hasField: true,
		},
		{
			name:     "selector_expr_non_ident_X",
			recv:     &ast.FieldList{List: []*ast.Field{{Type: &ast.SelectorExpr{X: &ast.SelectorExpr{X: &ast.Ident{Name: "x"}}}}}},
			want:     "",
			wantStr:  "",
			hasField: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := extractRecvName(tt.recv)
			assert.Equal(t, tt.want, r)
			assert.Equal(t, tt.wantStr, fmt.Sprintf("%s", r))
			hasField := tt.recv != nil && len(tt.recv.List) > 0
			assert.Equal(t, tt.hasField, hasField)
		})
	}
}

func Test_lookupFuncName(t *testing.T) {
	wd, err := os.Getwd()
	require.NoError(t, err)

	absSimple := filepath.Join(wd, "..", "testdata", "simple.go")

	tempDir, err := os.MkdirTemp("", "crap-lookup-")
	require.NoError(t, err)
	defer os.RemoveAll(tempDir)

	emptyFile := filepath.Join(tempDir, "empty.go")
	require.NoError(t, os.WriteFile(emptyFile, nil, 0644))

	methodSrc := `package testdata

type MyType struct{}

func (m MyType) DoStuff() {}

func (m *MyType) DoPointerStuff() {}

func standalone() {}
`
	methodFile := filepath.Join(tempDir, "method.go")
	require.NoError(t, os.WriteFile(methodFile, []byte(methodSrc), 0644))

	tests := []struct {
		name        string
		modDir      string
		profilePath string
		want        string
	}{
		{
			name:        "relative_path_simple_function",
			modDir:      filepath.Join(wd, "..", "testdata"),
			profilePath: "simple.go",
			want:        "simple",
		},
		{
			name:        "relative_path_complex_file",
			modDir:      filepath.Join(wd, "..", "testdata"),
			profilePath: "complex.go",
			want:        "veryComplex",
		},
		{
			name:        "absolute_path",
			modDir:      filepath.Join(wd, "..", "testdata"),
			profilePath: absSimple,
			want:        "simple",
		},
		{
			name:        "file_not_found",
			modDir:      filepath.Join(wd, "..", "testdata"),
			profilePath: "does_not_exist.go",
			want:        "",
		},
		{
			name:        "no_functions",
			modDir:      tempDir,
			profilePath: "empty.go",
			want:        "",
		},
		{
			name:        "value_receiver",
			modDir:      tempDir,
			profilePath: "method.go",
			want:        "MyType.DoStuff",
		},
		{
			name:        "non_go_file",
			modDir:      tempDir,
			profilePath: "non_go.txt",
			want:        "",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r := lookupFuncName(tt.modDir, tt.profilePath)
			assert.Equal(t, tt.want, r)
		})
	}
}
