package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/devrounds/devrounds/internal/scan"
	"github.com/spf13/cobra"
)

var (
	scanMarkdown bool
	scanOut      string
)

var scanCmd = &cobra.Command{
	Use:   "scan [path]",
	Short: "Takeover report for an unfamiliar repo",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		dir := projectDir(args)
		full := scan.Full{Report: scan.Run(dir), Risks: scan.Risks(dir), History: scan.Git(dir)}
		all := scan.Complete{Full: full, Deps: scan.Deps(dir)}
		brief := scan.Brief(all)

		var w io.Writer = os.Stdout
		if scanOut != "" {
			f, err := os.Create(scanOut)
			if err != nil {
				fmt.Fprintln(os.Stderr, "devrounds:", err)
				os.Exit(1)
			}
			defer f.Close()
			w = f
		}

		switch {
		case jsonOutput:
			enc := json.NewEncoder(w)
			enc.SetEscapeHTML(false)
			enc.SetIndent("", "  ")
			_ = enc.Encode(brief)
		case scanMarkdown:
			scan.PrintMarkdown(w, brief)
		default:
			scan.PrintBriefing(w, brief)
		}
		if scanOut != "" {
			fmt.Fprintln(os.Stderr, "wrote", scanOut)
		}
	},
}

func init() {
	scanCmd.Flags().BoolVar(&scanMarkdown, "markdown", false, "print the report as Markdown")
	scanCmd.Flags().StringVarP(&scanOut, "out", "o", "", "write the report to a file instead of the screen")
	rootCmd.AddCommand(scanCmd)
}
