package merge

import (
	"testing"

	"github.com/padiazg/go-crap/internal/complexity"
	"github.com/padiazg/go-crap/internal/coverage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type MergeFn func(*testing.T, []MergedEntry)

var checkMerge = func(fns ...MergeFn) []MergeFn { return fns }

func checkLen(count int) MergeFn {
	return func(t *testing.T, m []MergedEntry) {
		t.Helper()
		assert.Equal(t, count, len(m))
	}
}

func checkCoverage(index int, wantNil bool) MergeFn {
	return func(t *testing.T, m []MergedEntry) {
		t.Helper()
		if wantNil {
			assert.Nil(t, m[index].Coverage, "expected nil coverage for %s", m[index].FuncName)
		} else {
			assert.NotNil(t, m[index].Coverage, "expected non-nil coverage for %s", m[index].FuncName)
		}
	}
}

func checkCoverageValue(index int, want float64) MergeFn {
	return func(t *testing.T, m []MergedEntry) {
		t.Helper()
		if assert.NotNil(t, m[index].Coverage, "expected non-nil coverage for %s", m[index].FuncName) {
			assert.Equal(t, want, *m[index].Coverage, "coverage mismatch for %s", m[index].FuncName)
		}
	}
}

func checkFuncName(index int, want string) MergeFn {
	return func(t *testing.T, m []MergedEntry) {
		t.Helper()
		assert.Equal(t, want, m[index].FuncName)
	}
}

func TestMerge(t *testing.T) {
	tests := []struct {
		name      string
		coverages []coverage.ModuleCoverage
		stats     []complexity.Stat
		checks    []MergeFn
	}{
		{
			name: "function with zero coverage returns pointer to 0",
			coverages: []coverage.ModuleCoverage{
				{
					Dir:        "/test",
					ModulePath: "test/pkg",
					Functions: []coverage.FunctionCoverage{
						{
							File:     "/test/pkg/foo.go",
							Package:  "pkg",
							Name:     "Foo",
							Line:     1,
							Coverage: 0.0,
						},
					},
				},
			},
			stats: []complexity.Stat{
				{
					PkgName:    "pkg",
					FuncName:   "Foo",
					Complexity: 5,
					Pos: struct {
						Filename string
						Offset   int
						Line     int
						Column   int
					}{
						Filename: "/test/pkg/foo.go",
						Line:     1,
					},
				},
			},
			checks: checkMerge(
				checkLen(1),
				checkFuncName(0, "Foo"),
				checkCoverage(0, false),
				checkCoverageValue(0, 0.0),
			),
		},
		{
			name: "function with positive coverage returns pointer",
			coverages: []coverage.ModuleCoverage{
				{
					Dir:        "/test",
					ModulePath: "test/pkg",
					Functions: []coverage.FunctionCoverage{
						{
							File:     "/test/pkg/foo.go",
							Package:  "pkg",
							Name:     "Foo",
							Line:     1,
							Coverage: 100.0,
						},
						{
							File:     "/test/pkg/bar.go",
							Package:  "pkg",
							Name:     "Bar",
							Line:     5,
							Coverage: 50.0,
						},
					},
				},
			},
			stats: []complexity.Stat{
				{
					PkgName:    "pkg",
					FuncName:   "Foo",
					Complexity: 5,
					Pos: struct {
						Filename string
						Offset   int
						Line     int
						Column   int
					}{
						Filename: "/test/pkg/foo.go",
						Line:     1,
					},
				},
				{
					PkgName:    "pkg",
					FuncName:   "Bar",
					Complexity: 3,
					Pos: struct {
						Filename string
						Offset   int
						Line     int
						Column   int
					}{
						Filename: "/test/pkg/bar.go",
						Line:     5,
					},
				},
			},
			checks: checkMerge(
				checkLen(2),
				checkFuncName(0, "Foo"),
				checkCoverage(0, false),
				checkCoverageValue(0, 100.0),
				checkFuncName(1, "Bar"),
				checkCoverage(1, false),
				checkCoverageValue(1, 50.0),
			),
		},
		{
			name: "function not in coverage returns nil",
			coverages: []coverage.ModuleCoverage{
				{
					Dir:        "/test",
					ModulePath: "test/pkg",
					Functions:  []coverage.FunctionCoverage{},
				},
			},
			stats: []complexity.Stat{
				{
					PkgName:    "pkg",
					FuncName:   "Baz",
					Complexity: 2,
					Pos: struct {
						Filename string
						Offset   int
						Line     int
						Column   int
					}{
						Filename: "/test/pkg/baz.go",
						Line:     10,
					},
				},
			},
			checks: checkMerge(
				checkLen(1),
				checkFuncName(0, "Baz"),
				checkCoverage(0, true),
			),
		},
		{
			name: "mixed zero and non-zero coverage",
			coverages: []coverage.ModuleCoverage{
				{
					Dir:        "/test",
					ModulePath: "test/pkg",
					Functions: []coverage.FunctionCoverage{
						{
							File:     "/test/pkg/a.go",
							Package:  "pkg",
							Name:     "A",
							Line:     1,
							Coverage: 0.0,
						},
						{
							File:     "/test/pkg/b.go",
							Package:  "pkg",
							Name:     "B",
							Line:     2,
							Coverage: 25.5,
						},
					},
				},
			},
			stats: []complexity.Stat{
				{
					PkgName:    "pkg",
					FuncName:   "A",
					Complexity: 3,
					Pos: struct {
						Filename string
						Offset   int
						Line     int
						Column   int
					}{
						Filename: "/test/pkg/a.go",
						Line:     1,
					},
				},
				{
					PkgName:    "pkg",
					FuncName:   "B",
					Complexity: 1,
					Pos: struct {
						Filename string
						Offset   int
						Line     int
						Column   int
					}{
						Filename: "/test/pkg/b.go",
						Line:     2,
					},
				},
			},
			checks: checkMerge(
				checkLen(2),
				checkFuncName(0, "A"),
				checkCoverage(0, false),
				checkCoverageValue(0, 0.0),
				checkFuncName(1, "B"),
				checkCoverage(1, false),
				checkCoverageValue(1, 25.5),
			),
		},
		{
			name:      "empty inputs",
			coverages: []coverage.ModuleCoverage{},
			stats:     []complexity.Stat{},
			checks:    checkMerge(checkLen(0)),
		},
		{
			name: "module path vs filesystem path match",
			coverages: []coverage.ModuleCoverage{
				{
					Dir:        "/home/runner/work/go-crap/go-crap",
					ModulePath: "github.com/padiazg/go-crap",
					Functions: []coverage.FunctionCoverage{
						{
							File:     "github.com/padiazg/go-crap/internal/merge/merge.go",
							Package:  "merge",
							Name:     "Merge",
							Line:     74,
							Coverage: 94.1,
						},
					},
				},
			},
			stats: []complexity.Stat{
				{
					PkgName:    "merge",
					FuncName:   "Merge",
					Complexity: 7,
					Pos: struct {
						Filename string
						Offset   int
						Line     int
						Column   int
					}{
						Filename: "/home/runner/work/go-crap/go-crap/internal/merge/merge.go",
						Line:     74,
					},
				},
			},
			checks: checkMerge(
				checkLen(1),
				checkFuncName(0, "Merge"),
				checkCoverage(0, false),
				checkCoverageValue(0, 94.1),
			),
		},
		{
			name: "relative filesystem path with module suffix",
			coverages: []coverage.ModuleCoverage{
				{
					Dir:        "/test",
					ModulePath: "test/pkg",
					Functions: []coverage.FunctionCoverage{
						{
							File:     "test/pkg/foo.go",
							Package:  "pkg",
							Name:     "Foo",
							Line:     1,
							Coverage: 75.0,
						},
					},
				},
			},
			stats: []complexity.Stat{
				{
					PkgName:    "pkg",
					FuncName:   "Foo",
					Complexity: 5,
					Pos: struct {
						Filename string
						Offset   int
						Line     int
						Column   int
					}{
						Filename: "test/pkg/foo.go",
						Line:     1,
					},
				},
			},
			checks: checkMerge(
				checkLen(1),
				checkFuncName(0, "Foo"),
				checkCoverage(0, false),
				checkCoverageValue(0, 75.0),
			),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := Merge(tt.coverages, tt.stats)
			for _, c := range tt.checks {
				c(t, r)
			}
		})
	}
}

