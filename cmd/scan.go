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
		dir := projectDir(args)
		full := scan.Full{Report: scan.Run(dir), Risks: scan.Risks(dir), History: scan.Git(dir)}
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetEscapeHTML(false)
			enc.SetIndent("", "  ")
			_ = enc.Encode(full)
			return
		}
		scan.PrintFull(os.Stdout, full)
	},
}

func init() {
	rootCmd.AddCommand(scanCmd)
}
