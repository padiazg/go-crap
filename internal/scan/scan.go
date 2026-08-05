package scan

import (
	"context"
	"errors"
	"fmt"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/padiazg/go-crap/internal/complexity"
	"github.com/padiazg/go-crap/internal/coverage"
	"github.com/padiazg/go-crap/internal/merge"
	"github.com/padiazg/go-crap/internal/packages"
	"github.com/padiazg/go-crap/internal/score"
	"github.com/padiazg/go-crap/pkg/logger"
	pkgprogress "github.com/padiazg/go-crap/pkg/progress"
	"github.com/padiazg/go-crap/pkg/utils"
)

// Sentinel errors.
var (
	ErrUnknownPolicy     = errors.New("unknown missing policy")
	ErrThresholdExceeded = errors.New("CRAP threshold exceeded")
	ErrRegression        = errors.New("CRAP regression detected")
)

type Options struct {
	Logger           logger.Logger
	ProgressReporter pkgprogress.Reporter
	CoverageProfile  string
	CoverPkg         string
	Missing          string
	MutationReport   string
	Path             string
	Patterns         []string
	Exclude          []string
	Min              float64
	Timeout          time.Duration
	Top              int
	IncludeTests     bool
}

// DefaultTimeout is used when Options.Timeout is unset (zero).
const DefaultTimeout = 10 * time.Minute

// resolveTimeout returns t, or DefaultTimeout when t is zero.
func resolveTimeout(t time.Duration) time.Duration {
	if t == 0 {
		return DefaultTimeout
	}
	return t
}

func Scan(options *Options) (*Entries, error) {
	// TODO: use goroutine to catch timeout signal
	timeout := resolveTimeout(options.Timeout)
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	exclude, err := utils.BuildExcludeRegex(buildExcludePatterns(options.Exclude, options.IncludeTests))
	if err != nil {
		return nil, fmt.Errorf("coverage scan: %w", err)
	}

	pr := options.ProgressReporter
	if pr == nil {
		pr = pkgprogress.NoopReporter{}
	}
	defer pr.Done()
	defer pr.Errored()

	// Resolve patterns or fall back to path-based discovery.
	patterns := options.Patterns
	if len(patterns) == 0 && options.Path != "" {
		// Back-compat: if Patterns not set but Path is, treat Path as a single pattern.
		patterns = []string{options.Path}
	}
	if len(patterns) == 0 {
		patterns = []string{"./..."}
	}

	coveragePhase := pkgprogress.PhaseCoverageTests
	if options.CoverageProfile != "" {
		coveragePhase = pkgprogress.PhaseCoverageProfile
	}
	pr.StartPhase(coveragePhase, 0)
	coverages, err := runCoverageAnalysis(ctx, options, patterns, exclude, timeout)
	if err != nil {
		return nil, err
	}
	pr.FinishPhase()

	logCoverageErrors(options.Logger, coverages)

	pr.StartPhase(pkgprogress.PhaseComplexity, 0)
	stats := runComplexityAnalysis(ctx, options, patterns, exclude)
	pr.FinishPhase()

	pr.StartPhase(pkgprogress.PhaseProcessing, 0)
	merged := merge.Merge(coverages, stats)

	policy, err := parseMissingPolicy(options.Missing)
	if err != nil {
		return nil, err
	}

	entries, err := NewEntries(options, merged, policy)
	pr.FinishPhase()
	return entries, err
}

// func runCoverageAnalysis(ctx context.Context, options *Options, patterns []string, exclude *regexp.Regexp, timeout time.Duration) ([]coverage.ModuleCoverage, error) {
// 	scanner := coverage.NewScanner(".", exclude, options.Logger, timeout)
// 	scanner.Profile = options.CoverageProfile
// 	scanner.Progress = options.ProgressReporter

// 	// If a profile is provided, fall back to path-based discovery (no go list needed).
// 	if scanner.Profile != "" {
// 		coverages, err := scanner.Scan(ctx)
// 		if err != nil {
// 			return nil, fmt.Errorf("coverage scan: %w", err)
// 		}
// 		return coverages, nil
// 	}

// 	// Try to resolve patterns via go list. On failure, fall back to Path-based discovery.
// 	if len(patterns) == 0 && options.Path != "" {
// 		patterns = []string{options.Path}
// 	}

// 	targets, resolveErr := packages.Resolve(ctx, patterns, options.IncludeTests)
// 	if resolveErr == nil && len(targets) > 0 {
// 		// Group targets by module dir.
// 		mt := groupTargets(targets)
// 		scanner.Targets = mt

// 		coverages, err := scanner.Scan(ctx)
// 		if err != nil {
// 			return nil, fmt.Errorf("coverage scan: %w", err)
// 		}
// 		return coverages, nil
// 	}

// 	// Fallback to Path-based discovery (e.g. nested module dirs outside go list reach).
// 	if options.Path != "" {
// 		scanner.Path = options.Path
// 		coverages, err := scanner.Scan(ctx)
// 		if err != nil {
// 			return nil, fmt.Errorf("coverage scan: %w", err)
// 		}
// 		return coverages, nil
// 	}

// 	return nil, fmt.Errorf("coverage scan: resolve patterns: %w", resolveErr)
// }

