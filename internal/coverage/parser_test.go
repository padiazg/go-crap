package coverage

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
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

func Test_findFunctionForBlock_body_start_boundary(t *testing.T) {
	src := `package test

func testFunc() {
	x := 1
}
`
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	fn := findFunctionForBlock(fset, node, 3)
	require.NotNil(t, fn)
	assert.Equal(t, "testFunc", fn.name)

	fn = findFunctionForBlock(fset, node, 5)
	require.NotNil(t, fn)
	assert.Equal(t, "testFunc", fn.name)

	fn = findFunctionForBlock(fset, node, 2)
	assert.Nil(t, fn)

	fn = findFunctionForBlock(fset, node, 6)
	assert.Nil(t, fn)
}

func Test_parseProfileLine_boundary_edge_cases(t *testing.T) {
	entry, err := parseProfileLine("file.go:1.1,5.1 0 0")
	assert.NoError(t, err)
	assert.Equal(t, 1, entry.start)
	assert.False(t, entry.covered)

	entry, err = parseProfileLine("file.go:100.1,100.1 1 1")
	assert.NoError(t, err)
	assert.Equal(t, 100, entry.start)
	assert.True(t, entry.covered)
}

func Test_findFunctionForBlock_boundary_nested_functions(t *testing.T) {
	src := `package test

func outer() {
	inner := func() {
		x := 1
	}
	_ = inner
}
`
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	fn := findFunctionForBlock(fset, node, 5)
	require.NotNil(t, fn)
	assert.Equal(t, "outer", fn.name)

	fn = findFunctionForBlock(fset, node, 4)
	require.NotNil(t, fn)
	assert.Equal(t, "outer", fn.name)
}

func Test_findFunctionForBlock_boundary_multiple_lines(t *testing.T) {
	src := `package test

func testFunc() {
	x := 1
	y := 2
	z := 3
}
`
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	for line := 4; line <= 7; line++ {
		fn := findFunctionForBlock(fset, node, line)
		require.NotNil(t, fn)
		assert.Equal(t, "testFunc", fn.name)
	}
}

func Test_parseCoverProfile_boundary_empty_profile(t *testing.T) {
	tempDir := t.TempDir()
	profPath := filepath.Join(tempDir, "cover.out")
	os.WriteFile(profPath, []byte("mode: set\n"), 0644)

	entries, err := parseCoverProfile(profPath, tempDir, "mod/path")
	assert.NoError(t, err)
	assert.Empty(t, entries)
}

func Test_parseCoverProfile_boundary_multiple_blocks_same_func(t *testing.T) {
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "to")
	os.MkdirAll(subDir, 0755)

	goSrc := `package testdata

func MultipleBlocks() {
	x := 1
	y := 2
}
`
	os.WriteFile(filepath.Join(subDir, "file.go"), []byte(goSrc), 0644)
	profPath := filepath.Join(tempDir, "cover.out")
	os.WriteFile(profPath, []byte("mode: set\n"+
		"mod/path/to/file.go:1.1,10.1 1 0\n"+
		"mod/path/to/file.go:1.1,10.1 1 1\n"), 0644)

	entries, err := parseCoverProfile(profPath, tempDir, "mod/path")
	assert.NoError(t, err)
	assert.NotEmpty(t, entries)
}

func Test_parseCoverProfile_boundary_no_go_files(t *testing.T) {
	tempDir := t.TempDir()
	profPath := filepath.Join(tempDir, "cover.out")
	os.WriteFile(profPath, []byte("mode: set\n"+
		"nonexistent.go:1.1,5.1 1 1\n"), 0644)

	entries, err := parseCoverProfile(profPath, tempDir, "mod/path")
	assert.NoError(t, err)
	assert.Empty(t, entries)
}

func Test_parseProfileLine_boundary_no_colon(t *testing.T) {
	_, err := parseProfileLine("invalidline")
	assert.Error(t, err)
}

func Test_parseProfileLine_boundary_colon_at_zero(t *testing.T) {
	_, err := parseProfileLine(":10.5,20.10 0 1")
	assert.Error(t, err, "colon at position 0 means empty path should be rejected")
}

func Test_readProfileEntries_boundary_empty_reader(t *testing.T) {
	entries, err := readProfileEntries(strings.NewReader(""))
	assert.NoError(t, err)
	assert.Empty(t, entries)
}

