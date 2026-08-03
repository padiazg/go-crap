package packages

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type getTargetsFn func(*testing.T, []Target, []string, error)

// fakeGoListRunner is a test double for GoListRunner.
type fakeGoListRunner struct {
	Output   []byte
	Err      error
	Patterns []string
}

func (f *fakeGoListRunner) List(ctx context.Context, patterns []string) ([]byte, error) {
	f.Patterns = patterns
	return f.Output, f.Err
}

var checkgetTargets = func(fns ...getTargetsFn) []getTargetsFn { return fns }

func checkgetTargetsError(want string) getTargetsFn {
	return func(t *testing.T, _ []Target, _ []string, err error) {
		t.Helper()
		if want == "" {
			assert.NoErrorf(t, err, "checkgetTargetsError: expected no error, got %v", err)
			return
		}
		if assert.Errorf(t, err, "checkgetTargetsError: expected error %q", want) {
			assert.Containsf(t, err.Error(), want, "checkgetTargetsError mismatch")
		}
	}
}

func Test_NewGoListRunner(t *testing.T) {
	r := NewGoListRunner()
	assert.NotNil(t, r)
	_, ok := r.(*goListRunner)
	assert.True(t, ok, "NewGoListRunner should return *goListRunner")

	r2 := NewGoListRunnerWithDir("/custom/dir")
	gr, ok := r2.(*goListRunner)
	assert.True(t, ok)
	assert.Equal(t, "/custom/dir", gr.dir)
}

func Test_WithEnv(t *testing.T) {
	r := WithEnv([]string{"GO111MODULE=off", "FOO=bar"}).(*goListRunner)
	assert.Equal(t, []string{"GO111MODULE=off", "FOO=bar"}, r.env)
}