func runCoverageAnalysis(ctx context.Context, options *Options, patterns []string, exclude *regexp.Regexp, timeout time.Duration) ([]coverage.ModuleCoverage, error) {
	scanner := coverage.NewScanner(".", exclude, options.Logger, timeout)
	scanner.Profile = options.CoverageProfile
	scanner.CoverPkg = options.CoverPkg
	scanner.Progress = options.ProgressReporter

	wrapError := func(fn func(ctx context.Context) ([]coverage.ModuleCoverage, error)) ([]coverage.ModuleCoverage, error) {
		coverages, err := fn(ctx)
		if err != nil {
			return nil, fmt.Errorf("coverage scan: %w", err)
		}

		return coverages, nil
	}

	// If a profile is provided, fall back to path-based discovery (no go list needed).
	if scanner.Profile != "" {
		if modDir := resolveProfileModule(options); modDir != "" {
			scanner.Path = modDir
			scanner.Logger.Debug("using coverage profile", "profile", options.CoverageProfile, "module", modDir)
		}
		return wrapError(scanner.Scan)
	}

	// Try to resolve patterns via go list. On failure, fall back to Path-based discovery.
	if len(patterns) == 0 && options.Path != "" {
		patterns = []string{options.Path}
	}

	resolver := packages.NewResolver(nil)
	targets, resolveErr := resolver.Resolve(ctx, patterns, options.IncludeTests)
	if resolveErr == nil && len(targets) > 0 {
		// Group targets by module dir.
		mt := groupTargets(targets)
		scanner.Targets = mt

		return wrapError(scanner.Scan)
	}

	// Fallback to Path-based discovery (e.g. nested module dirs outside go list reach).
	if options.Path != "" {
		scanner.Path = options.Path
		return wrapError(scanner.Scan)
	}

	return nil, fmt.Errorf("coverage scan: resolve patterns: %w", resolveErr)
}

// resolveProfileModule determines the module to analyze when a coverage
// profile is supplied. An explicit path target wins; otherwise the module is
// derived from the profile's location on disk (the nearest go.mod ancestor of
// the profile file). Returns "" to fall back to the current working directory.
func resolveProfileModule(options *Options) string {
	if options.Path != "" && options.Path != "./..." && options.Path != "." {
		if modDir := coverage.FindEnclosingModule(options.Path); modDir != "" {
			return modDir
		}
		if fi, err := os.Stat(options.Path); err == nil && fi.IsDir() {
			return options.Path
		}
	}
	if modDir := coverage.FindEnclosingModule(options.CoverageProfile); modDir != "" {
		return modDir
	}
	return ""
}

func groupTargets(targets []packages.Target) []coverage.ModuleTarget {
	modTargets := make(map[string]*coverage.ModuleTarget)
	for _, t := range targets {
		if t.ModuleDir == "" {
			continue
		}
		relDir := t.Dir
		if mtd, ok := modTargets[t.ModuleDir]; ok {
			mtd.PkgDirs = append(mtd.PkgDirs, relDir)
		} else {
			modTargets[t.ModuleDir] = &coverage.ModuleTarget{
				ModDir:  t.ModuleDir,
				PkgDirs: []string{relDir},
			}
		}
	}

	var mt []coverage.ModuleTarget
	for _, t := range modTargets {
		mt = append(mt, *t)
	}

	return mt
}

func runComplexityAnalysis(_ context.Context, options *Options, patterns []string, exclude *regexp.Regexp) []complexity.Stat {
	// When a coverage profile is supplied, analyze the same module the profile
	// belongs to so coverage and complexity stay aligned.
	if options.CoverageProfile != "" {
		if modDir := resolveProfileModule(options); modDir != "" {
			return complexity.Analyze([]string{modDir}, exclude, options.Logger)
		}
	}

	resolver := packages.NewResolver(nil)
	targets, err := resolver.Resolve(context.Background(), patterns, options.IncludeTests)
	if err != nil {
		// Fallback: if resolve fails (e.g. outside a module), fall back to Path-based walk.
		if options.Path != "" {
			return complexity.Analyze([]string{options.Path}, exclude, options.Logger)
		}
		return nil
	}

	var files []string
	for _, t := range targets {
		files = append(files, t.Files...)
		files = append(files, t.TestFiles...)
	}

	return complexity.AnalyzeFiles(files, exclude, options.Logger)
}

func logCoverageErrors(l logger.Logger, coverages []coverage.ModuleCoverage) {
	if l == nil {
		return
	}

	for _, mc := range coverages {
		if mc.Error != nil {
			l.Debug("coverage scan error", "module", mc.Dir, "error", mc.Error.Error())
		}
	}
}

func parseMissingPolicy(s string) (score.MissingPolicy, error) {
	switch strings.ToLower(s) {
	case "pessimistic", "":
		return score.MissingPessimistic, nil
	case "optimistic":
		return score.MissingOptimistic, nil
	case "skip":
		return score.MissingSkip, nil
	default:
		return 0, fmt.Errorf("%w: %s (use pessimistic, optimistic, or skip)", ErrUnknownPolicy, s)
	}
}

func effectiveCRAP(e score.CRAPEntry) float64 {
	return e.EffectiveScore()
}

func buildExcludePatterns(userPatterns []string, includeTests bool) []string {
	if includeTests {
		return userPatterns
	}
	return append([]string{utils.DefaultExcludePattern}, userPatterns...)
}
