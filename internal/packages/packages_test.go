package packages

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

type checkResolverResolveFn func(*testing.T, *Resolver, []Target, error)

var checkResolverResolve = func(fns ...checkResolverResolveFn) []checkResolverResolveFn { return fns }

func checkResolveError(want string) checkResolverResolveFn {
	return func(t *testing.T, s *Resolver, _ []Target, err error) {
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

func checkTargetsEqual(want []Target) checkResolverResolveFn {
	return func(t *testing.T, s *Resolver, got []Target, _ error) {
		t.Helper()
		assert.Equalf(t, want, got, "checkTargetsEqual mismatch")
	}
}

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

func TestResolver_Resolve(t *testing.T) {
	fakeSingle := &fakeGoListRunner{
		Output: []byte(`{"Dir":"/proj/root","ImportPath":"example.com/fixture","ModuleDir":"/proj","ModulePath":"example.com/fixture","GoFiles":["root.go"]}`),
	}
	fakeTests := &fakeGoListRunner{
		Output: []byte(`{"Dir":"/proj/root","ImportPath":"example.com/fixture","ModuleDir":"/proj","ModulePath":"example.com/fixture","GoFiles":["root.go"],"TestGoFiles":["root_test.go"],"XTestGoFiles":["root_x_test.go"]}`),
	}
	fakeMultiple := &fakeGoListRunner{
		Output: []byte(`{"Dir":"/proj/root","ImportPath":"example.com/fixture","GoFiles":["root.go"]}
{"Dir":"/proj/sub","ImportPath":"example.com/fixture/sub","GoFiles":["sub.go"]}`),
	}
	fakePartial := &fakeGoListRunner{
		Output: []byte(`{"Dir":"/proj/root","ImportPath":"example.com/fixture","GoFiles":["root.go"]}
{"ImportPath":"example.com/fixture/nope","Error":{"Err":"no Go files in /proj/nope"}}`),
	}
	fakeAllFailed := &fakeGoListRunner{
		Output: []byte(`{"ImportPath":"example.com/fixture/a","Error":{"Err":"no Go files in /proj/a"}}
{"ImportPath":"example.com/fixture/b","Error":{"Err":"no Go files in /proj/b"}}`),
	}
	fakeRunnerErr := &fakeGoListRunner{Err: errors.New("boom")}
	fakeBadJSON := &fakeGoListRunner{
		Output: []byte(`{"Dir":"/proj/root","ImportPath":"example.com/fixture"`),
	}
	fakePatterns := &fakeGoListRunner{
		Output: []byte(`{"Dir":"/proj/root","ImportPath":"example.com/fixture","GoFiles":["root.go"]}`),
	}

	tests := []struct {
		name         string
		patterns     []string
		includeTests bool
		checks       []checkResolverResolveFn
		config       *ResolverConfig
		before       func(*Resolver)
	}{
		{
			name:     "resolves_single_package",
			patterns: []string{"."},
			config:   &ResolverConfig{Runner: fakeSingle},
			checks: checkResolverResolve(
				checkResolveError(""),
				checkTargetsEqual([]Target{{
					Dir:        "/proj/root",
					ImportPath: "example.com/fixture",
					ModuleDir:  "/proj",
					ModulePath: "example.com/fixture",
					Files:      []string{"/proj/root/root.go"},
				}}),
			),
		},
		{
			name:         "collects_test_files",
			patterns:     []string{"."},
			includeTests: true,
			config:       &ResolverConfig{Runner: fakeTests},
			checks: checkResolverResolve(
				checkResolveError(""),
				checkTargetsEqual([]Target{{
					Dir:        "/proj/root",
					ImportPath: "example.com/fixture",
					ModuleDir:  "/proj",
					ModulePath: "example.com/fixture",
					Files:      []string{"/proj/root/root.go"},
					TestFiles:  []string{"/proj/root/root_test.go", "/proj/root/root_x_test.go"},
				}}),
			),
		},
		{
			name:     "resolves_multiple_packages",
			patterns: []string{"./..."},
			config:   &ResolverConfig{Runner: fakeMultiple},
			checks: checkResolverResolve(
				checkResolveError(""),
				checkTargetsEqual([]Target{
					{
						Dir:        "/proj/root",
						ImportPath: "example.com/fixture",
						Files:      []string{"/proj/root/root.go"},
					},
					{
						Dir:        "/proj/sub",
						ImportPath: "example.com/fixture/sub",
						Files:      []string{"/proj/sub/sub.go"},
					},
				}),
			),
		},
		{
			name:     "skips_packages_with_error",
			patterns: []string{"./..."},
			config:   &ResolverConfig{Runner: fakePartial},
			checks: checkResolverResolve(
				checkResolveError(""),
				checkTargetsEqual([]Target{{
					Dir:        "/proj/root",
					ImportPath: "example.com/fixture",
					Files:      []string{"/proj/root/root.go"},
				}}),
			),
		},
		{
			name:     "returns_error_when_all_packages_fail",
			patterns: []string{"./..."},
			config:   &ResolverConfig{Runner: fakeAllFailed},
			checks: checkResolverResolve(
				checkResolveError("resolve patterns: example.com/fixture/a: no Go files in /proj/a; example.com/fixture/b: no Go files in /proj/b"),
			),
		},
		{
			name:     "propagates_runner_error",
			patterns: []string{"."},
			config:   &ResolverConfig{Runner: fakeRunnerErr},
			checks: checkResolverResolve(
				checkResolveError("boom"),
			),
		},
		{
			name:     "invalid_json_returns_decode_error",
			patterns: []string{"."},
			config:   &ResolverConfig{Runner: fakeBadJSON},
			checks: checkResolverResolve(
				checkResolveError("decode JSON"),
			),
		},
		{
			name:     "empty_output_returns_no_targets",
			patterns: []string{"."},
			config:   &ResolverConfig{Runner: &fakeGoListRunner{}},
			checks: checkResolverResolve(
				checkResolveError(""),
				checkTargetsEqual(nil),
			),
		},
		{
			name:     "forwards_patterns_to_runner",
			patterns: []string{"./...", "./sub"},
			config:   &ResolverConfig{Runner: fakePatterns},
			checks: checkResolverResolve(
				checkResolveError(""),
				func(t *testing.T, s *Resolver, _ []Target, err error) {
					t.Helper()
					assert.Equalf(t, []string{"./...", "./sub"}, fakePatterns.Patterns, "patterns should be forwarded to the runner")
				},
			),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			s := NewResolver(tt.config)
			if tt.before != nil {
				tt.before(s)
			}
			r, err := s.Resolve(context.Background(), tt.patterns, tt.includeTests)
			for _, c := range tt.checks {
				c(t, s, r, err)
			}
		})
	}
}
