package cmd

import (
	"encoding/json"
	"fmt"
	"os"

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
		containers, docker := portmap.Containers()
		rep := portmap.Build(entries, containers, docker)
		if jsonOutput {
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			_ = enc.Encode(rep)
			return
		}
		portmap.PrintText(os.Stdout, rep)
	},
}

func init() {
	rootCmd.AddCommand(mapCmd)
}