func TestMerge_ValueReceiverMatch(t *testing.T) {
	coverages := []coverage.ModuleCoverage{
		{
			Dir:        "/test",
			ModulePath: "test/pkg",
			Functions: []coverage.FunctionCoverage{
				{
					File:     "/test/pkg/logger.go",
					Package:  "logger",
					Name:     "Level.String",
					Line:     21,
					Coverage: 100.0,
				},
			},
		},
	}
	stats := []complexity.Stat{
		{
			PkgName:  "logger",
			FuncName: "String",
			Receiver: "Level",
			Pos: struct {
				Filename string
				Offset   int
				Line     int
				Column   int
			}{
				Filename: "/test/pkg/logger.go",
				Line:     21,
			},
			Complexity: 6,
		},
	}
	r := Merge(coverages, stats)
	for _, c := range checkMerge(
		checkLen(1),
		checkFuncName(0, "Level.String"),
		checkCoverage(0, false),
		checkCoverageValue(0, 100.0),
	) {
		c(t, r)
	}
}

func TestMerge_RelativePathsNotResolvedAgainstCWD(t *testing.T) {
	coverages := []coverage.ModuleCoverage{
		{
			Dir:        "/some/long/path",
			ModulePath: "test/pkg",
			Functions: []coverage.FunctionCoverage{
				{
					File:     "internal/foo/bar.go",
					Package:  "pkg",
					Name:     "Foo",
					Line:     10,
					Coverage: 80.0,
				},
			},
		},
	}
	stats := []complexity.Stat{
		{
			PkgName:    "pkg",
			FuncName:   "Foo",
			Complexity: 5,
			Pos: struct {
				Filename string
				Offset   int
				Line     int
				Column   int
			}{
				Filename: "internal/foo/bar.go",
				Line:     10,
			},
		},
	}
	r := Merge(coverages, stats)
	for _, c := range checkMerge(
		checkLen(1),
		checkFuncName(0, "Foo"),
		checkCoverage(0, false),
		checkCoverageValue(0, 80.0),
	) {
		c(t, r)
	}
	assert.Equal(t, "internal/foo/bar.go", r[0].File, "relative path should be preserved unchanged")
}

