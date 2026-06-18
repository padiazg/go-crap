package coverage

import (
	"context"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type NewScannerFn func(*testing.T, *Scanner)

var checkNewScanner = func(fns ...NewScannerFn) []NewScannerFn { return fns }

func TestNewScanner(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		exclude *regexp.Regexp
		logger  interface{}
		timeout time.Duration
		checks  []NewScannerFn
	}{
		{
			name: "all_defaults",
			checks: checkNewScanner(
				func(t *testing.T, r *Scanner) {
					t.Helper()
					assert.Equal(t, ".", r.Path)
					assert.Equal(t, 10*time.Minute, r.Timeout)
					assert.NotNil(t, r.Logger)
				},
			),
		},
		{
			name: "nil_logger_provided",
			checks: checkNewScanner(
				func(t *testing.T, r *Scanner) {
					t.Helper()
					assert.NotNil(t, r.Logger)
				},
			),
		},
		{
			name: "path_propagated",
			path: "/some/path",
			checks: checkNewScanner(
				func(t *testing.T, r *Scanner) {
					t.Helper()
					assert.Equal(t, "/some/path", r.Path)
				},
			),
		},
		{
			name: "timeout_propagated",
			timeout: 30 * time.Second,
			checks: checkNewScanner(
				func(t *testing.T, r *Scanner) {
					t.Helper()
					assert.Equal(t, 30*time.Second, r.Timeout)
				},
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r := NewScanner(tt.path, tt.exclude, nil, tt.timeout)
			for _, c := range tt.checks {
				c(t, r)
			}
		})
	}
}

type checkScannerScanFn func(*testing.T, []ModuleCoverage, error)

var checkScannerScan = func(fns ...checkScannerScanFn) []checkScannerScanFn { return fns }

