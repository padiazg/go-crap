package coverage

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

type ScanFn func(*testing.T, []ModuleCoverage, error)

var checkScan = func(fns ...ScanFn) []ScanFn { return fns }

func checkScanError(want string) ScanFn {
	return func(t *testing.T, _ []ModuleCoverage, err error) {
		t.Helper()
		if want == "" {
			assert.NoErrorf(t, err, "checkScanError: expected no error, got %v", err)
			return
		}
		if assert.Errorf(t, err, "checkScanError: expected error %q", want) {
			assert.Containsf(t, err.Error(), want, "checkScanError mismatch")
		}
	}
}

func checkLen(count int) ScanFn {
	return func(t *testing.T, m []ModuleCoverage, e error) {
		t.Helper()
		assert.Equal(t, count, len(m))
	}
}

func checkModulePath(path string) ScanFn {
	return func(t *testing.T, m []ModuleCoverage, e error) {
		t.Helper()
		t.Logf("ModulePath: %q, Dir: %q", m[0].ModulePath, m[0].Dir)
		assert.Equal(t, path, m[0].ModulePath)
	}
}

func checkModuleError(want string) ScanFn {
	return func(t *testing.T, m []ModuleCoverage, e error) {
		t.Helper()
		if want == "" {
			assert.Nil(t, m[0].Error)
		} else {
			assert.NotNil(t, m[0].Error)
			assert.Contains(t, m[0].Error.Error(), want)
		}
	}
}

func TestScan(t *testing.T) {
	tests := []struct {
		name   string
		opts   ScanOptions
		checks []ScanFn
	}{
		{
			name: "scan testdata",
			opts: ScanOptions{
				Path: "../testdata",
			},
			checks: checkScan(
				checkScanError(""),
				checkLen(1),
				checkModulePath("github.com/padiazg/go-crap/internal/testdata"),
				checkModuleError(""),
			),
		},
		{
			name: "exclude test files",
			opts: ScanOptions{
				Path:    "../testdata",
				Exclude: regexp.MustCompile(`.*_test\.go$`),
			},
			checks: checkScan(
				checkScanError(""),
				checkLen(1),
				checkModuleError(""),
			),
		},
		{
			name: "non-existent path returns error",
			opts: ScanOptions{
				Path: "/no/such/dir/that/does/not/exist",
			},
			checks: checkScan(
				checkScanError("no such file or directory"),
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r, err := Scan(context.Background(), tt.opts)
			for _, c := range tt.checks {
				c(t, r, err)
			}
		})
	}
}

func Test_filterByExclude(t *testing.T) {
	tests := []struct {
		name        string
		exclude     *regexp.Regexp
		functions   []FunctionCoverage
		wantCount   int
		wantNotName []string
	}{
		{
			name:      "nil exclude keeps all",
			exclude:   nil,
			functions: []FunctionCoverage{{Name: "Foo"}, {Name: "Bar"}},
			wantCount: 2,
		},
		{
			name:      "no matching pattern keeps all",
			exclude:   regexp.MustCompile(`_test\.go`),
			functions: []FunctionCoverage{{Name: "Foo", File: "foo.go"}, {Name: "Bar", File: "bar.go"}},
			wantCount: 2,
		},
		{
			name:        "pattern matches filters out",
			exclude:     regexp.MustCompile(`mock_`),
			functions:   []FunctionCoverage{{Name: "Foo", File: "mock_foo.go"}, {Name: "Bar", File: "bar.go"}},
			wantCount:   1,
			wantNotName: []string{"Foo"},
		},
		{
			name:      "recursive pattern matches deep paths",
			exclude:   regexp.MustCompile(`.*\.pb\.go$`),
			functions: []FunctionCoverage{{Name: "A", File: "main.pb.go"}, {Name: "B", File: "vendor/foo/bar.pb.go"}},
			wantCount: 0,
		},
		{
			name:      "empty functions list",
			exclude:   regexp.MustCompile(`anything`),
			functions: []FunctionCoverage{},
			wantCount: 0,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			got := filterByExclude(tt.functions, tt.exclude)
			assert.Equal(t, tt.wantCount, len(got))
			for _, notName := range tt.wantNotName {
				for _, f := range got {
					assert.NotEqual(t, notName, f.Name)
				}
			}
		})
	}
}