func Test_readProfileEntries_boundary_only_header(t *testing.T) {
	entries, err := readProfileEntries(strings.NewReader("mode: set\n\nmode: count\n"))
	assert.NoError(t, err)
	assert.Empty(t, entries)
}

func Test_buildFuncMap_empty(t *testing.T) {
	funcMap, ordered := buildFuncMap([]fileFunc{})
	assert.Empty(t, funcMap)
	assert.Empty(t, ordered)
}

func Test_buildFuncMap_with_receiver(t *testing.T) {
	funcs := []fileFunc{
		{name: "MyType.Do", startLine: 5, endLine: 10, declLine: 5},
		{name: "standalone", startLine: 15, endLine: 20, declLine: 15},
	}
	funcMap, ordered := buildFuncMap(funcs)
	assert.Equal(t, 2, len(funcMap))
	assert.Equal(t, 2, len(ordered))
	assert.Equal(t, "MyType.Do", ordered[0])
	assert.Equal(t, "standalone", ordered[1])
	assert.Equal(t, "MyType.Do", funcMap["MyType.Do"].name)
}

func Test_buildCoverageResults_zero_total(t *testing.T) {
	funcMap := map[string]*funcCoverage{
		"noBlocks": {name: "noBlocks", declLine: 1, total: 0, covered: 0},
	}
	ordered := []string{"noBlocks"}
	results := buildCoverageResults(funcMap, ordered, "test.go", "mod/path")
	assert.Len(t, results, 1)
	assert.Equal(t, 0.0, results[0].Coverage)
}

func Test_buildCoverageResults_full_coverage(t *testing.T) {
	funcMap := map[string]*funcCoverage{
		"allCovered": {name: "allCovered", declLine: 1, total: 5, covered: 5},
	}
	ordered := []string{"allCovered"}
	results := buildCoverageResults(funcMap, ordered, "test.go", "mod/path")
	assert.Len(t, results, 1)
	assert.Equal(t, 100.0, results[0].Coverage)
}

func Test_buildCoverageResults_partial_coverage(t *testing.T) {
	funcMap := map[string]*funcCoverage{
		"partial": {name: "partial", declLine: 1, total: 3, covered: 1},
	}
	ordered := []string{"partial"}
	results := buildCoverageResults(funcMap, ordered, "test.go", "mod/path")
	assert.Len(t, results, 1)
	assert.Equal(t, 33.33333333333333, results[0].Coverage)
}

func Test_extractFileFuncs(t *testing.T) {
	tests := []struct {
		name      string
		src       string
		wantFuncs []fileFunc
	}{
		{
			name: "simple_function",
			src: `package test
func myFunc() {
	x := 1
}
`,
			wantFuncs: []fileFunc{
				{name: "myFunc", startLine: 2, endLine: 4, declLine: 2},
			},
		},
		{
			name: "function_on_multi_line_signature",
			src: `package test
func myFunc(
	a int,
) {
	x := 1
}
`,
			wantFuncs: []fileFunc{
				{name: "myFunc", startLine: 2, endLine: 6, declLine: 4},
			},
		},
		{
			name: "method_with_value_receiver",
			src: `package test
type MyType struct{}
func (m MyType) MyMethod() {
	x := 1
}
`,
			wantFuncs: []fileFunc{
				{name: "MyType.MyMethod", startLine: 3, endLine: 5, declLine: 3},
			},
		},
		{
			name: "method_with_pointer_receiver",
			src: `package test
type MyType struct{}
func (m *MyType) MyMethod() {
	x := 1
}
`,
			wantFuncs: []fileFunc{
				{name: "*MyType.MyMethod", startLine: 3, endLine: 5, declLine: 3},
			},
		},
		{
			name: "multiple_functions",
			src: `package test
func first() {
}
func second() {
}
`,
			wantFuncs: []fileFunc{
				{name: "first", startLine: 2, endLine: 3, declLine: 2},
				{name: "second", startLine: 4, endLine: 5, declLine: 4},
			},
		},
		{
			name: "no_functions",
			src: `package test
const X = 1
`,
			wantFuncs: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			node, err := parser.ParseFile(fset, "test.go", tt.src, 0)
			require.NoError(t, err)
			got := extractFileFuncs(fset, node)
			assert.Equal(t, tt.wantFuncs, got)
		})
	}
}

