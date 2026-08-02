package packages

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
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

// Resolve resolves package patterns via `go list -json -e`.
func Resolve(ctx context.Context, patterns []string, includeTests bool) ([]Target, error) {
	args := []string{"list", "-e", "-json"}
	args = append(args, patterns...)

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = "."
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		// -e flag means errors are per-package in JSON; command may still exit non-zero
		// for patterns that resolve to no packages at all.
		if stdout.Len() == 0 {
			msg := "go list: " + stderr.String()
			return nil, fmt.Errorf("resolve patterns: %w\n%s", err, msg)
		}
	}

	// Stream JSON objects (one per package) from the combined output.
	dec := json.NewDecoder(&stdout)
	var targets []Target
	var errors []string

	for dec.More() {
		var pkg listPackage
		if err := dec.Decode(&pkg); err != nil {
			return nil, fmt.Errorf("resolve patterns: decode JSON: %w", err)
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

	if len(targets) == 0 && len(errors) > 0 {
		return nil, fmt.Errorf("resolve patterns: %s", strings.Join(errors, "; "))
	}

	return targets, nil
}

// listPackage is a minimal representation of go list -json output for a package.
type listPackage struct {
	ImportPath  string
	Dir         string
	ModulePath  string
	ModuleDir   string
	GoFiles     []string
	TestGoFiles []string
	XTestGoFiles []string
	Error       struct {
		ImportPath string
		Pos        string
		Err        string
	}
}
