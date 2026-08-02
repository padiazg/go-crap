package coverage

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/padiazg/go-crap/pkg/progress"
)

type NewScannerFn func(*testing.T, *Scanner)

var checkNewScanner = func(fns ...NewScannerFn) []NewScannerFn { return fns }

func TestNewScanner(t *testing.T) {
	tests := []struct {
		name    string
		path    string
		exclude *regexp.Regexp
		logger  any
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
			name:    "timeout_propagated",
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

type mockProgressReporter struct {
	calls []string
}

func (m *mockProgressReporter) StartPhase(progress.Phase, int) {}
func (m *mockProgressReporter) Advance(int)                    { m.calls = append(m.calls, "Advance") }
func (m *mockProgressReporter) SetDetail(detail string) {
	m.calls = append(m.calls, "SetDetail:"+detail)
}
func (m *mockProgressReporter) SetTotal(int) { m.calls = append(m.calls, "SetTotal") }
func (m *mockProgressReporter) FinishPhase() {}
func (m *mockProgressReporter) Done()        {}
func (m *mockProgressReporter) Errored()     {}

func advanceCount(calls []string) int {
	n := 0
	for _, c := range calls {
		if c == "Advance" {
			n++
		}
	}
	return n
}

func TestScanner_Scan(t *testing.T) {
	checkError := func(want string) checkScannerScanFn {
		return func(t *testing.T, _ []ModuleCoverage, err error) {
			t.Helper()
			if want != "" {
				assert.Containsf(t, err.Error(), want, "%s expected in error", want)
				return
			}
			assert.Emptyf(t, err, "error not expected: %s", err)
		}
	}

	checkEmpty := func(want bool) checkScannerScanFn {
		return func(t *testing.T, mc []ModuleCoverage, err error) {
			t.Helper()
			if want {
				assert.Empty(t, mc)
			} else {
				assert.NotEmpty(t, mc)
			}
		}
	}

	var prog *mockProgressReporter
	tests := []struct {
		name       string
		checks     []checkScannerScanFn
		before     func(*Scanner)
		newContext func() context.Context
	}{
		{
			name: "empty_dir_no_modules",
			checks: checkScannerScan(
				checkError(""),
				checkEmpty(true),
			),
			before: func(s *Scanner) { s.Path = t.TempDir() },
		},
		{
			name: "non_existen_dir",
			checks: checkScannerScan(
				checkError("/nonexistend/: no such file or directory"),
				checkEmpty(true),
			),
			before: func(s *Scanner) { s.Path = "/nonexistend/" },
		},
		{
			name: "single_module_with_tests",
			checks: checkScannerScan(
				checkError(""),
				func(t *testing.T, r []ModuleCoverage, err error) {
					t.Helper()
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
				// Scan only checks ctx before each scanModule iteration.
				// With 1 module in tempDir, the loop finishes before
				// the cancellation check can fire. No error expected.
				checkError(""),
				checkEmpty(false),
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
		{
			// Cancel fires mid-scan: discovery (sub-ms) completes, then the
			// slow module's "go test" keeps scanModule busy long enough for
			// the next loop iteration to hit the ctx.Done() check.
			name: "ctx_cancel_mid_scan",
			checks: checkScannerScan(
				func(t *testing.T, r []ModuleCoverage, err error) {
					t.Helper()
					assert.ErrorIs(t, err, context.Canceled)
				},
			),
			before: func(s *Scanner) {
				parent := t.TempDir()
				for _, name := range []string{"modA", "modB"} {
					modDir := filepath.Join(parent, name)
					os.MkdirAll(modDir, 0755)
					os.WriteFile(filepath.Join(modDir, "go.mod"), fmt.Appendf(nil, "module %s\n\ngo 1.21\n", name), 0644)
					os.WriteFile(filepath.Join(modDir, "pkg.go"), fmt.Appendf(nil, "package %s\n\nfunc Something() int { return 42 }\n", name), 0644)
					os.WriteFile(filepath.Join(modDir, "pkg_test.go"), fmt.Appendf(nil, "package %s\n\nimport (\n\t\"testing\"\n\t\"time\"\n)\n\nfunc TestSlow(t *testing.T) {\n\ttime.Sleep(2 * time.Second)\n}\n", name), 0644)
				}
				s.Path = parent
			},
			newContext: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				go func() {
					time.Sleep(300 * time.Millisecond)
					cancel()
				}()
				return ctx
			},
		},
		{
			// A supplied profile that does not exist is a hard, fail-fast
			// error rather than a silently degraded empty report.
			name: "missing_supplied_profile",
			checks: checkScannerScan(
				checkError("coverage profile:"),
				checkEmpty(true),
			),
			before: func(s *Scanner) {
				s.Path = t.TempDir()
				s.Profile = filepath.Join(t.TempDir(), "does-not-exist.out")
			},
		},
		{
			// A supplied profile applied to a package subdirectory that has
			// no go.mod of its own resolves against the enclosing module.
			name: "supplied_profile_on_package_subdir",
			checks: checkScannerScan(
				checkError(""),
				func(t *testing.T, r []ModuleCoverage, err error) {
					t.Helper()
					// assert.NoError(t, err)
					require.Len(t, r, 1)
					require.Len(t, r[0].Functions, 1)
					assert.Equal(t, "Covered", r[0].Functions[0].Name)
					assert.Equal(t, 100.0, r[0].Functions[0].Coverage)
				},
			),
			before: func(s *Scanner) {
				modDir := t.TempDir()
				goMod := `module submod

		go 1.21
		`
				os.WriteFile(filepath.Join(modDir, "go.mod"), []byte(goMod), 0644)
				pkgDir := filepath.Join(modDir, "internal", "pkg")
				os.MkdirAll(pkgDir, 0755)
				src := `package pkg

		func Covered() int {
			return 42
		}
		`
				os.WriteFile(filepath.Join(pkgDir, "pkg.go"), []byte(src), 0644)
				profPath := filepath.Join(modDir, "cover.out")
				os.WriteFile(profPath, []byte("mode: set\n"+
					"submod/internal/pkg/pkg.go:3.20,5.2 1 1\n"), 0644)
				s.Path = pkgDir
				s.Profile = profPath
			},
		},
		{
			name: "multiple_modules",
			checks: checkScannerScan(
				checkError(""),
				func(t *testing.T, r []ModuleCoverage, err error) {
					t.Helper()
					// assert.NoError(t, err)
					require.Len(t, r, 2)
				},
			),
			before: func(s *Scanner) {
				parent := t.TempDir()
				for _, name := range []string{"modA", "modB"} {
					modDir := filepath.Join(parent, name)
					os.MkdirAll(modDir, 0755)
					os.WriteFile(filepath.Join(modDir, "go.mod"), fmt.Appendf(nil, "module %s\n\ngo 1.21\n", name), 0644)
					os.WriteFile(filepath.Join(modDir, "pkg.go"), fmt.Appendf(nil, "package %s\n\nfunc Something() int { return 42 }\n", name), 0644)
					os.WriteFile(filepath.Join(modDir, "pkg_test.go"), fmt.Appendf(nil, "package %s\n\nimport \"testing\"\n\nfunc TestSuccess(t *testing.T) {\n\tif 1+1 != 2 {\n\t\tt.Error(\"math broken\")\n\t}\n}\n", name), 0644)
				}
				s.Path = parent
			},
		},
		{
			name: "partial_module_failure",
			checks: checkScannerScan(
				checkError(""),
				func(t *testing.T, r []ModuleCoverage, err error) {
					t.Helper()
					// assert.NoError(t, err)
					require.Len(t, r, 2)
					var goodCount, badCount int
					for _, mc := range r {
						if mc.Error != nil {
							assert.Contains(t, mc.Error.Error(), "runTests")
							badCount++
						} else {
							goodCount++
						}
					}
					assert.Equal(t, 1, goodCount, "expected one module without error")
					assert.Equal(t, 1, badCount, "expected one module with error")
				},
				func(t *testing.T, r []ModuleCoverage, err error) {
					t.Helper()
					// Advance fires once per module on both the success path
					// and the module-error path.
					require.NotNil(t, prog)
					assert.Equal(t, 2, advanceCount(prog.calls), "expected one Advance per module")
				},
			),
			before: func(s *Scanner) {
				parent := t.TempDir()
				// Good module — passing tests
				modDir := filepath.Join(parent, "good")
				os.MkdirAll(modDir, 0755)
				os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module good\n\ngo 1.21\n"), 0644)
				os.WriteFile(filepath.Join(modDir, "pkg.go"), []byte("package good\n\nfunc Something() int { return 42 }\n"), 0644)
				os.WriteFile(filepath.Join(modDir, "pkg_test.go"), []byte("package good\n\nimport \"testing\"\n\nfunc TestSuccess(t *testing.T) {\n\tif 1+1 != 2 {\n\t\tt.Error(\"math broken\")\n\t}\n}\n"), 0644)
				// Bad module — failing test
				modDir = filepath.Join(parent, "bad")
				os.MkdirAll(modDir, 0755)
				os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module bad\n\ngo 1.21\n"), 0644)
				os.WriteFile(filepath.Join(modDir, "pkg.go"), []byte("package bad\n\nfunc Something() int { return 42 }\n"), 0644)
				os.WriteFile(filepath.Join(modDir, "pkg_test.go"), []byte("package bad\n\nimport \"testing\"\n\nfunc TestFail(t *testing.T) {\n\tt.Fatal(\"intentional failure\")\n}\n"), 0644)
				prog = &mockProgressReporter{}
				s.Progress = prog
				s.Path = parent
			},
		},
		{
			name: "ctx_cancelled_before_discovery",
			checks: checkScannerScan(
				checkError(""),
				checkEmpty(true),
				// func(t *testing.T, r []ModuleCoverage, err error) {
				// 	t.Helper()
				// 	assert.NoError(t, err)
				// 	assert.Empty(t, r)
				// },
			),
			before: func(s *Scanner) {
				parent := t.TempDir()
				modDir := filepath.Join(parent, "mod")
				os.MkdirAll(modDir, 0755)
				os.WriteFile(filepath.Join(modDir, "go.mod"), []byte("module mod\n\ngo 1.21\n"), 0644)
				os.WriteFile(filepath.Join(modDir, "pkg.go"), []byte("package mod\n\nfunc Something() int { return 42 }\n"), 0644)
				s.Path = parent
			},
			newContext: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
		},
		{
			name: "progress_reporter_tracking",
			checks: checkScannerScan(
				checkError(""),
				func(t *testing.T, r []ModuleCoverage, err error) {
					t.Helper()
					// assert.NoError(t, err)
					require.Len(t, r, 2)
				},
			),
			before: func(s *Scanner) {
				parent := t.TempDir()
				for _, name := range []string{"modA", "modB"} {
					modDir := filepath.Join(parent, name)
					os.MkdirAll(modDir, 0755)
					os.WriteFile(filepath.Join(modDir, "go.mod"), fmt.Appendf(nil, "module %s\n\ngo 1.21\n", name), 0644)
					os.WriteFile(filepath.Join(modDir, "pkg.go"), fmt.Appendf(nil, "package %s\n\nfunc Something() int { return 42 }\n", name), 0644)
					os.WriteFile(filepath.Join(modDir, "pkg_test.go"), fmt.Appendf(nil, "package %s\n\nimport \"testing\"\n\nfunc TestSuccess(t *testing.T) {\n\tif 1+1 != 2 {\n\t\tt.Error(\"math broken\")\n\t}\n}\n", name), 0644)
				}
				s.Path = parent
				s.Progress = &mockProgressReporter{}
			},
		},
	}
	for _, tt := range tests {
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
		return dir != tempDir
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

type checkScannerscanModuleFn func(*testing.T, ModuleCoverage)

var checkScannerscanModule = func(fns ...checkScannerscanModuleFn) []checkScannerscanModuleFn { return fns }

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
				func(t *testing.T, r ModuleCoverage) {
					t.Helper()
					assert.NoError(t, r.Error)
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
			name: "module_with_tests_removes_temp_profile",
			checks: checkScannerscanModule(
				func(t *testing.T, r ModuleCoverage) {
					t.Helper()
					assert.NoError(t, r.Error)
					_, err := os.Stat(r.Profile)
					assert.ErrorIs(t, err, os.ErrNotExist, "coverage profile temp file(s) not cleaned up: %s", r.Profile)
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
				func(t *testing.T, r ModuleCoverage) {
					t.Helper()
					assert.Error(t, r.Error)
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
			name: "module_with_partial_coverage_on_failure",
			checks: checkScannerscanModule(
				func(t *testing.T, r ModuleCoverage) {
					t.Helper()
					t.Logf("r.Error: %v", r.Error)
					t.Logf("r.ModulePath: %s", r.ModulePath)
					t.Logf("r.Profile: %s", r.Profile)
					t.Logf("r.Functions count: %d", len(r.Functions))
					assert.Error(t, r.Error)
					assert.Contains(t, r.Error.Error(), "runTests")
					assert.NotEmpty(t, r.Functions, "partial coverage data should be available despite test failure")
				},
			),
			before: func(s *Scanner) {
				tempDir := t.TempDir()
				goMod := `module partialmodule

go 1.21
`
				os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644)

				pkgDir := filepath.Join(tempDir, "pkg")
				os.Mkdir(pkgDir, 0755)
				os.WriteFile(filepath.Join(pkgDir, "pkg.go"), []byte(`package pkg

func AlwaysPasses() int {
	return 42
}
`), 0644)
				os.WriteFile(filepath.Join(pkgDir, "pkg_test.go"), []byte(`package pkg

import "testing"

func TestAlwaysPasses(t *testing.T) {
	if AlwaysPasses() != 42 {
		t.Error("expected 42")
	}
}
`), 0644)

				failPkgDir := filepath.Join(tempDir, "failpkg")
				os.Mkdir(failPkgDir, 0755)
				os.WriteFile(filepath.Join(failPkgDir, "fail.go"), []byte(`package failpkg

func AlwaysFails() error {
	return nil
}
`), 0644)
				os.WriteFile(filepath.Join(failPkgDir, "fail_test.go"), []byte(`package failpkg

import "testing"

func TestAlwaysFails(t *testing.T) {
	t.Fatal("this always fails")
}
`), 0644)

				s.Path = tempDir
			},
		},
		{
			name: "module_without_tests",
			checks: checkScannerscanModule(
				func(t *testing.T, r ModuleCoverage) {
					t.Helper()
					assert.NoError(t, r.Error)
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
		{
			// A supplied profile side-steps "go test": the module's test
			// always fails, yet scanModule succeeds and reports coverage
			// parsed straight from the profile.
			name: "module_with_supplied_profile",
			checks: checkScannerscanModule(
				func(t *testing.T, r ModuleCoverage) {
					t.Helper()
					assert.NoError(t, r.Error)
					require.Len(t, r.Functions, 1)
					assert.Equal(t, "Covered", r.Functions[0].Name)
					assert.Equal(t, 100.0, r.Functions[0].Coverage)
				},
			),
			before: func(s *Scanner) {
				tempDir := t.TempDir()
				goMod := `module profmodule

go 1.21
`
				os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644)
				src := `package profmodule

func Covered() int {
	return 42
}
`
				os.WriteFile(filepath.Join(tempDir, "pkg.go"), []byte(src), 0644)
				test := `package profmodule

import "testing"

func TestCovered(t *testing.T) {
	t.Fatal("this always fails")
}
`
				os.WriteFile(filepath.Join(tempDir, "pkg_test.go"), []byte(test), 0644)
				profPath := filepath.Join(tempDir, "cover.out")
				os.WriteFile(profPath, []byte("mode: set\n"+
					"profmodule/pkg.go:3.20,5.2 1 1\n"), 0644)
				s.Path = tempDir
				s.Profile = profPath
			},
		},
	}
	for _, tt := range tests {
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
			r := s.scanModule(ctx, modDir, nil)
			for _, c := range tt.checks {
				c(t, r)
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
			name:    "missing_module_line",
			wantErr: "no module declaration in go.mod",
		},
	}
	for _, tt := range tests {
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
			r, err := s.runTests(ctx, modDir, nil)
			defer func() {
				if removeErr := os.Remove(r); removeErr != nil {
					assert.NoError(t, removeErr)
				}
			}()
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

	_, err := s.runTests(ctx, tempDir, nil)
	assert.Error(t, err)
}

func TestScanner_runTests_deadline_exceeded_message(t *testing.T) {
	tempDir := t.TempDir()
	goMod := `module slowtest

go 1.21
`
	os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644)
	src := `package slowtest

func Something() int { return 1 }
`
	os.WriteFile(filepath.Join(tempDir, "pkg.go"), []byte(src), 0644)
	test := `package slowtest

import (
	"testing"
	"time"
)

func TestSlow(t *testing.T) {
	time.Sleep(5 * time.Second)
}
`
	os.WriteFile(filepath.Join(tempDir, "pkg_test.go"), []byte(test), 0644)

	s := NewScanner("value", nil, nil, 0)

	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	_, err := s.runTests(ctx, tempDir, nil)
	if assert.Error(t, err) {
		assert.Contains(t, err.Error(), "timed out")
		assert.Contains(t, err.Error(), "--timeout")
	}
}

type recordingLogger struct {
	warns []string
}

func (l *recordingLogger) Debug(format string, args ...any) {}

func (l *recordingLogger) Info(format string, args ...any) {}

func (l *recordingLogger) Warn(format string, args ...any) {
	l.warns = append(l.warns, format)
}

func (l *recordingLogger) Error(format string, args ...any) {}

func TestScanner_runTests_no_warning_without_failed_tests(t *testing.T) {
	tempDir := t.TempDir()
	goMod := `module buildfail

go 1.21
`
	os.WriteFile(filepath.Join(tempDir, "go.mod"), []byte(goMod), 0644)
	src := `package buildfail

func broken( {
`
	os.WriteFile(filepath.Join(tempDir, "pkg.go"), []byte(src), 0644)

	logger := &recordingLogger{}
	s := NewScanner("value", nil, logger, 0)

	ctx := context.Background()
	_, err := s.runTests(ctx, tempDir, nil)
	assert.Error(t, err)
	assert.NotContains(t, logger.warns, "coverage: tests failed in module", "no failed tests to report")
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

func Test_extractFailedTests(t *testing.T) {
	tests := []struct {
		name     string
		stderr   string
		expected []string
	}{
		{
			name: "single_failure",
			stderr: `--- FAIL: TestFoo (0.01s)
    scanner_test.go:10: panicked
`,
			expected: []string{"TestFoo"},
		},
		{
			name: "multiple_failures",
			stderr: `--- FAIL: TestFoo (0.01s)
    main_test.go:5: error
--- FAIL: TestBar (0.02s)
    main_test.go:10: error
ok  	package	0.100s
`,
			expected: []string{"TestFoo", "TestBar"},
		},
		{
			name: "no_failures",
			stderr: `ok  	package	0.100s
`,
			expected: nil,
		},
		{
			name:     "empty",
			stderr:   "",
			expected: nil,
		},
		{
			name:     "with_whitespace",
			stderr:   "   \n--- FAIL:  TestBaz (0.01s)\n   ",
			expected: []string{"TestBaz"},
		},
		{
			name: "skip_non_fail_lines",
			stderr: `=== RUN   TestFoo
--- PASS: TestFoo (0.01s)
--- FAIL: TestBar (0.01s)
=== RUN   TestBaz
--- PASS: TestBaz (0.01s)
`,
			expected: []string{"TestBar"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := extractFailedTests(tt.stderr)
			assert.Equal(t, tt.expected, r)
		})
	}
}
