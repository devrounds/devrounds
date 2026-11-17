package cmd

import "github.com/spf13/cobra"

var checkCmd = &cobra.Command{
	Use:   "check [path]",
	Short: "Readiness checks: runtime, binaries, ports, services",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runReport("Check", args, true, false)
	},
}

func init() {
	rootCmd.AddCommand(checkCmd)
}
