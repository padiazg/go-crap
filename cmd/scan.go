package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/padiazg/go-crap/internal/report"
	"github.com/padiazg/go-crap/internal/scan"
	"github.com/padiazg/go-crap/internal/score"
	"github.com/padiazg/go-crap/pkg/logger"
	"github.com/padiazg/go-crap/pkg/slogger"
	"github.com/spf13/cobra"
)

var (
	flagThreshold               float64
	flagFailAbove               bool
	flagFormat                  string
	flagTop                     int
	flagMin                     float64
	flagMissing                 string
	flagExclude                 []string
	flagIncludeTests            bool
	flagVerbose                 bool
	flagOutput                  string
	flagMutation                string
	flagDetailed                bool
	flagTimeout                 time.Duration
	flagCoverProf               string
	flagBaseline                string
	flagFailRegression          bool
	flagFailRegressionThreshold float64

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
		"Output format: table|json|github|sarif|pr-comment")
	scanCmd.Flags().IntVar(&flagTop, "top", 0,
		"Show only the N worst offenders (0 = all)")
	scanCmd.Flags().Float64Var(&flagMin, "min", 0,
		"Hide entries below this score")
	scanCmd.Flags().StringVar(&flagMissing, "missing", "pessimistic",
		"Policy for functions without coverage: pessimistic|optimistic|skip")
	scanCmd.Flags().StringArrayVar(&flagExclude, "exclude", nil,
		"Exclude files matching this regex (repeatable). Use . for any character, .* for any path depth. _test.go files are excluded by default. e.g. 'pb/.*\\.go' to exclude protobuf files")
	scanCmd.Flags().BoolVar(&flagIncludeTests, "include-tests", false,
		"Include _test.go files in analysis (default: excluded)")
	scanCmd.Flags().BoolVar(&flagVerbose, "verbose", false,
		"Enable verbose (debug-level) logging")
	scanCmd.Flags().StringVarP(&flagOutput, "output", "o", "",
		"Output file path (default: stdout)")
	scanCmd.Flags().StringVar(&flagMutation, "mutation-report", "",
		"Path to gremlins JSON mutation report to validate coverage reliability")
	scanCmd.Flags().BoolVar(&flagDetailed, "detailed", false,
		"Include mutation failure details in report output")
	scanCmd.Flags().DurationVar(&flagTimeout, "timeout", 10*time.Minute,
		"Timeout for the full scan (e.g. 30s, 5m, 1h30m)")
	scanCmd.Flags().StringVar(&flagCoverProf, "coverage-profile", "",
		`Use an existing coverage profile (as produced by "go test -coverprofile") instead of running go test`)
	scanCmd.Flags().StringVar(&flagBaseline, "baseline", "",
		"Path to a previous JSON report to use as baseline for comparison")
	scanCmd.Flags().BoolVar(&flagFailRegression, "fail-regression", false,
		"Exit with code 1 if any function's CRAP score regressed compared to baseline")
	scanCmd.Flags().Float64Var(&flagFailRegressionThreshold, "fail-regression-threshold", 0.01,
		"Minimum delta to consider a regression when comparing against baseline")
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	if flagFailRegression && flagBaseline == "" {
		return fmt.Errorf("--fail-regression requires --baseline")
	}

	logLevel := "error"
	if flagVerbose {
		logLevel = "debug"
	}
	l := slogger.New(&logger.Config{
		Level:  logLevel,
		Format: "text",
	})
	lp := &l

	entries, err := scan.Scan(&scan.Options{
		Exclude:         flagExclude,
		IncludeTests:    flagIncludeTests,
		Path:            path,
		Missing:         flagMissing,
		Top:             flagTop,
		Min:             flagMin,
		Logger:          lp,
		MutationReport:  flagMutation,
		Timeout:         flagTimeout,
		CoverageProfile: flagCoverProf,
	})
	if err != nil {
		return err
	}

	var baseline *report.Baseline
	if flagBaseline != "" {
		baseline, err = report.LoadBaseline(flagBaseline)
		if err != nil {
			return err
		}
	}

	err = output(entries, outputConfig{
		path:      path,
		writer:    cmd.OutOrStdout(),
		output:    flagOutput,
		format:    flagFormat,
		threshold: flagThreshold,
		detailed:  flagDetailed,
		baseline:  baseline,
	})
	if err != nil {
		return err
	}

	if flagFailAbove && entries.ThresholdExceeded(flagThreshold) {
		return scan.ErrThresholdExceeded
	}

	if flagFailRegression && baseline != nil {
		regressions := report.FindRegressions(entries.List, flagFailRegressionThreshold)
		if len(regressions) > 0 {
			return fmtRegressionError(cmd.OutOrStderr(), regressions, baseline.Summary)
		}
	}

	return nil
}

func fmtRegressionError(w io.Writer, regressions []score.CRAPEntry, baselineSummary report.Summary) error {
	fmt.Fprintln(w, "CRAP regression detected:")
	for _, e := range regressions {
		fmt.Fprintf(w, "  %s:%d %s: %.2f -> %.2f (+%.2f)\n",
			e.File, e.Line, e.FuncName,
			e.BaselineCRAP, e.EffectiveCRAP,
			e.EffectiveCRAP-e.BaselineCRAP)
	}
	fmt.Fprintf(w, "Combined CRAP: %+0.2f vs baseline\n", baselineSummary.Combined)
	return scan.ErrRegression
}

type outputConfig struct {
	writer    io.Writer
	baseline  *report.Baseline
	format    string
	output    string
	path      string
	threshold float64
	detailed  bool
}

func output(entries *scan.Entries, config outputConfig) error {
	if config.output != "" {
		f, err := os.Create(config.output)
		if err != nil {
			return fmt.Errorf("output: %w", err)
		}

		defer f.Close()

		config.writer = f
	}

	var summary report.Summary
	var baselineSummary *report.Baseline
	if config.baseline != nil {
		for i := range entries.List {
			entries.List[i].File = report.RelativizePath(entries.List[i].File, config.path)
		}
		entries.List = report.AnnotateWithBaseline(entries.List, config.baseline)
		summary = report.ComputeSummaryWithBaseline(entries, config.threshold, config.baseline)
		baselineSummary = config.baseline
	} else {
		summary = report.ComputeSummary(entries, config.threshold)
	}

	formatter, err := resolveFormatter(config.format)
	if err != nil {
		return err
	}

	opts := report.FormatOptions{
		Threshold: config.threshold,
		Writer:    config.writer,
		BaseDir:   config.path,
		Detailed:  config.detailed,
		Summary:   &summary,
		Baseline:  baselineSummary,
	}

	if err := formatter.Format(entries, opts); err != nil {
		return err
	}

	return nil
}

func resolveFormatter(format string) (report.Formatter, error) {
	switch strings.ToLower(format) {
	case "table", "":
		return &report.TableFormatter{}, nil
	case "json":
		return report.NewJSONFormatter(), nil
	case "github":
		return &report.GithubFormatter{}, nil
	case "sarif":
		return &report.SARIFFormatter{}, nil
	case "pr-comment":
		return &report.PRCommentFormatter{}, nil
	default:
		return nil, fmt.Errorf("unknown format: %s (use table, json, github, sarif, or pr-comment)", format)
	}
}
