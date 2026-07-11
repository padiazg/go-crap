package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/padiazg/go-crap/internal/report"
	"github.com/padiazg/go-crap/internal/scan"
	"github.com/padiazg/go-crap/pkg/logger"
	"github.com/padiazg/go-crap/pkg/slogger"
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
	flagVerbose   bool
	flagOutput    string
	flagMutation  string
	flagDetailed  bool
	flagTimeout   time.Duration
	flagCoverProf string

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
		"Exclude files matching this regex (repeatable). Use . for any character, .* for any path depth. e.g. '.*_test\\.go' to exclude all test files, 'pb/.*\\.go' to exclude protobuf files")
	scanCmd.Flags().BoolVar(&flagVerbose, "verbose", false,
		"Enable verbose (debug-level) logging")
	scanCmd.Flags().StringVarP(&flagOutput, "output", "o", "",
		"Output file path (default: stdout)")
	scanCmd.Flags().StringVar(&flagMutation, "mutation-report", "",
		"Path to gremlins JSON mutation report to validate coverage reliability")
	scanCmd.Flags().BoolVar(&flagDetailed, "detailed", false,
		"Include mutation failure details in report output")
	scanCmd.Flags().StringVar(&flagCoverProf, "coverage-profile", "",
		`Use an existing coverage profile (as produced by "go test -coverprofile") instead of running go test`)
	scanCmd.Flags().DurationVar(&flagTimeout, "timeout", 10*time.Minute,
		"Timeout for the full scan (e.g. 30s, 5m, 1h30m)")
	scanCmd.Flags().StringVar(&flagCoverProf, "coverage-profile", "",
		`Use an existing coverage profile (as produced by "go test -coverprofile") instead of running go test`)
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
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

	err = output(entries, outputConfig{
		path:      path,
		writer:    cmd.OutOrStdout(),
		output:    flagOutput,
		format:    flagFormat,
		threshold: flagThreshold,
		detailed:  flagDetailed,
	})
	if err != nil {
		return err
	}

	if flagFailAbove && entries.ThresholdExceeded(flagThreshold) {
		return scan.ErrThresholdExceeded
	}

	return nil
}

type outputConfig struct {
	path      string
	writer    io.Writer
	output    string
	format    string
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

	formatter, err := resolveFormatter(config.format)
	if err != nil {
		return err
	}

	opts := report.FormatOptions{
		Threshold: config.threshold,
		Writer:    config.writer,
		BaseDir:   config.path,
		Detailed:  config.detailed,
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
