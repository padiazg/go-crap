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
	"github.com/padiazg/go-crap/pkg/utils"
)

// Sentinel errors.
var (
	ErrUnknownPolicy     = errors.New("unknown missing policy")
	ErrThresholdExceeded = errors.New("CRAP threshold exceeded")
)

type Options struct {
	Logger         logger.Logger
	Timeout        time.Duration
	Missing        string
	MutationReport string
	Path           string
	Exclude        []string
	Min            float64
	Top            int
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

	exclude, err := utils.BuildExcludeRegex(options.Exclude)
	if err != nil {
		return nil, fmt.Errorf("coverage scan: %w", err)
	}

	coverages, err := runCoverageAnalysis(ctx, options, exclude, timeout)
	if err != nil {
		return nil, err
	}

	logCoverageErrors(options.Logger, coverages)

	stats := complexity.Analyze([]string{options.Path}, exclude, options.Logger)

	merged := merge.Merge(coverages, stats)

	policy, err := parseMissingPolicy(options.Missing)
	if err != nil {
		return nil, err
	}

	return NewEntries(options, merged, policy)
}

func runCoverageAnalysis(ctx context.Context, options *Options, exclude *regexp.Regexp, timeout time.Duration) ([]coverage.ModuleCoverage, error) {
	scanner := coverage.NewScanner(options.Path, exclude, options.Logger, timeout)
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
