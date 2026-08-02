package coverage

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/padiazg/go-crap/pkg/dummylogger"
	"github.com/padiazg/go-crap/pkg/logger"
	"github.com/padiazg/go-crap/pkg/progress"
)

// ModuleTarget holds per-module package targets for targeted coverage analysis.
type ModuleTarget struct {
	ModDir   string
	PkgDirs  []string // module-relative package directories to run go test on
}

type Scanner struct {
	Logger   logger.Logger
	Progress progress.Reporter
	Exclude  *regexp.Regexp
	Path     string
	// Profile, when set, is used as the coverage profile instead of running
	// "go test". The same profile is applied to every discovered module;
	// entries whose paths do not belong to a module are skipped.
	Profile string
	Timeout time.Duration

	// Targets, when set, overrides the Path-based module discovery with
	// explicit per-module package targets.
	Targets []ModuleTarget
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
	if s.Profile != "" {
		if _, err := os.Stat(s.Profile); err != nil {
			return nil, fmt.Errorf("coverage profile: %w", err)
		}
	}

	var modules []string

	if len(s.Targets) > 0 {
		for _, t := range s.Targets {
			modules = append(modules, t.ModDir)
		}
	} else {
		var err error
		modules, err = s.discoverModules(ctx)
		if err != nil {
			return nil, fmt.Errorf("discover modules: %w", err)
		}
	}

	// A supplied profile needs the enclosing module only for path
	// resolution, so honor package subdirectories that contain no go.mod of
	// their own by walking up to the nearest module root. This does not
	// apply to the "go test" path, which must run from an actual module.
	if s.Profile != "" && len(modules) == 0 {
		if modDir := findEnclosingModule(s.Path); modDir != "" {
			modules = []string{modDir}
		}
	}

	if s.Progress != nil {
		s.Progress.SetTotal(len(modules))
	}

	var results []ModuleCoverage
	for i, modDir := range modules {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		if s.Progress != nil {
			s.Progress.SetDetail(filepath.Base(modDir))
		}

		var pkgDirs []string
		if i < len(s.Targets) {
			pkgDirs = s.Targets[i].PkgDirs
		}

		mc := s.scanModule(ctx, modDir, pkgDirs)
		if mc.Error != nil {
			s.Logger.Debug("coverage scan: module error", "module", modDir, "error", mc.Error.Error())
			mc.Error = fmt.Errorf("scan %s: %w", modDir, mc.Error)
			results = append(results, mc)
			if s.Progress != nil {
				s.Progress.Advance(1)
			}
			continue
		}

		results = append(results, mc)
		if s.Progress != nil {
			s.Progress.Advance(1)
		}
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

// findEnclosingModule walks up from path to the nearest ancestor directory
// that contains a go.mod, returning its absolute path, or "" if none exists.
func findEnclosingModule(path string) string {
	dir, err := filepath.Abs(path)
	if err != nil {
		return ""
	}
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
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

func (s *Scanner) scanModule(ctx context.Context, modDir string, pkgDirs []string) ModuleCoverage {
	mc := ModuleCoverage{Dir: modDir}
	modulePath, err := readModulePath(modDir)
	if err != nil {
		s.Logger.Debug("coverage scan: read module path", "error", err.Error())
		modulePath = filepath.Base(modDir)
	}

	mc.ModulePath = modulePath
	mc.Profile = s.Profile
	var parseErr error
	runTestsSuccess := false
	if mc.Profile == "" {
		mc.Profile, err = s.runTests(ctx, modDir, pkgDirs)
		if err != nil {
			mc.Error = fmt.Errorf("runTests: %w", err)
		} else {
			runTestsSuccess = true
		}
	}

	functions, parseErr := parseCoverProfile(mc.Profile, modDir, modulePath)
	if runTestsSuccess {
		os.Remove(mc.Profile)
	}

	if parseErr != nil {
		if mc.Error != nil {
			return mc
		}
		mc.Error = fmt.Errorf("parseCoverProfile: %w", parseErr)
		return mc
	}

	mc.Functions = s.filterByExclude(functions)
	return mc
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

func (s *Scanner) runTests(ctx context.Context, modDir string, pkgDirs []string) (string, error) {
	tmpfile, err := os.CreateTemp("", "coverage-*.out")
	if err != nil {
		return "", err
	}
	if err := tmpfile.Close(); err != nil {
		s.Logger.Debug("coverage scan: tmpfile close error", "error", err.Error())
	}
	profile := tmpfile.Name()

	args := []string{"test", "-coverprofile=" + profile}
	if len(pkgDirs) > 0 {
		args = append(args, pkgDirs...)
	} else {
		args = append(args, "./...")
	}

	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = modDir
	cmd.Env = append(os.Environ(), "GO111MODULE=on")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		if failed := extractFailedTests(stderr.String()); len(failed) > 0 {
			s.Logger.Warn("coverage: tests failed in module", "module", modDir, "failed_tests", failed)
		}
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return profile, fmt.Errorf("go test: timed out (increase --timeout to allow more time): %w", err)
		}
		return profile, fmt.Errorf("go test: %w\n%s", err, stderr.String())
	}

	return profile, nil
}

func extractFailedTests(stderr string) []string {
	var failed []string
	for line := range strings.SplitSeq(stderr, "\n") {
		line = strings.TrimSpace(line)
		if _, after, ok := strings.Cut(line, "--- FAIL: "); ok {
			after = strings.TrimSpace(after)
			if name, _, _ := strings.Cut(after, " "); name != "" {
				failed = append(failed, name)
			}
		}
	}
	return failed
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