func Test_getTargets(t *testing.T) {
	marshalJSON := func(v any) string {
		b, _ := json.Marshal(v)
		return string(b)
	}

	pkgJSON := func(imp, dir, modPath, modDir string, goFiles, testGoFiles, xtestGoFiles []string, errStr string) string {
		var errJSON string
		if errStr == "" {
			errJSON = "null"
		} else {
			b, _ := json.Marshal(packageError{Err: errStr})
			errJSON = string(b)
		}
		return fmt.Sprintf(`{"ImportPath":%q,"Dir":%q,"ModulePath":%q,"ModuleDir":%q,"GoFiles":%s,"TestGoFiles":%s,"XTestGoFiles":%s,"Error":%s}`+"\n",
			imp, dir, modPath, modDir,
			marshalJSON(goFiles), marshalJSON(testGoFiles), marshalJSON(xtestGoFiles), errJSON)
	}

	tests := []struct {
		name         string
		stdout       *bytes.Buffer
		includeTests bool
		checks       []getTargetsFn
	}{
		{
			name:   "empty_output",
			stdout: bytes.NewBuffer(nil),
			checks: checkgetTargets(
				func(t *testing.T, targets []Target, errors []string, err error) {
					t.Helper()
					assert.NoError(t, err)
					assert.Empty(t, targets)
					assert.Empty(t, errors)
				},
			),
		},
		{
			name:   "single_valid_package_no_test_files",
			stdout: bytes.NewBufferString(pkgJSON("github.com/padiazg/go-crap/internal/packages", "/go/src/github.com/padiazg/go-crap/internal/packages", "github.com/padiazg/go-crap", "/go/src/github.com/padiazg/go-crap", []string{"packages.go"}, nil, nil, "")),
			checks: checkgetTargets(
				func(t *testing.T, targets []Target, errors []string, err error) {
					t.Helper()
					assert.NoError(t, err)
					require.Len(t, targets, 1)
					assert.Equal(t, "github.com/padiazg/go-crap/internal/packages", targets[0].ImportPath)
					assert.Equal(t, "/go/src/github.com/padiazg/go-crap/internal/packages", targets[0].Dir)
					require.Len(t, targets[0].Files, 1)
					assert.Contains(t, targets[0].Files[0], "internal/packages/packages.go")
					assert.Empty(t, targets[0].TestFiles)
				},
				checkgetTargetsError(""),
			),
		},
		{
			name:         "single_valid_package_with_test_files",
			stdout:       bytes.NewBufferString(pkgJSON("github.com/padiazg/go-crap/internal/packages", "/go/src/github.com/padiazg/go-crap/internal/packages", "github.com/padiazg/go-crap", "/go/src/github.com/padiazg/go-crap", []string{"packages.go"}, []string{"packages_test.go"}, []string{"packages_extra_test.go"}, "")),
			includeTests: true,
			checks: checkgetTargets(
				func(t *testing.T, targets []Target, errors []string, err error) {
					t.Helper()
					assert.NoError(t, err)
					require.Len(t, targets, 1)
					require.Len(t, targets[0].TestFiles, 2)
					assert.Contains(t, targets[0].TestFiles[0], "packages_test.go")
					assert.Contains(t, targets[0].TestFiles[1], "packages_extra_test.go")
				},
				checkgetTargetsError(""),
			),
		},
		{
			name:   "single_package_with_error",
			stdout: bytes.NewBufferString(pkgJSON("github.com/padiazg/go-crap/internal/bad", "/go/src/github.com/padiazg/go-crap/internal/bad", "github.com/padiazg/go-crap", "/go/src/github.com/padiazg/go-crap", nil, nil, nil, "cannot find module and listed as ./bad")),
			checks: checkgetTargets(
				func(t *testing.T, targets []Target, errors []string, err error) {
					t.Helper()
					assert.NoError(t, err)
					assert.Empty(t, targets)
					require.Len(t, errors, 1)
					assert.Contains(t, errors[0], "cannot find module")
				},
			),
		},
		{
			name: "multiple_packages_mixed",
			stdout: func() *bytes.Buffer {
				var b bytes.Buffer
				b.WriteString(pkgJSON("github.com/padiazg/go-crap/internal/packages", "/go/src/github.com/padiazg/go-crap/internal/packages", "github.com/padiazg/go-crap", "/go/src/github.com/padiazg/go-crap", []string{"packages.go"}, nil, nil, ""))
				b.WriteString(pkgJSON("github.com/padiazg/go-crap/internal/bad", "/go/src/github.com/padiazg/go-crap/internal/bad", "github.com/padiazg/go-crap", "/go/src/github.com/padiazg/go-crap", nil, nil, nil, "build error"))
				b.WriteString(pkgJSON("github.com/padiazg/go-crap/internal/score", "/go/src/github.com/padiazg/go-crap/internal/score", "github.com/padiazg/go-crap", "/go/src/github.com/padiazg/go-crap", []string{"score.go"}, nil, nil, ""))
				return &b
			}(),
			checks: checkgetTargets(
				func(t *testing.T, targets []Target, errors []string, err error) {
					t.Helper()
					assert.NoError(t, err)
					require.Len(t, targets, 2)
					assert.Equal(t, "github.com/padiazg/go-crap/internal/packages", targets[0].ImportPath)
					assert.Equal(t, "github.com/padiazg/go-crap/internal/score", targets[1].ImportPath)
					require.Len(t, errors, 1)
					assert.Contains(t, errors[0], "build error")
				},
				checkgetTargetsError(""),
			),
		},
		{
			name:   "malformed_json",
			stdout: bytes.NewBufferString(`{"ImportPath":`),
			checks: checkgetTargets(
				checkgetTargetsError("decode JSON"),
			),
		},
		{
			name:   "null_fields",
			stdout: bytes.NewBufferString(`{"ImportPath":"example.com/pkg","Dir":"/go/src/example.com/pkg","GoFiles":null,"TestGoFiles":null,"XTestGoFiles":null}` + "\n"),
			checks: checkgetTargets(
				func(t *testing.T, targets []Target, errors []string, err error) {
					t.Helper()
					assert.NoError(t, err)
					require.Len(t, targets, 1)
					assert.Equal(t, "example.com/pkg", targets[0].ImportPath)
					assert.Nil(t, targets[0].Files)
					assert.Empty(t, targets[0].TestFiles)
				},
				checkgetTargetsError(""),
			),
		},
		{
			name:   "empty_import_path",
			stdout: bytes.NewBufferString(`{"ImportPath":"","Dir":"","GoFiles":null}` + "\n"),
			checks: checkgetTargets(
				func(t *testing.T, targets []Target, errors []string, err error) {
					t.Helper()
					assert.NoError(t, err)
					require.Len(t, targets, 1)
					assert.Empty(t, targets[0].ImportPath)
				},
				checkgetTargetsError(""),
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r, r2, err := getTargets(tt.stdout, tt.includeTests)
			for _, c := range tt.checks {
				c(t, r, r2, err)
			}
		})
	}
}

type NewResolverFn func(*testing.T, *Resolver)

var checkNewResolver = func(fns ...NewResolverFn) []NewResolverFn { return fns }

