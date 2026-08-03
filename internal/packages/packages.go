package packages

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

// Target represents a single Go package resolved from a pattern.
type Target struct {
	ImportPath string
	Dir        string
	Files      []string
	TestFiles  []string
	ModulePath string
	ModuleDir  string
}

type packageError struct {
	ImportPath string
	Pos        string
	Err        string
}

// listPackage is a minimal representation of go list -json output for a package.
type listPackage struct {
	ImportPath   string
	Dir          string
	ModulePath   string
	ModuleDir    string
	GoFiles      []string
	TestGoFiles  []string
	XTestGoFiles []string
	Error        packageError
}

type ResolverConfig struct {
	Runner GoListRunner
}

// Resolver resolves package patterns via `go list -json -e`.
type Resolver struct {
	runner GoListRunner
}

// NewResolver creates a Resolver with the given options.
// Defaults to the real GoListRunner.
func NewResolver(config *ResolverConfig) *Resolver {
	if config == nil {
		config = &ResolverConfig{}
	}

	if config.Runner == nil {
		config.Runner = NewGoListRunner()
	}

	return &Resolver{
		runner: config.Runner,
	}
}

// Resolve resolves package patterns via `go list -json -e`.
func (r *Resolver) Resolve(ctx context.Context, patterns []string, includeTests bool) ([]Target, error) {
	stdout, err := r.runner.List(ctx, patterns)
	if err != nil {
		return nil, err
	}

	var b bytes.Buffer
	b.Write(stdout)

	targets, errors, err := getTargets(&b, includeTests)
	if err != nil {
		return nil, err
	}

	if len(targets) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("resolve patterns: %s", strings.Join(errors, "; "))
	}

	return targets, nil
}

func getTargets(stdout *bytes.Buffer, includeTests bool) ([]Target, []string, error) {
	dec := json.NewDecoder(stdout)
	var targets []Target
	var errors []string

	for dec.More() {
		var pkg listPackage
		if err := dec.Decode(&pkg); err != nil {
			return nil, nil, fmt.Errorf("resolve patterns: decode JSON: %w", err)
		}

		if pkg.Error.Err != "" {
			errors = append(errors, fmt.Sprintf("%s: %s", pkg.ImportPath, pkg.Error.Err))
			continue
		}

		t := Target{
			ImportPath: pkg.ImportPath,
			Dir:        pkg.Dir,
			ModulePath: pkg.ModulePath,
			ModuleDir:  pkg.ModuleDir,
		}

		// Convert relative GoFiles to absolute paths.
		for _, f := range pkg.GoFiles {
			abs := filepath.Join(pkg.Dir, f)
			t.Files = append(t.Files, abs)
		}

		if includeTests {
			for _, f := range append(pkg.TestGoFiles, pkg.XTestGoFiles...) {
				abs := filepath.Join(pkg.Dir, f)
				t.TestFiles = append(t.TestFiles, abs)
			}
		}

		targets = append(targets, t)
	}

	return targets, errors, nil
}
