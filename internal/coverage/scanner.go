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

	"github.com/padiazg/go-crap/pkg/dummylogger"
	"github.com/padiazg/go-crap/pkg/logger"
)

type Scanner struct {
	Exclude *regexp.Regexp
	Logger  logger.Logger
	Timeout time.Duration
	Path    string
}

func NewScanner(path string, exclude *regexp.Regexp, logger logger.Logger, timeout time.Duration) *Scanner {
	opts := &Scanner{
		Path:    path,
		Exclude: exclude,
		Logger:  logger,
		Timeout: timeout,
	}
	if timeout == 0 {
		opts.Timeout = 10 * time.Minute
	}
	if path == "" {
		opts.Path = "."
	}
	if logger == nil {
		opts.Logger = dummylogger.New(nil)
	}
	return opts
}

// Scan walks the filesystem for Go modules, runs tests with coverage, and returns coverage data for each module.
func (s *Scanner) Scan(ctx context.Context) ([]ModuleCoverage, error) {
	modules, err := s.discoverModules(ctx)
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

		mc, err := s.scanModule(ctx, modDir)
		if err != nil {
			s.Logger.Debug("coverage scan: module error", "module", modDir, "error", err.Error())
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

func (s *Scanner) discoverModules(ctx context.Context) ([]string, error) {
	var modules []string
	err := walkForModules(s.Path, func(dir string) bool {
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
				s.Logger.Debug("coverage scan: could not resolve absolute path", "dir", dir, "error", err.Error())
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

func (s *Scanner) scanModule(ctx context.Context, modDir string) (ModuleCoverage, error) {
	mc := ModuleCoverage{Dir: modDir}
	modulePath, err := readModulePath(modDir)
	if err != nil {
		s.Logger.Debug("coverage scan: read module path", "error", err.Error())
		modulePath = filepath.Base(modDir)
	}

	mc.ModulePath = modulePath
	profile, err := s.runTests(ctx, modDir)
	if err != nil {
		mc.Error = fmt.Errorf("runTests: %w", err)
		return mc, mc.Error
	}
	defer func() {
		if removeErr := os.Remove(profile); removeErr != nil {
			s.Logger.Debug("coverage scan: remove temp file error", "profile", profile, "error", removeErr.Error())
		}
	}()

	functions, err := parseCoverProfile(profile, modDir, modulePath)
	if err != nil {
		mc.Error = fmt.Errorf("parseCoverProfile: %w", err)
		return mc, mc.Error
	}

	mc.Functions = s.filterByExclude(functions)
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

func (s *Scanner) runTests(ctx context.Context, modDir string) (string, error) {
	tmpfile, err := os.CreateTemp("", "coverage-*.out")
	if err != nil {
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		s.Logger.Debug("coverage scan: tmpfile close error", "error", err.Error())
	}
	profile := tmpfile.Name()
	// cmd := exec.CommandContext(ctx, "go", "test", "-coverpkg=./...", "-coverprofile="+profile, "./...")
	cmd := exec.CommandContext(ctx, "go", "test", "-coverprofile="+profile, "./...")
	cmd.Dir = modDir
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		if removeErr := os.Remove(profile); removeErr != nil {
			s.Logger.Debug("coverage scan: remove temp file error", "profile", profile, "error", removeErr.Error())
		}
		return "", fmt.Errorf("go test: %w\n%s", err, stderr.String())
	}

	return profile, nil
}

func (s *Scanner) filterByExclude(functions []FunctionCoverage) []FunctionCoverage {
	if s.Exclude == nil {
		return functions
	}

	var kept []FunctionCoverage
	for _, fn := range functions {
		if !s.Exclude.MatchString(fn.File) {
			kept = append(kept, fn)
		}
	}
	return kept
}