func TestNewResolver(t *testing.T) {
	sampleJSON := `{"ImportPath":"example.com/pkg","Dir":"/go/src/example.com/pkg","GoFiles":["a.go"]}` + "\n"

	fakeRunner := &fakeGoListRunner{Output: []byte(sampleJSON)}
	dirRunner := NewGoListRunnerWithDir("/custom/dir")

	checkDefaults := func(t *testing.T, r *Resolver) {
		t.Helper()
		require.NotNil(t, r)
		require.IsType(t, &goListRunner{}, r.runner)
		gr := r.runner.(*goListRunner)
		assert.Equal(t, ".", gr.dir)
		assert.Contains(t, gr.env, "GO111MODULE=on")
	}

	checkSame := func(want GoListRunner) NewResolverFn {
		return func(t *testing.T, r *Resolver) {
			t.Helper()
			require.NotNil(t, r)
			assert.Same(t, want, r.runner)
		}
	}

	// checkOutput := func() NewResolverFn {
	// 	return func(t *testing.T, r *Resolver) {
	// 		t.Helper()
	// 		targets, err := r.Resolve(context.Background(), []string{"example.com/..."}, false)
	// 		require.NoError(t, err)
	// 		require.Len(t, targets, 1)
	// 		assert.Equal(t, "example.com/pkg", targets[0].ImportPath)
	// 	}
	// }

	// checkError := func() NewResolverFn {
	// 	return func(t *testing.T, r *Resolver) {
	// 		t.Helper()
	// 		_, err := r.Resolve(context.Background(), []string{"."}, false)
	// 		assert.EqualError(t, err, "boom")
	// 	}
	// }

	tests := []struct {
		name   string
		config *ResolverConfig
		checks []NewResolverFn
	}{
		{
			name:   "nil_config_uses_default_runner",
			config: nil,
			checks: checkNewResolver(checkDefaults),
		},
		{
			name:   "empty_config_uses_default_runner",
			config: &ResolverConfig{},
			checks: checkNewResolver(checkDefaults),
		},
		{
			name:   "nil_runner_uses_default_runner",
			config: &ResolverConfig{Runner: nil},
			checks: checkNewResolver(checkDefaults),
		},
		{
			name:   "custom_runner_kept",
			config: &ResolverConfig{Runner: fakeRunner},
			checks: checkNewResolver(checkSame(fakeRunner)),
		},
		{
			name:   "custom_runner_with_dir_kept",
			config: &ResolverConfig{Runner: dirRunner},
			checks: checkNewResolver(
				checkSame(dirRunner),
				func(t *testing.T, r *Resolver) {
					t.Helper()
					gr := r.runner.(*goListRunner)
					assert.Equal(t, "/custom/dir", gr.dir)
				},
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			r := NewResolver(tt.config)
			for _, c := range tt.checks {
				c(t, r)
			}
		})
	}
}

type checkResolverResolveFn func(*testing.T, []Target, error)

var checkResolverResolve = func(fns ...checkResolverResolveFn) []checkResolverResolveFn { return fns }

var capturedRunner *fakeGoListRunner

