package cmd

import "github.com/spf13/cobra"

var checkCmd = &cobra.Command{
	Use:   "check [path]",
	Short: "Readiness checks: runtime, binaries, ports, services",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runChecks("Check", args)
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
