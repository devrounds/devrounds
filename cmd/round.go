package cmd

import "github.com/spf13/cobra"

var roundCmd = &cobra.Command{
	Use:   "round [path]",
	Short: "Run check and env in one pass and print one verdict",
	Args:  cobra.MaximumNArgs(1),
	Run:   runRound,
}

func runRound(cmd *cobra.Command, args []string) {
	// env checks will join here later
	runChecks("Round", args)
}

func init() {
	rootCmd.AddCommand(roundCmd)
}
