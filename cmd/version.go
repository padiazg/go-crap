package cmd

import (
	"fmt"

	"github.com/padiazg/go-crap/pkg/version"
	"github.com/spf13/cobra"
)

// versionCmd represents the version command
var (
	versionCmd = &cobra.Command{
		Use:   "version",
		Short: "Shows go-crap version",
		Run:   runVersion,
	}
)

func runVersion(cmd *cobra.Command, args []string) {
	simple, _ := cmd.Flags().GetBool("simple")
	if simple {
		fmt.Printf("%s", version.CurrentVersion().Version)
		return
	}

	version.Splash()

}

func init() {
	rootCmd.AddCommand(versionCmd)
	versionCmd.Flags().BoolP("simple", "s", false, "Prints only the version, useful for scripting")
}
