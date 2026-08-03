package packages

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"context"

	"github.com/stretchr/testify/assert"
)

type checkgoListRunnerListFn func(*testing.T, []byte, error)

var checkgoListRunnerList = func(fns ...checkgoListRunnerListFn) []checkgoListRunnerListFn { return fns }

func checkListError(want string) checkgoListRunnerListFn {
	return func(t *testing.T, _ []byte, err error) {
		t.Helper()
		if want == "" {
			assert.NoErrorf(t, err, "checkListError: expected no error, got %v", err)
			return
		}
		if assert.Errorf(t, err, "checkListError: expected error %q", want) {
			assert.Containsf(t, err.Error(), want, "checkListError mismatch")
		}
	}
}
func Test_goListRunner_List(t *testing.T) {
	checkOutputContains := func(want string) checkgoListRunnerListFn {
		return func(t *testing.T, got []byte, err error) {
			t.Helper()
			assert.Containsf(t, string(got), want, "output should contain %q", want)
		}
	}

	writeFixtureModule := func(t *testing.T) string {
		t.Helper()
		dir := t.TempDir()
		os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example.com/fixture\n\ngo 1.26\n"), 0644)
		os.WriteFile(filepath.Join(dir, "root.go"), []byte("package fixture\n\nfunc Root() int { return 1 }\n"), 0644)
		os.MkdirAll(filepath.Join(dir, "sub"), 0755)
		os.WriteFile(filepath.Join(dir, "sub", "sub.go"), []byte("package sub\n\nfunc Sub() int { return 2 }\n"), 0644)
		return dir
	}

	tests := []struct {
		name       string
		patterns   []string
		checks     []checkgoListRunnerListFn
		before     func(*goListRunner)
		newContext func() context.Context
	}{
		{
			name:     "success_lists_single_package",
			patterns: []string{"."},
			before:   func(s *goListRunner) { s.dir = writeFixtureModule(t) },
			checks: checkgoListRunnerList(
				checkListError(""),
				checkOutputContains(`"ImportPath":`),
				checkOutputContains(`"example.com/fixture"`),
			),
		},
		{
			name:     "wildcard_lists_all_packages",
			patterns: []string{"./..."},
			before:   func(s *goListRunner) { s.dir = writeFixtureModule(t) },
			checks: checkgoListRunnerList(
				checkListError(""),
				checkOutputContains(`"example.com/fixture/sub"`),
			),
		},
		{
			name:     "pattern_filters_package_set",
			patterns: []string{"./sub"},
			before:   func(s *goListRunner) { s.dir = writeFixtureModule(t) },
			checks: checkgoListRunnerList(
				checkListError(""),
				func(t *testing.T, got []byte, err error) {
					t.Helper()
					assert.NoError(t, err)
					assert.Contains(t, string(got), `"example.com/fixture/sub"`)
					assert.Equal(t, 1, strings.Count(string(got), `"ImportPath":`), "expected exactly one ImportPath")
				},
			),
		},
		{
			name:     "tolerant_of_missing_pattern",
			patterns: []string{"./nope"},
			before:   func(s *goListRunner) { s.dir = writeFixtureModule(t) },
			checks: checkgoListRunnerList(
				checkListError(""),
				checkOutputContains(`"Error":`),
			),
		},
		{
			name:     "nonexistent_directory_returns_error",
			patterns: []string{"."},
			before: func(s *goListRunner) {
				s.dir = filepath.Join(t.TempDir(), "nope")
			},
			checks: checkgoListRunnerList(
				checkListError("no such file or directory"),
			),
		},
		{
			name:       "canceled_context_returns_error",
			patterns:   []string{"."},
			before:     func(s *goListRunner) { s.dir = writeFixtureModule(t) },
			newContext: func() context.Context { ctx, cancel := context.WithCancel(context.Background()); cancel(); return ctx },
			checks: checkgoListRunnerList(
				checkListError("context canceled"),
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s := &goListRunner{}
			if tt.before != nil {
				tt.before(s)
			}
			ctx := context.Background()
			if tt.newContext != nil {
				ctx = tt.newContext()
			}
			r, err := s.List(ctx, tt.patterns)
			for _, c := range tt.checks {
				c(t, r, err)
			}
		})
	}
}
