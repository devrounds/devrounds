package cmd

import (
	"encoding/json"
	"os"

	"github.com/devrounds/devrounds/internal/scan"
	"github.com/spf13/cobra"
)

var scanCmd = &cobra.Command{
	Use:   "scan [path]",
	Short: "Takeover report for an unfamiliar repo",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		r := scan.Run(projectDir(args))
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetEscapeHTML(false)
			enc.SetIndent("", "  ")
			_ = enc.Encode(r)
			return
		}
		scan.PrintText(os.Stdout, r)
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
