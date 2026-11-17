package cmd

import "github.com/spf13/cobra"

var envCmd = &cobra.Command{
	Use:   "env [path]",
	Short: "Environment-variable drift between code, .env.example, Docker and CI",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		runReport("Env", args, false, true)
	},
}

func init() {
	rootCmd.AddCommand(envCmd)
}
