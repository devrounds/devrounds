package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:     "devrounds",
	Short:   "Run a round of checks on any repo or dev machine",
	Long:    "devrounds runs a local round of checks and gives one verdict on whether a project is ready to run.",
	Version: version,
	Run:     runRound,
}

// Execute runs the root command.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