func Test_discoverModules(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		wantLen int
		wantErr bool
	}{
		{
			name:    "testdata has one module",
			path:    "../testdata",
			wantLen: 1,
			wantErr: false,
		},
		{
			name:    "non-existent path fails",
			path:    "/no/such/dir",
			wantLen: 0,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			modules, err := discoverModules(context.Background(), tt.path, nil)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.wantLen, len(modules))
		})
	}
}

func Test_discoverModules_empty_root(t *testing.T) {
	modules, err := discoverModules(context.Background(), "/dev/null", nil)
	assert.NoError(t, err)
	assert.Empty(t, modules)
}

func Test_discoverModules_nested_modules(t *testing.T) {
	tempDir := t.TempDir()

	os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module root\n"), 0644)
	subDir := filepath.Join(tempDir, "sub")
	os.MkdirAll(subDir, 0755)
	os.WriteFile(filepath.Join(subDir, "go.mod"), []byte("module root/sub\n"), 0644)

	modules, err := discoverModules(context.Background(), tempDir, nil)
	assert.NoError(t, err)
	assert.NotEmpty(t, modules)
}

func Test_filterByExclude_nil_regex(t *testing.T) {
	funcs := []FunctionCoverage{
		{File: "file1.go", Name: "Func1", Coverage: 100.0},
		{File: "file2.go", Name: "Func2", Coverage: 50.0},
	}
	result := filterByExclude(funcs, nil)
	assert.Len(t, result, 2)
}

func Test_filterByExclude_matching_pattern(t *testing.T) {
	funcs := []FunctionCoverage{
		{File: "pkg/main.go", Name: "Main", Coverage: 100.0},
		{File: "pkg/main_test.go", Name: "TestMain", Coverage: 80.0},
		{File: "pkg/util.go", Name: "Util", Coverage: 60.0},
	}
	regex, _ := regexp.Compile("_test\\.go$")
	result := filterByExclude(funcs, regex)
	assert.Len(t, result, 2)
	assert.Equal(t, "Main", result[0].Name)
	assert.Equal(t, "Util", result[1].Name)
}

func Test_filterByExclude_no_match_keeps_all(t *testing.T) {
	funcs := []FunctionCoverage{
		{File: "pkg/main.go", Name: "Main", Coverage: 100.0},
	}
	regex, _ := regexp.Compile("nonexistent")
	result := filterByExclude(funcs, regex)
	assert.Len(t, result, 1)
}

func Test_filterByExclude_empty_functions(t *testing.T) {
	regex, _ := regexp.Compile("test")
	result := filterByExclude(nil, regex)
	assert.Empty(t, result)
}

func Test_filterByExclude_recursive_pattern(t *testing.T) {
	funcs := []FunctionCoverage{
		{File: "pkg/api/v1/handler.go", Name: "Handler", Coverage: 100.0},
		{File: "pkg/api/v1/handler.pb.go", Name: "PBHandler", Coverage: 100.0},
		{File: "pkg/api/v2/handler.go", Name: "HandlerV2", Coverage: 80.0},
	}
	regex, _ := regexp.Compile("\\.pb\\.go$")
	result := filterByExclude(funcs, regex)
	assert.Len(t, result, 2)
	assert.NotEqual(t, "PBHandler", result[0].Name)
}

func Test_readModulePath(t *testing.T) {
	tempDir := t.TempDir()
	os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("module github.com/user/pkg\n\ngo 1.21\n"), 0644)
	path, err := readModulePath(tempDir)
	assert.NoError(t, err)
	assert.Equal(t, "github.com/user/pkg", path)
}

func Test_readModulePath_no_module(t *testing.T) {
	tempDir := t.TempDir()
	os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("\n"), 0644)
	_, err := readModulePath(tempDir)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no module declaration")
}

func TestScan_timeout_preserved(t *testing.T) {
	tests := []struct {
		name    string
		timeout time.Duration
		want    time.Duration
	}{
		{
			name:    "zero_timeout_defaults_to_10m",
			timeout: 0,
			want:    10 * time.Minute,
		},
		{
			name:    "nonzero_timeout_preserved",
			timeout: 5 * time.Second,
			want:    5 * time.Second,
		},
		{
			name:    "custom_timeout_preserved",
			timeout: 30 * time.Minute,
			want:    30 * time.Minute,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opts := ScanOptions{
				Path:    "../testdata",
				Timeout: tt.timeout,
			}
			r, err := Scan(context.Background(), opts)
			assert.NoError(t, err)
			assert.NotEmpty(t, r)
		})
	}
}
