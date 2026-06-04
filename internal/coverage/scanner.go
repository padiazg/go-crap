package coverage

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

type ScanOptions struct {
	Exclude *regexp.Regexp
	Timeout time.Duration
	Path    string
}

func Scan(ctx context.Context, opts ScanOptions) ([]ModuleCoverage, error) {
	if opts.Timeout == 0 {
		opts.Timeout = 10 * time.Minute
	}

	if opts.Path == "" {
		opts.Path = "."
	}

	modules, err := discoverModules(ctx, opts.Path)
	if err != nil {
		return nil, fmt.Errorf("discover modules: %w", err)
	}

	var results []ModuleCoverage
	for _, modDir := range modules {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		mc, err := scanModule(ctx, modDir, opts.Exclude, opts.Timeout)
		if err != nil {
			results = append(results, ModuleCoverage{
				Dir:   modDir,
				Error: fmt.Errorf("scan %s: %w", modDir, err),
			})
			continue
		}

		results = append(results, mc)
	}

	return results, nil
}

func discoverModules(ctx context.Context, root string) ([]string, error) {
	var modules []string
	err := walkForModules(root, func(dir string) bool {
		select {
		case <-ctx.Done():
			return false
		default:
		}
		gomod := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(gomod); err == nil {
			absPath, err := filepath.Abs(dir)
			if err == nil {
				modules = append(modules, absPath)
			} else {
				modules = append(modules, dir)
			}
			return false
		}
		return true
	})
	if err != nil {
		return nil, err
	}
	return modules, nil
}

func walkForModules(root string, visit func(dir string) bool) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			return nil
		}
		if !visit(path) {
			return filepath.SkipDir
		}
		return nil
	})
}

func scanModule(ctx context.Context, modDir string, exclude *regexp.Regexp, timeout time.Duration) (ModuleCoverage, error) {
	mc := ModuleCoverage{Dir: modDir}
	modulePath, err := readModulePath(modDir)
	if err != nil {
		modulePath = filepath.Base(modDir)
	}

	mc.ModulePath = modulePath
	profile, err := runTests(ctx, modDir, exclude, timeout)
	if err != nil {
		mc.Error = err
		return mc, err
	}

	// defer os.Remove(profile) //nolint

	data, err := runCoverTool(ctx, profile, modDir)
	if err != nil {
		mc.Error = err
		return mc, err
	}

	functions, err := parseCoverOutput(bytes.NewReader(data))
	if err != nil {
		mc.Error = err
		return mc, err
	}

	mc.Functions = filterByExclude(functions, exclude)
	return mc, nil
}

func readModulePath(dir string) (string, error) {
	gomod, err := os.ReadFile(filepath.Join(dir, "go.mod"))
	if err != nil {
		return "", err
	}
	for line := range strings.SplitSeq(string(gomod), "\n") {
		line = strings.TrimSpace(line)
		if after, ok := strings.CutPrefix(line, "module "); ok {
			return strings.TrimSpace(after), nil
		}
	}
	return "", fmt.Errorf("no module declaration in go.mod")
}

func runTests(ctx context.Context, modDir string, _ *regexp.Regexp, timeout time.Duration) (string, error) {
	tmpfile, err := os.CreateTemp("", "coverage-*.out")
	if err != nil {
		return "", err
	}
	_ = tmpfile.Close()
	profile := tmpfile.Name()
	cmd := exec.CommandContext(ctx, "go", "test", "-coverprofile="+profile, "./...")
	cmd.Dir = modDir
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		_ = os.Remove(profile)
		return "", fmt.Errorf("go test: %w\n%s", err, stderr.String())
	}

	return profile, nil
}

func filterByExclude(functions []FunctionCoverage, ignore *regexp.Regexp) []FunctionCoverage {
	if ignore == nil {
		return functions
	}
	var kept []FunctionCoverage
	for _, fn := range functions {
		if !ignore.MatchString(fn.File) {
			kept = append(kept, fn)
		}
	}
	return kept
}

func runCoverTool(ctx context.Context, profile, modDir string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "go", "tool", "cover", "-func="+profile)
	cmd.Dir = modDir
	return cmd.Output()
}