func Test_resolveReceiverPrefix(t *testing.T) {
	src := `package test

type MyType struct{}

func standaloneFunc() {}

func (m MyType) ValueMethod() {}

func (m *MyType) PointerMethod() {}
`
	fset := token.NewFileSet()
	node, err := parser.ParseFile(fset, "test.go", src, 0)
	require.NoError(t, err)

	tests := []struct {
		name     string
		funcName string
		want     string
	}{
		{name: "standalone_function", funcName: "standaloneFunc", want: "standaloneFunc"},
		{name: "value_receiver_method", funcName: "ValueMethod", want: "MyType.ValueMethod"},
		{name: "pointer_receiver_method", funcName: "PointerMethod", want: "*MyType.PointerMethod"},
		{name: "function_not_found", funcName: "NoSuchFunc", want: "NoSuchFunc"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := resolveReceiverPrefix(node, tt.funcName)
			assert.Equal(t, tt.want, got)
		})
	}
}

func Test_parsePositionFields(t *testing.T) {
	tests := []struct {
		name    string
		rest    string
		want    int
		want2   bool
		wantErr string
	}{
		{
			name:    "normal_covered",
			rest:    "10.5,20.10 0 1",
			want:    10,
			want2:   true,
			wantErr: "",
		},
		{
			name:    "normal_not_covered",
			rest:    "10.5,20.10 0 0",
			want:    10,
			want2:   false,
			wantErr: "",
		},
		{
			name:    "zero_line",
			rest:    "0.1,0.1 1 1",
			want:    0,
			want2:   true,
			wantErr: "",
		},
		{
			name:    "space_at_position_zero",
			rest:    " 10.5,20.10 0 1",
			want:    0,
			want2:   false,
			wantErr: "invalid line",
		},
		{
			name:    "no_space_in_rest",
			rest:    "10.5,20.10",
			want:    0,
			want2:   false,
			wantErr: "invalid line",
		},
		{
			name:    "position_without_comma",
			rest:    "10.5 0 1",
			want:    0,
			want2:   false,
			wantErr: "invalid position",
		},
		{
			name:    "too_many_comma_parts",
			rest:    "10.5,20.10,extra 0 1",
			want:    0,
			want2:   false,
			wantErr: "invalid position",
		},
		{
			name:    "single_covered_field",
			rest:    "10.5,20.10 0",
			want:    0,
			want2:   false,
			wantErr: "invalid fields",
		},
		{
			name:    "non_numeric_covered_value",
			rest:    "10.5,20.10 0 abc",
			want:    10,
			want2:   false,
			wantErr: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r, r2, err := parsePositionFields(tt.rest)
			assert.Equal(t, tt.want, r)
			assert.Equal(t, tt.want2, r2)
			if tt.wantErr != "" {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), tt.wantErr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

type findInnermostFuncFn func(*testing.T, *funcDeclInfo)

var checkfindInnermostFunc = func(fns ...findInnermostFuncFn) []findInnermostFuncFn { return fns }

func Test_findInnermostFunc(t *testing.T) {
	parseTestSrc := func(src string) (*token.FileSet, *ast.File) {
		fset := token.NewFileSet()
		node, err := parser.ParseFile(fset, "test.go", src, 0)
		require.NoError(t, err)
		return fset, node
	}

	checkName := func(want string) findInnermostFuncFn {
		return func(t *testing.T, f *funcDeclInfo) {
			t.Helper()
			assert.Equalf(t, want, f.name, "name: %s, expected %s", f.name, want)
		}
	}

	checkBodyStart := func(want int) findInnermostFuncFn {
		return func(t *testing.T, f *funcDeclInfo) {
			t.Helper()
			assert.Equalf(t, want, f.bodyStart, "bodyStart: %d, expected %d", f.bodyStart, want)
		}
	}

	checkBodyEnd := func(want int) findInnermostFuncFn {
		return func(t *testing.T, f *funcDeclInfo) {
			t.Helper()
			assert.Equalf(t, want, f.bodyEnd, "bodyEnd: %d, expected %d", f.bodyEnd, want)
		}
	}

	checkNil := func(t *testing.T, f *funcDeclInfo) {
		t.Helper()
		assert.Nil(t, f)
	}

	tests := []struct {
		name      string
		src       string
		blockLine int
		checks    []findInnermostFuncFn
	}{
		{
			name:      "block_at_body_start",
			src:       "package test\nfunc foo() {\n\tx := 1\n}\n",
			blockLine: 2,
			checks: checkfindInnermostFunc(
				checkName("foo"),
				checkBodyStart(2),
				checkBodyEnd(4),
			),
		},
		{
			name:      "block_inside_function_range",
			src:       "package test\nfunc foo() {\n\ta := 1\n\tb := 2\n}\n",
			blockLine: 3,
			checks: checkfindInnermostFunc(
				checkName("foo"),
				checkBodyStart(2),
				checkBodyEnd(5),
			),
		},
		{
			name:      "outside_function_body",
			src:       "package test\nfunc foo() {\n\tx := 1\n}\n",
			blockLine: 1,
			checks:    checkfindInnermostFunc(checkNil),
		},
		{
			name:      "after_function_body",
			src:       "package test\nfunc foo() {\n\tx := 1\n}\n",
			blockLine: 5,
			checks:    checkfindInnermostFunc(checkNil),
		},
		{
			name: "non_overlapping_funcs_selects_first",
			src: `package test
func foo() {
	x := 1
}
func bar() {
	y := 2
}
`,
			blockLine: 2,
			checks: checkfindInnermostFunc(
				checkName("foo"),
				checkBodyStart(2),
				checkBodyEnd(4),
			),
		},
		{
			name: "non_overlapping_funcs_selects_second",
			src: `package test
func foo() {
	x := 1
}
func bar() {
	y := 2
}
`,
			blockLine: 6,
			checks: checkfindInnermostFunc(
				checkName("bar"),
				checkBodyStart(5),
				checkBodyEnd(7),
			),
		},
		{
			name: "func_lit_not_matched",
			src: `package test
func outer() {
	f := func() {
		x := 1
	}
	_ = f
}
`,
			blockLine: 4,
			checks: checkfindInnermostFunc(
				checkName("outer"),
				checkBodyStart(2),
				checkBodyEnd(7),
			),
		},
		{
			name: "multiple_blocks_in_range",
			src: `package test
func foo() {
	a := 1
	b := 2
	c := 3
}
`,
			blockLine: 4,
			checks: checkfindInnermostFunc(
				checkName("foo"),
				checkBodyStart(2),
				checkBodyEnd(6),
			),
		},
		{
			name: "same_bodyStart_keeps_first_found",
			src: `package test
func foo() { x := 1 }; func bar() { y := 2 }
`,
			blockLine: 2,
			checks: checkfindInnermostFunc(
				checkName("foo"),
				checkBodyStart(2),
				checkBodyEnd(2),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset, node := parseTestSrc(tt.src)
			r := findInnermostFunc(fset, node, tt.blockLine)
			for _, c := range tt.checks {
				c(t, r)
			}
		})
	}
}

func Test_parseCoord(t *testing.T) {
	tests := []struct {
		name     string
		s        string
		wantLine int
		wantCol  int
	}{
		{name: "normal", s: "10.5", wantLine: 10, wantCol: 5},
		{name: "zero_line", s: "0.1", wantLine: 0, wantCol: 1},
		{name: "multi_digit_line_col", s: "123.456", wantLine: 123, wantCol: 456},
		{name: "dot_at_start", s: ".5", wantLine: 0, wantCol: 0},
		{name: "no_dot", s: "123", wantLine: 0, wantCol: 0},
		{name: "negative_line", s: "-1.5", wantLine: -1, wantCol: 5},
		{name: "non_numeric_line", s: "abc.5", wantLine: 0, wantCol: 0},
		{name: "non_numeric_col", s: "10.abc", wantLine: 10, wantCol: 0},
		{name: "empty_string", s: "", wantLine: 0, wantCol: 0},
		{name: "not_present", s: "abc", wantLine: 0, wantCol: 0},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			l, c := parseCoord(tt.s)
			assert.Equalf(t, tt.wantLine, l, "line: %d, want %d", l, tt.wantLine)
			assert.Equalf(t, tt.wantCol, c, "col: %d, want %d", c, tt.wantCol)
		})
	}
}