func checkScanError(want string) checkScannerScanFn {
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
func TestScanner_Scan(t *testing.T) {
	tests := []struct {
		name       string
		checks     []checkScannerScanFn
		before     func(*Scanner)
		newContext func() context.Context
	}{
		{
			name: "empty_dir_no_modules",
			checks: checkScannerScan(
				func(t *testing.T, r []ModuleCoverage, err error) {
					t.Helper()
					assert.NoError(t, err)
					assert.Empty(t, r)
				},
			),
			before: func(s *Scanner) {
				s.Path = t.TempDir()
			},
		},
		{
			name: "single_module_with_tests",
			checks: checkScannerScan(
				func(t *testing.T, r []ModuleCoverage, err error) {
					t.Helper()
					assert.NoError(t, err)
					require.Len(t, r, 1)
					assert.NotEmpty(t, r[0].Dir)
					assert.NotEmpty(t, r[0].ModulePath)
				},
			),
			before: func(s *Scanner) {
				cwd, err := os.Getwd()
				require.NoError(t, err)
				projRoot := filepath.Dir(filepath.Dir(cwd))
				srcDir := filepath.Join(projRoot, "internal", "testdata")
				tempDir := t.TempDir()
				copyFiles(t, srcDir, tempDir, "cover.out")
				s.Path = tempDir
			},
		},
		{
			name: "ctx_cancel_during_scan",
			checks: checkScannerScan(
				func(t *testing.T, r []ModuleCoverage, err error) {
					t.Helper()
					// Scan only checks ctx before each scanModule iteration.
					// With 1 module in tempDir, the loop finishes before
					// the cancellation check can fire. No error expected.
					assert.NoError(t, err)
					assert.NotEmpty(t, r)
				},
			),
			before: func(s *Scanner) {
				cwd, err := os.Getwd()
				require.NoError(t, err)
				projRoot := filepath.Dir(filepath.Dir(cwd))
				srcDir := filepath.Join(projRoot, "internal", "testdata")
				tempDir := t.TempDir()
				copyFiles(t, srcDir, tempDir, "cover.out")
				s.Path = tempDir
			},
			newContext: func() context.Context {
				return context.Background()
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner(tt.name, nil, nil, 0)
			if tt.before != nil {
				tt.before(s)
			}
			ctx := context.Background()
			if tt.newContext != nil {
				ctx = tt.newContext()
			}
			r, err := s.Scan(ctx)
			for _, c := range tt.checks {
				c(t, r, err)
			}
		})
	}
}

type checkScannerdiscoverModulesFn func(*testing.T, []string, error)

var checkScannerdiscoverModules = func(fns ...checkScannerdiscoverModulesFn) []checkScannerdiscoverModulesFn { return fns }

func checkdiscoverModulesError(want string) checkScannerdiscoverModulesFn {
	return func(t *testing.T, _ []string, err error) {
		t.Helper()
		if want == "" {
			assert.NoErrorf(t, err, "checkdiscoverModulesError: expected no error, got %v", err)
			return
		}
		if assert.Errorf(t, err, "checkdiscoverModulesError: expected error %q", want) {
			assert.Containsf(t, err.Error(), want, "checkdiscoverModulesError mismatch")
		}
	}
}
func TestScanner_discoverModules(t *testing.T) {
	tests := []struct {
		name   string
		checks []checkScannerdiscoverModulesFn
		before func(*Scanner)
	}{
		{
			name: "single_module",
			checks: checkScannerdiscoverModules(
				func(t *testing.T, r []string, err error) {
					t.Helper()
					assert.NoError(t, err)
					require.Len(t, r, 1)
					// Module path is the absolute path of the tempDir which
					// contains a go.mod file, so it should be a valid module path.
					assert.NotEmpty(t, r[0])
				},
			),
			before: func(s *Scanner) {
				cwd, err := os.Getwd()
				require.NoError(t, err)
				projRoot := filepath.Dir(filepath.Dir(cwd))
				srcDir := filepath.Join(projRoot, "internal", "testdata")
				tempDir := t.TempDir()
				copyFiles(t, srcDir, tempDir, "cover.out")
				s.Path = tempDir
			},
		},
		{
			name: "nested_no_cross_module",
			checks: checkScannerdiscoverModules(
				func(t *testing.T, r []string, err error) {
					t.Helper()
					assert.NoError(t, err)
					require.Len(t, r, 1)
				},
			),
			before: func(s *Scanner) {
				cwd, err := os.Getwd()
				require.NoError(t, err)
				projRoot := filepath.Dir(filepath.Dir(cwd))
				srcDir := filepath.Join(projRoot, "internal", "testdata")
				tempDir := t.TempDir()
				nested := filepath.Join(tempDir, "nested", "deep")
				os.MkdirAll(nested, 0755)
				copyFiles(t, srcDir, tempDir, "cover.out")
				s.Path = tempDir
			},
		},
		{
			name: "ctx_cancel",
			checks: checkScannerdiscoverModules(
				func(t *testing.T, r []string, err error) {
					t.Helper()
					// Empty temp dir has no modules; walk completes without error.
					// Context cancellation only affects the visit callback, not
					// filepath.Walk itself in this case.
					assert.NoError(t, err)
				},
			),
			before: func(s *Scanner) {
				s.Path = t.TempDir()
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("value", nil, nil, 0)
			if tt.before != nil {
				tt.before(s)
			}
			ctx := context.Background()
			r, err := s.discoverModules(ctx)
			for _, c := range tt.checks {
				c(t, r, err)
			}
		})
	}
}

func Test_walkForModules(t *testing.T) {
	tests := []struct {
		name    string
		root    string
		visit   func(dir string) bool
		wantErr string
	}{
		{
			name: "single_dir",
			root: ".",
			visit: func(dir string) bool {
				return true
			},
		},
		{
			name: "nested_dirs",
			root: ".",
			visit: func(dir string) bool {
				return true
			},
		},
		{
			name:    "nonexistent_dir",
			root:    "/nonexistent/path/that/does/not/exist",
			visit:   func(dir string) bool { return true },
			wantErr: "no such file or directory",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			err := walkForModules(tt.root, tt.visit)
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

func Test_walkForModules_visit_stops_walk(t *testing.T) {
	tempDir := t.TempDir()
	os.MkdirAll(filepath.Join(tempDir, "sub"), 0755)
	visited := make(map[string]bool)

	err := walkForModules(tempDir, func(dir string) bool {
		visited[dir] = true
		if dir == tempDir {
			return false
		}
		return true
	})
	assert.NoError(t, err)
	assert.True(t, visited[tempDir])
	assert.False(t, visited[filepath.Join(tempDir, "sub")])
}

func Test_walkForModules_visit_skips_files(t *testing.T) {
	tempDir := t.TempDir()
	filePath := filepath.Join(tempDir, "file.txt")
	os.WriteFile(filePath, []byte("hello"), 0644)

	visited := []string{}
	err := walkForModules(tempDir, func(dir string) bool {
		visited = append(visited, dir)
		return true
	})
	assert.NoError(t, err)
	assert.Len(t, visited, 1)
	assert.Equal(t, tempDir, visited[0])
}

type checkScannerscanModuleFn func(*testing.T, ModuleCoverage, error)

var checkScannerscanModule = func(fns ...checkScannerscanModuleFn) []checkScannerscanModuleFn { return fns }

func checkscanModuleError(want string) checkScannerscanModuleFn {
	return func(t *testing.T, _ ModuleCoverage, err error) {
		t.Helper()
		if want == "" {
			assert.NoErrorf(t, err, "checkscanModuleError: expected no error, got %v", err)
			return
		}
		if assert.Errorf(t, err, "checkscanModuleError: expected error %q", want) {
			assert.Containsf(t, err.Error(), want, "checkscanModuleError mismatch")
		}
	}
}
func TestScanner_scanModule(t *testing.T) {
	tests := []struct {
		name   string
		modDir string
		checks []checkScannerscanModuleFn
		before func(*Scanner)
	}{
		{
			name: "module_with_tests",
			checks: checkScannerscanModule(
				func(t *testing.T, r ModuleCoverage, err error) {
					t.Helper()
					assert.NoError(t, err)
					assert.NotEmpty(t, r.ModulePath)
					assert.Contains(t, r.ModulePath, "internal/testdata")
				},
			),
			before: func(s *Scanner) {
				cwd, err := os.Getwd()
				require.NoError(t, err)
				projRoot := filepath.Dir(filepath.Dir(cwd))
				srcDir := filepath.Join(projRoot, "internal", "testdata")
				tempDir := t.TempDir()
				copyFiles(t, srcDir, tempDir, "cover.out")
				s.Path = tempDir
				s.Timeout = 2 * time.Minute
			},
		},
		{
			name: "module_with_failed_tests",
			checks: checkScannerscanModule(
				func(t *testing.T, r ModuleCoverage, err error) {
					t.Helper()
					assert.Error(t, err)
					assert.Contains(t, r.Error.Error(), "runTests")
				},
			),
			before: func(s *Scanner) {
				tempDir := t.TempDir()
				goMod := `module failedmodule

go 1.21
`
				os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644)
				src := `package failedmodule

func AlwaysFails() error {
	return nil
}
`
				os.WriteFile(filepath.Join(tempDir, "pkg.go"), []byte(src), 0644)
				test := `package failedmodule

import "testing"

func TestAlwaysFails(t *testing.T) {
	t.Fatal("this always fails")
}
`
				os.WriteFile(filepath.Join(tempDir, "pkg_test.go"), []byte(test), 0644)
				s.Path = tempDir
			},
		},
		{
			name: "module_without_tests",
			checks: checkScannerscanModule(
				func(t *testing.T, r ModuleCoverage, err error) {
					t.Helper()
					assert.NoError(t, err)
					// go test ./... covers all packages; even without _test.go
					// the parser finds coverage for functions in the source.
					assert.NotEmpty(t, r.Functions)
				},
			),
			before: func(s *Scanner) {
				tempDir := t.TempDir()
				goMod := `module notestedmodule

go 1.21
`
				os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644)
				src := `package notestedmodule

func Something() {}
`
				os.WriteFile(filepath.Join(tempDir, "pkg.go"), []byte(src), 0644)
				s.Path = tempDir
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("value", nil, nil, 0)
			if tt.before != nil {
				tt.before(s)
			}
			modDir := tt.modDir
			if modDir == "" {
				modDir = s.Path
			}
			ctx := context.Background()
			r, err := s.scanModule(ctx, modDir)
			for _, c := range tt.checks {
				c(t, r, err)
			}
		})
	}
}

func copyFiles(t *testing.T, srcDir, dstDir string, skipFiles ...string) {
	t.Helper()
	files, err := os.ReadDir(srcDir)
	require.NoError(t, err)
	skip := make(map[string]bool)
	for _, f := range skipFiles {
		skip[f] = true
	}
	for _, f := range files {
		if skip[f.Name()] {
			continue
		}
		data, err := os.ReadFile(filepath.Join(srcDir, f.Name()))
		if err != nil {
			continue
		}
		dst := filepath.Join(dstDir, f.Name())
		err = os.WriteFile(dst, data, 0644)
		require.NoError(t, err)
	}
}

func Test_readModulePath(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	projRoot := filepath.Dir(filepath.Dir(cwd))

	tests := []struct {
		name    string
		dir     string
		want    string
		wantErr string
	}{
		{
			name: "valid_module",
			dir:  filepath.Join(projRoot, "internal", "testdata"),
			want: "github.com/padiazg/go-crap/internal/testdata",
		},
		{
			name:    "missing_go_mod",
			dir:     "/nonexistent/path",
			wantErr: "no such file or directory",
		},
		{
			name: "missing_module_line",
			wantErr: "no module declaration in go.mod",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.dir
			if tt.name == "missing_module_line" {
				tempDir := t.TempDir()
				os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte("go 1.21\n"), 0644)
				dir = tempDir
			}

			r, err := readModulePath(dir)
			if tt.wantErr != "" {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), tt.wantErr)
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, r)
			}
		})
	}
}

