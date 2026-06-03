package cmd

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/padiazg/go-crap/internal/complexity"
	"github.com/padiazg/go-crap/internal/coverage"
	"github.com/padiazg/go-crap/internal/merge"
	"github.com/padiazg/go-crap/internal/report"
	"github.com/padiazg/go-crap/internal/score"
	"github.com/spf13/cobra"
)

var (
	flagThreshold float64
	flagFailAbove bool
	flagFormat    string
	flagTop       int
	flagMin       float64
	flagMissing   string
	flagExclude   []string

	scanCmd = &cobra.Command{
		Use:   "scan [path]",
		Short: "Analyze Go modules and calculate CRAP scores",
		Args:  cobra.MaximumNArgs(1),
		RunE:  runScan,
	}
)

func init() {
	scanCmd.Flags().Float64VarP(&flagThreshold, "threshold", "t", 30.0,
		"Score above which a function is marked as problematic")
	scanCmd.Flags().BoolVar(&flagFailAbove, "fail-above", false,
		"Exit with code 1 if any function exceeds the threshold")
	scanCmd.Flags().StringVarP(&flagFormat, "format", "f", "table",
		"Output format: table|json|github")
	scanCmd.Flags().IntVar(&flagTop, "top", 0,
		"Show only the N worst offenders (0 = all)")
	scanCmd.Flags().Float64Var(&flagMin, "min", 0,
		"Hide entries below this score")
	scanCmd.Flags().StringVar(&flagMissing, "missing", "pessimistic",
		"Policy for functions without coverage: pessimistic|optimistic|skip")
	scanCmd.Flags().StringArrayVar(&flagExclude, "exclude", nil,
		"Exclude files matching this glob (repeatable)")
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	coverages, err := coverage.Scan(ctx, coverage.ScanOptions{
		Path:    path,
		Exclude: flagExclude,
	})
	if err != nil {
		return fmt.Errorf("coverage scan: %w", err)
	}

	stats := complexity.Analyze([]string{path}, buildIgnoreRegex(flagExclude))

	merged := merge.Merge(coverages, stats)

	policy, err := parseMissingPolicy(flagMissing)
	if err != nil {
		return err
	}

	entries := score.Score(merged, policy)
	entries = applyFilters(entries, flagTop, flagMin)

	formatter, err := resolveFormatter(flagFormat)
	if err != nil {
		return err
	}

	opts := report.FormatOptions{
		Threshold: flagThreshold,
		Writer:    cmd.OutOrStdout(),
		BaseDir:   path,
	}

	if err := formatter.Format(entries, opts); err != nil {
		return err
	}

	if flagFailAbove {
		for _, e := range entries {
			if e.CRAP > flagThreshold {
				os.Exit(1)
			}
		}
	}

	return nil
}

func buildIgnoreRegex(exclude []string) *regexp.Regexp {
	if len(exclude) == 0 {
		return nil
	}
	parts := make([]string, len(exclude))
	for i, pat := range exclude {
		parts[i] = regexp.QuoteMeta(pat)
	}
	return regexp.MustCompile(strings.Join(parts, "|"))
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

func applyFilters(entries []score.CRAPEntry, top int, min float64) []score.CRAPEntry {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].CRAP > entries[j].CRAP
	})
	if min > 0 {
		var filtered []score.CRAPEntry
		for _, e := range entries {
			if e.CRAP >= min {
				filtered = append(filtered, e)
			}
		}
		entries = filtered
	}
	if top > 0 && top < len(entries) {
		return entries[:top]
	}
	return entries
}

func resolveFormatter(format string) (report.Formatter, error) {
	switch strings.ToLower(format) {
	case "table", "":
		return &report.TableFormatter{}, nil
	case "json":
		return &report.JSONFormatter{}, nil
	case "github":
		return &report.GithubFormatter{}, nil
	default:
		return nil, fmt.Errorf("unknown format: %s (use table, json, or github)", format)
	}
}
