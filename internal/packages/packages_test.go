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

type getListFn func(*testing.T, *bytes.Buffer, error)

var checkgetList = func(fns ...getListFn) []getListFn { return fns }

func checkgetListError(want string) getListFn {
	return func(t *testing.T, _ *bytes.Buffer, err error) {
		t.Helper()
		if want == "" {
			assert.NoErrorf(t, err, "checkgetListError: expected no error, got %v", err)
			return
		}
		if assert.Errorf(t, err, "checkgetListError: expected error %q", want) {
			assert.Containsf(t, err.Error(), want, "checkgetListError mismatch")
		}
	}
}

func checkgetListBuffer(want string) getListFn {
	return func(t *testing.T, buf *bytes.Buffer, err error) {
		t.Helper()
		assert.NotNil(t, buf, "buffer should not be nil")
		assert.Contains(t, buf.String(), want, "getListBuffer mismatch")
	}
}

func Test_getList(t *testing.T) {
	tests := []struct {
		name       string
		patterns   []string
		checks     []getListFn
		newContext func() context.Context
	}{
		{
			name: "empty_patterns",
			checks: checkgetList(
				checkgetListError(""),
				checkgetListBuffer("ImportPath"),
			),
		},
		{
			name:     "dot_pattern",
			patterns: []string{"."},
			checks: checkgetList(
				checkgetListError(""),
				checkgetListBuffer("github.com/padiazg/go-crap/internal/packages"),
			),
		},
		{
			name:     "recursive_pattern",
			patterns: []string{"./..."},
			checks: checkgetList(
				checkgetListError(""),
				checkgetListBuffer("github.com/padiazg/go-crap/internal/packages"),
			),
		},
		{
			name:     "module_path_pattern",
			patterns: []string{"github.com/padiazg/go-crap/internal/packages"},
			checks: checkgetList(
				checkgetListError(""),
				checkgetListBuffer("github.com/padiazg/go-crap/internal/packages"),
			),
		},
		{
			name: "multiple_patterns",
			patterns: []string{
				"github.com/padiazg/go-crap/internal/packages",
				"github.com/padiazg/go-crap/internal/score",
			},
			checks: checkgetList(
				checkgetListError(""),
				func(t *testing.T, buf *bytes.Buffer, err error) {
					t.Helper()
					require.NoError(t, err)
					dec := json.NewDecoder(buf)
					var paths []string
					for dec.More() {
						var pkg listPackage
						err := dec.Decode(&pkg)
						require.NoError(t, err, "unexpected decode error")
						paths = append(paths, pkg.ImportPath)
					}
					require.Len(t, paths, 2, "expected exactly 2 packages")
					assert.Contains(t, paths, "github.com/padiazg/go-crap/internal/packages")
					assert.Contains(t, paths, "github.com/padiazg/go-crap/internal/score")
				},
			),
		},
		{
			name:     "nonexistent_pattern",
			patterns: []string{"./nonexistent/..."},
			checks: checkgetList(
				// -e embeds errors in JSON; getList defers to getTargets.
				checkgetListError(""),
				checkgetListBuffer("no such file or directory"),
			),
		},
		{
			name:     "malformed_pattern",
			patterns: []string{"("},
			checks: checkgetList(
				checkgetListError(""),
				checkgetListBuffer("malformed import path"),
			),
		},
		{
			name: "ctx_cancelled",
			checks: checkgetList(
				checkgetListError("context canceled"),
			),
			newContext: func() context.Context {
				ctx, cancel := context.WithCancel(context.Background())
				cancel()
				return ctx
			},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()
			if tt.newContext != nil {
				ctx = tt.newContext()
			}
			r, err := getList(ctx, tt.patterns)
			for _, c := range tt.checks {
				c(t, r, err)
			}
		})
	}
}

type getTargetsFn func(*testing.T, []Target, []string, error)

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