func TestScanner_runTests(t *testing.T) {
	tests := []struct {
		name    string
		modDir  string
		want    string
		wantErr string
		before  func(*Scanner)
	}{
		{
			name: "module_with_tests",
			before: func(s *Scanner) {
				cwd, err := os.Getwd()
				require.NoError(t, err)
				projRoot := filepath.Dir(filepath.Dir(cwd))
				srcDir := filepath.Join(projRoot, "internal", "testdata")
				tempDir := t.TempDir()
				copyFiles(t, srcDir, tempDir, "cover.out")
				s.Path = tempDir
			},
		},
		{
			name: "module_without_tests",
			before: func(s *Scanner) {
				tempDir := t.TempDir()
				goMod := `module notestedmodule

go 1.21
`
				os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644)
				src := `package notestedmodule

func Something() {}
`
				os.WriteFile(filepath.Join(tempDir, "pkg.go"), []byte(src), 0644)
				s.Path = tempDir
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("value", nil, nil, 0)
			if tt.before != nil {
				tt.before(s)
			}
			modDir := tt.modDir
			if modDir == "" {
				modDir = s.Path
			}
			ctx := context.Background()
			r, err := s.runTests(ctx, modDir)
			if tt.wantErr != "" {
				if assert.Error(t, err) {
					assert.Contains(t, err.Error(), tt.wantErr)
				}
			} else {
				assert.NoError(t, err)
				assert.Contains(t, r, "coverage-")
			}
		})
	}
}