func checkResolveError(want string) checkResolverResolveFn {
	return func(t *testing.T, _ []Target, err error) {
		t.Helper()
		if want == "" {
			assert.NoErrorf(t, err, "checkResolveError: expected no error, got %v", err)
			return
		}
		if assert.Errorf(t, err, "checkResolveError: expected error %q", want) {
			assert.Containsf(t, err.Error(), want, "checkResolveError mismatch")
		}
	}
}
func TestResolver_Resolve(t *testing.T) {
	marshalJSON := func(v any) string {
		b, _ := json.Marshal(v)
		return string(b)
	}

	pkgJSON := func(imp, dir, modPath, modDir string, goFiles, testGoFiles, xtestGoFiles []string, errStr string) string {
		var errJSON string
		if errStr == "" {
			errJSON = "null"
		} else {
			b, _ := json.Marshal(packageError{Err: errStr})
			errJSON = string(b)
		}
		return fmt.Sprintf(`{"ImportPath":%q,"Dir":%q,"ModulePath":%q,"ModuleDir":%q,"GoFiles":%s,"TestGoFiles":%s,"XTestGoFiles":%s,"Error":%s}`+"\n",
			imp, dir, modPath, modDir,
			marshalJSON(goFiles), marshalJSON(testGoFiles), marshalJSON(xtestGoFiles), errJSON)
	}

	singlePkg := pkgJSON(
		"github.com/padiazg/go-crap/internal/packages",
		"/go/src/github.com/padiazg/go-crap/internal/packages",
		"github.com/padiazg/go-crap",
		"/go/src/github.com/padiazg/go-crap",
		[]string{"packages.go"},
		nil,
		nil,
		"",
	)

	singlePkgWithTests := pkgJSON(
		"github.com/padiazg/go-crap/internal/packages",
		"/go/src/github.com/padiazg/go-crap/internal/packages",
		"github.com/padiazg/go-crap",
		"/go/src/github.com/padiazg/go-crap",
		[]string{"packages.go"},
		[]string{"packages_test.go"},
		[]string{"packages_extra_test.go"},
		"",
	)

	badPkg := pkgJSON(
		"github.com/padiazg/go-crap/internal/bad",
		"/go/src/github.com/padiazg/go-crap/internal/bad",
		"github.com/padiazg/go-crap",
		"/go/src/github.com/padiazg/go-crap",
		nil,
		nil,
		nil,
		"cannot find module",
	)

	tests := []struct {
		name         string
		patterns     []string
		includeTests bool
		checks       []checkResolverResolveFn
		before       func(*Resolver)
	}{
		{
			name:     "resolves_single_package",
			patterns: []string{"github.com/padiazg/go-crap/..."},
			before:   func(s *Resolver) { s.runner = &fakeGoListRunner{Output: []byte(singlePkg)} },
			checks: checkResolverResolve(
				func(t *testing.T, targets []Target, err error) {
					t.Helper()
					require.NoError(t, err)
					require.Len(t, targets, 1)
					assert.Equal(t, "github.com/padiazg/go-crap/internal/packages", targets[0].ImportPath)
					assert.Equal(t, "/go/src/github.com/padiazg/go-crap/internal/packages", targets[0].Dir)
					require.Len(t, targets[0].Files, 1)
					assert.Equal(t, "/go/src/github.com/padiazg/go-crap/internal/packages/packages.go", targets[0].Files[0])
				},
				checkResolveError(""),
			),
		},
		{
			name:     "propagates_runner_error",
			patterns: []string{"./..."},
			before:   func(s *Resolver) { s.runner = &fakeGoListRunner{Err: fmt.Errorf("boom")} },
			checks: checkResolverResolve(
				checkResolveError("boom"),
			),
		},
		{
			name:     "empty_output_returns_empty",
			patterns: []string{"./..."},
			before:   func(s *Resolver) { s.runner = &fakeGoListRunner{Output: nil} },
			checks: checkResolverResolve(
				func(t *testing.T, targets []Target, err error) {
					t.Helper()
					assert.NoError(t, err)
					assert.Empty(t, targets)
				},
				checkResolveError(""),
			),
		},
		{
			name:     "malformed_json_returns_decode_error",
			patterns: []string{"./..."},
			before:   func(s *Resolver) { s.runner = &fakeGoListRunner{Output: []byte(`{"ImportPath":`)} },
			checks: checkResolverResolve(
				checkResolveError("decode JSON"),
			),
		},
		{
			name:     "all_packages_error_returns_joined_error",
			patterns: []string{"./..."},
			before:   func(s *Resolver) { s.runner = &fakeGoListRunner{Output: []byte(badPkg)} },
			checks: checkResolverResolve(
				checkResolveError("resolve patterns"),
			),
		},
		{
			name:     "mixed_valid_and_error_returns_targets",
			patterns: []string{"./..."},
			before: func(s *Resolver) {
				var buf bytes.Buffer
				buf.WriteString(singlePkg)
				buf.WriteString(badPkg)
				s.runner = &fakeGoListRunner{Output: buf.Bytes()}
			},
			checks: checkResolverResolve(
				func(t *testing.T, targets []Target, err error) {
					t.Helper()
					require.NoError(t, err)
					require.Len(t, targets, 1)
					assert.Equal(t, "github.com/padiazg/go-crap/internal/packages", targets[0].ImportPath)
				},
				checkResolveError(""),
			),
		},
		{
			name:         "include_tests_collects_test_files",
			patterns:     []string{"./..."},
			includeTests: true,
			before:       func(s *Resolver) { s.runner = &fakeGoListRunner{Output: []byte(singlePkgWithTests)} },
			checks: checkResolverResolve(
				func(t *testing.T, targets []Target, err error) {
					t.Helper()
					require.NoError(t, err)
					require.Len(t, targets, 1)
					require.Len(t, targets[0].TestFiles, 2)
					assert.Contains(t, targets[0].TestFiles[0], "packages_test.go")
					assert.Contains(t, targets[0].TestFiles[1], "packages_extra_test.go")
				},
				checkResolveError(""),
			),
		},
		{
			name:     "patterns_passed_to_runner",
			patterns: []string{"github.com/padiazg/go-crap/internal/...", "./..."},
			before: func(s *Resolver) {
				fake := &fakeGoListRunner{Output: []byte(singlePkg)}
				s.runner = fake
				capturedRunner = fake
			},
			checks: checkResolverResolve(
				func(t *testing.T, targets []Target, err error) {
					t.Helper()
					assert.NoError(t, err)
					require.Len(t, capturedRunner.Patterns, 2)
					assert.Equal(t, "github.com/padiazg/go-crap/internal/...", capturedRunner.Patterns[0])
					assert.Equal(t, "./...", capturedRunner.Patterns[1])
				},
				checkResolveError(""),
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			capturedRunner = nil
			s := NewResolver(nil)
			if tt.before != nil {
				tt.before(s)
			}
			r, err := s.Resolve(context.Background(), tt.patterns, tt.includeTests)
			for _, c := range tt.checks {
				c(t, r, err)
			}
		})
	}
}
