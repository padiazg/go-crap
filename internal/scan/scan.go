package scan

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/padiazg/go-crap/internal/complexity"
	"github.com/padiazg/go-crap/internal/coverage"
	"github.com/padiazg/go-crap/internal/merge"
	"github.com/padiazg/go-crap/internal/mutation"
	"github.com/padiazg/go-crap/internal/score"
	"github.com/padiazg/go-crap/pkg/logger"
	"github.com/padiazg/go-crap/pkg/utils"
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

func Scan(options *Options) (*score.EntryList, error) {
	// TODO: make timeout configurable
	// TODO: use goroutine to catch timeout signal
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	exclude, err := buildExcludeRegex(options.Exclude)
	if err != nil {
		return nil, fmt.Errorf("coverage scan: %w", err)
	}

	coverages, err := runCoverageAnalysis(ctx, options, exclude)
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

	entries := score.Score(merged, policy)

	entries = applyMutationAnnotations(options, entries, merged)

	entries = applyFilters(entries, options.Top, options.Min)

	return &score.EntryList{List: entries}, nil
}

func buildExcludeRegex(exclude []string) (*regexp.Regexp, error) {
	return utils.BuildExcludeRegex(exclude)
}

func runCoverageAnalysis(ctx context.Context, options *Options, exclude *regexp.Regexp) ([]coverage.ModuleCoverage, error) {
	coverages, err := coverage.Scan(ctx, coverage.ScanOptions{
		Path:    options.Path,
		Exclude: exclude,
		Logger:  options.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("coverage scan: %w", err)
	}
	return coverages, nil
}

func logCoverageErrors(l logger.Logger, coverages []coverage.ModuleCoverage) {
	if l != nil {
		for _, mc := range coverages {
			if mc.Error != nil {
				l.Debug("coverage scan error", "module", mc.Dir, "error", mc.Error.Error())
			}
		}
	}
}

func applyMutationAnnotations(options *Options, entries []score.CRAPEntry, merged []merge.MergedEntry) []score.CRAPEntry {
	if options.MutationReport != "" {
		mutReport, err := mutation.ParseReport(options.MutationReport)
		if err != nil {
			return nil
		}
		return mutation.Annotate(entries, mutReport, merged)
	}
	return entries
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
		return 0, fmt.Errorf("unknown missing policy: %s (use pessimistic, optimistic, or skip)", s)
	}
}

func effectiveCRAP(e score.CRAPEntry) float64 {
	return e.EffectiveScore()
}

func applyFilters(entries []score.CRAPEntry, top int, min float64) []score.CRAPEntry {
	sort.Slice(entries, func(i, j int) bool {
		return effectiveCRAP(entries[i]) > effectiveCRAP(entries[j])
	})

	if min > 0 {
		entries = filterByMinCRAP(entries, min)
	}

	if top > 0 && top < len(entries) {
		entries = filterByTop(entries, top)
	}

	return entries
}

func filterByMinCRAP(entries []score.CRAPEntry, min float64) []score.CRAPEntry {
	var filtered []score.CRAPEntry
	for _, e := range entries {
		if e.CoverageUntrusted || effectiveCRAP(e) >= min {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

func filterByTop(entries []score.CRAPEntry, top int) []score.CRAPEntry {
	var result []score.CRAPEntry
	for _, e := range entries {
		if e.CoverageUntrusted {
			result = append(result, e)
		}
	}
	for _, e := range entries {
		if !e.CoverageUntrusted && len(result) < top {
			result = append(result, e)
		}
	}
	return result
}
