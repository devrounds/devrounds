package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/devrounds/devrounds/internal/portmap"
	"github.com/spf13/cobra"
)

var mapCmd = &cobra.Command{
	Use:   "map",
	Short: "Show what is listening on which port, and from which folder",
	Args:  cobra.NoArgs,
	Run: func(cmd *cobra.Command, args []string) {
		entries, err := portmap.Listening()
		if err != nil {
			fmt.Fprintln(os.Stderr, "devrounds: could not read listening ports:", err)
			os.Exit(1)
		}
		if jsonOutput {
			if entries == nil {
				entries = []portmap.Entry{}
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(entries)
			return
		}
		tw := tabwriter.NewWriter(os.Stdout, 0, 8, 2, ' ', 0)
		fmt.Fprintln(tw, "PORT\tPROCESS\tPID\tFOLDER")
		for _, e := range entries {
			name, folder := e.Process, e.Folder
			if name == "" {
				name = "-"
			}
			if folder == "" {
				folder = "-"
			}
			fmt.Fprintf(tw, "%d\t%s\t%d\t%s\n", e.Port, name, e.PID, folder)
		}
		tw.Flush()
	},
}

func init() {
	rootCmd.AddCommand(mapCmd)
}
