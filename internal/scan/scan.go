package scan

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/padiazg/go-crap/internal/complexity"
	"github.com/padiazg/go-crap/internal/coverage"
	"github.com/padiazg/go-crap/internal/merge"
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
	Missing          string
	MutationReport   string
	Path             string
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

	pr.StartPhase(pkgprogress.PhaseCoverageTests, 0)
	coverages, err := runCoverageAnalysis(ctx, options, exclude, timeout)
	if err != nil {
		return nil, err
	}
	pr.FinishPhase()

	logCoverageErrors(options.Logger, coverages)

	pr.StartPhase(pkgprogress.PhaseComplexity, 0)
	stats := complexity.Analyze([]string{options.Path}, exclude, options.Logger)
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

func runCoverageAnalysis(ctx context.Context, options *Options, exclude *regexp.Regexp, timeout time.Duration) ([]coverage.ModuleCoverage, error) {
	scanner := coverage.NewScanner(options.Path, exclude, options.Logger, timeout)
	scanner.Profile = options.CoverageProfile
	scanner.Progress = options.ProgressReporter
	coverages, err := scanner.Scan(ctx)
	if err != nil {
		return nil, fmt.Errorf("coverage scan: %w", err)
	}
	return coverages, nil
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
