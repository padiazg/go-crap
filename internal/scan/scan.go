package scan

import (
	"context"
	"fmt"
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
	Path           string
	Exclude        []string
	Min            float64
	Top            int
	MutationReport string
}

func Scan(options *Options) (*score.EntryList, error) {
	// TODO: make timeout configurable
	// TODO: use goroutine to catch timeout signal
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	exclude, err := utils.BuildExcludeRegex(options.Exclude)
	if err != nil {
		return nil, fmt.Errorf("coverage scan: %w", err)
	}

	coverages, err := coverage.Scan(ctx, coverage.ScanOptions{
		Path:    options.Path,
		Exclude: exclude,
		Logger:  options.Logger,
	})
	if err != nil {
		return nil, fmt.Errorf("coverage scan: %w", err)
	}

	if options.Logger != nil {
		for _, mc := range coverages {
			if mc.Error != nil {
				options.Logger.Debug("coverage scan error", "module", mc.Dir, "error", mc.Error.Error())
			}
		}
	}

	stats := complexity.Analyze([]string{options.Path}, exclude, options.Logger)

	merged := merge.Merge(coverages, stats)

	policy, err := parseMissingPolicy(options.Missing)
	if err != nil {
		return nil, err
	}

	entries := score.Score(merged, policy)

	if options.MutationReport != "" {
		mutReport, err := mutation.ParseReport(options.MutationReport)
		if err != nil {
			return nil, fmt.Errorf("mutation report: %w", err)
		}

		entries = mutation.Annotate(entries, mutReport, merged)
	}

	entries = applyFilters(entries, options.Top, options.Min)

	return &score.EntryList{List: entries}, nil
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
	if e.EffectiveCRAP == 0 {
		return e.CRAP
	}
	return e.EffectiveCRAP
}

func applyFilters(entries []score.CRAPEntry, top int, min float64) []score.CRAPEntry {
	sort.Slice(entries, func(i, j int) bool {
		return effectiveCRAP(entries[i]) > effectiveCRAP(entries[j])
	})

	if min > 0 {
		var filtered []score.CRAPEntry
		for _, e := range entries {
			if e.CoverageUntrusted || effectiveCRAP(e) >= min {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}

	if top > 0 && top < len(entries) {
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

	return entries
}