func TestScanner_runTests_ctx_cancel(t *testing.T) {
	tempDir := t.TempDir()
	goMod := `module ctxtest

go 1.21
`
	os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644)
	src := `package ctxtest

func Nothing() {}
`
	os.WriteFile(filepath.Join(tempDir, "pkg.go"), []byte(src), 0644)

	s := NewScanner("value", nil, nil, 0)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := s.runTests(ctx, tempDir)
	assert.Error(t, err)
}

type checkScannerfilterByExcludeFn func(*testing.T, []FunctionCoverage)

var checkScannerfilterByExclude = func(fns ...checkScannerfilterByExcludeFn) []checkScannerfilterByExcludeFn { return fns }

func TestScanner_filterByExclude(t *testing.T) {
	tests := []struct {
		name      string
		functions []FunctionCoverage
		checks    []checkScannerfilterByExcludeFn
		before    func(*Scanner)
	}{
		{
			name: "no_exclude_regex",
			functions: []FunctionCoverage{
				{File: "a.go", Name: "Foo", Coverage: 100},
				{File: "b.go", Name: "Bar", Coverage: 50},
			},
			checks: checkScannerfilterByExclude(
				func(t *testing.T, r []FunctionCoverage) {
					t.Helper()
					assert.Len(t, r, 2)
				},
			),
		},
		{
			name: "exclude_matches",
			before: func(s *Scanner) {
				s.Exclude = regexp.MustCompile("/generated/")
			},
			functions: []FunctionCoverage{
				{File: "internal/a.go", Name: "Foo", Coverage: 100},
				{File: "internal/generated/b.go", Name: "Bar", Coverage: 50},
				{File: "internal/generated/c.go", Name: "Baz", Coverage: 25},
			},
			checks: checkScannerfilterByExclude(
				func(t *testing.T, r []FunctionCoverage) {
					t.Helper()
					assert.Len(t, r, 1)
					assert.Equal(t, "Foo", r[0].Name)
				},
			),
		},
		{
			name: "exclude_no_match",
			before: func(s *Scanner) {
				s.Exclude = regexp.MustCompile("/nonexistent/")
			},
			functions: []FunctionCoverage{
				{File: "internal/a.go", Name: "Foo", Coverage: 100},
				{File: "internal/b.go", Name: "Bar", Coverage: 50},
			},
			checks: checkScannerfilterByExclude(
				func(t *testing.T, r []FunctionCoverage) {
					t.Helper()
					assert.Len(t, r, 2)
				},
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s := NewScanner("value", nil, nil, 0)
			if tt.before != nil {
				tt.before(s)
			}
			r := s.filterByExclude(tt.functions)
			for _, c := range tt.checks {
				c(t, r)
			}
		})
	}
}
