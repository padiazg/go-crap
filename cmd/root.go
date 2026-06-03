package cmd

import (
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "go-crap",
	Short: "CRAP score calculator for Go projects",
	Long:  `go-crap calculates the CRAP score (cyclomatic complexity x coverage) for Go projects.`,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
