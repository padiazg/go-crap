package cmd

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/padiazg/go-crap/internal/report"
	"github.com/padiazg/go-crap/internal/scan"
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
		"Exclude files matching this regex (repeatable). Use . for any character, .* for any path depth. e.g. '.*_test\\.go' to exclude all test files, 'pb/.*\\.go' to exclude protobuf files")
	rootCmd.AddCommand(scanCmd)
}

func runScan(cmd *cobra.Command, args []string) error {
	path := "."
	if len(args) > 0 {
		path = args[0]
	}

	entries, err := scan.Scan(&scan.Options{
		Exclude: flagExclude,
		Path:    path,
		Missing: flagMissing,
		Top:     flagTop,
		Min:     flagMin,
	})
	if err != nil {
		return err
	}

	err = output(path, entries, cmd.OutOrStdout())
	if err != nil {
		return err
	}

	if flagFailAbove && entries.ThresholdExeeded(flagThreshold) {
		os.Exit(1)
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
	default:
		return nil, fmt.Errorf("unknown format: %s (use table, json, or github)", format)
	}
}

func output(path string, entries *score.EntryList, writter io.Writer) error {
	formatter, err := resolveFormatter(flagFormat)
	if err != nil {
		return err
	}

	opts := report.FormatOptions{
		Threshold: flagThreshold,
		Writer:    writter,
		BaseDir:   path,
	}

	if err := formatter.Format(entries, opts); err != nil {
		return err
	}

	return nil
}