func TestMerge_MethodMatch(t *testing.T) {
	tests := []struct {
		name           string
		complexityName string
		coverageName   string
		wantMatch      bool
	}{
		{
			name:           "pointer receiver method (*Type).Method",
			complexityName: "(*JSONFormatter).Format",
			coverageName:   "Format",
			wantMatch:      true,
		},
		{
			name:           "value receiver method Type.Method",
			complexityName: "Level.String",
			coverageName:   "String",
			wantMatch:      true,
		},
		{
			name:           "plain function",
			complexityName: "Foo",
			coverageName:   "Foo",
			wantMatch:      true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			normalized := normalizeFuncName(tt.complexityName)
			if normalized != tt.coverageName {
				t.Errorf("normalizeFuncName(%q) = %q, want %q", tt.complexityName, normalized, tt.coverageName)
			}
		})
	}
}

func Test_buildIndex_empty(t *testing.T) {
	idx := buildIndex(nil)
	require.NotNil(t, idx)
	assert.Empty(t, idx.byAbsolute)
	assert.Empty(t, idx.bySuffix)
}

func Test_buildIndex_with_data(t *testing.T) {
	coverages := []coverage.ModuleCoverage{
		{
			ModulePath: "mod/path",
			Functions: []coverage.FunctionCoverage{
				{File: "/home/user/mod/path/file.go", Name: "Func1", Coverage: 100.0},
			},
		},
	}
	idx := buildIndex(coverages)
	require.NotNil(t, idx)
	require.NotEmpty(t, idx.byAbsolute)
	require.NotEmpty(t, idx.bySuffix)
	absKey := "/home/user/mod/path/file.go"
	_, ok := idx.byAbsolute[absKey]
	assert.True(t, ok)
}

func Test_pathIndex_lookup_by_absolute(t *testing.T) {
	coverages := []coverage.ModuleCoverage{
		{
			ModulePath: "mod/path",
			Functions: []coverage.FunctionCoverage{
				{File: "/home/user/mod/path/file.go", Name: "Func1", Coverage: 100.0},
			},
		},
	}
	idx := buildIndex(coverages)
	fns, ok := idx.lookup("/home/user/mod/path/file.go")
	assert.True(t, ok)
	assert.NotEmpty(t, fns)
	assert.Equal(t, "Func1", fns[0].Name)
}

func Test_pathIndex_lookup_by_suffix(t *testing.T) {
	coverages := []coverage.ModuleCoverage{
		{
			ModulePath: "github.com/user/pkg",
			Functions: []coverage.FunctionCoverage{
				{File: "/home/user/gopath/src/github.com/user/pkg/file.go", Name: "Func1", Coverage: 50.0},
			},
		},
	}
	idx := buildIndex(coverages)
	fns, ok := idx.lookup("user/pkg/file.go")
	assert.True(t, ok)
	assert.NotEmpty(t, fns)
	assert.Equal(t, "Func1", fns[0].Name)
}

func Test_pathIndex_lookup_not_found(t *testing.T) {
	idx := buildIndex(nil)
	_, ok := idx.lookup("/nonexistent/path/file.go")
	assert.False(t, ok)
}

func Test_buildSuffix(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
	}{
		{"single segment no slash", "pkg", "pkg"},
		{"single file", "file.go", "file.go"},
		{"single path segment", "/path", "path"},
		{"two segments", "a/b", "a/b"},
		{"vendor path", "vendor/pkg", "vendor/pkg"},
		{"internal path", "internal/pkg", "internal/pkg"},
		{"three segments", "a/b/c", "a/b/c"},
		{"more than three segments", "a/b/c/d/e", "c/d/e"},
		{"internal pkg file", "internal/pkg/file", "internal/pkg/file"},
		{"module path three", "github.com/user/pkg", "github.com/user/pkg"},
		{"module subpackage", "github.com/user/pkg/subpkg/file", "pkg/subpkg/file"},
		{"windows path", "C:\\Users\\pkg\\subpkg\\file", "pkg/subpkg/file"},
		{"windows root path", "C:\\file.go", "C:/file.go"},
		{"empty segments", "/a//b", "a/b"},
		{"leading slash", "/a/b/c", "a/b/c"},
		{"leading and trailing slash", "/a/b/c/", "a/b/c"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := buildSuffix(tt.path)
			assert.Equal(t, tt.want, r)
		})
	}
}
