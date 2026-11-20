package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/devrounds/devrounds/internal/checks"
	"github.com/devrounds/devrounds/internal/portmap"
	"github.com/spf13/cobra"
)

var mapCmd = &cobra.Command{
	Use:   "map [path]",
	Short: "Show what is listening on which port; with a path, also port conflicts for that project",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		entries, err := portmap.Listening()
		if err != nil {
			fmt.Fprintln(os.Stderr, "devrounds: could not read listening ports:", err)
			os.Exit(1)
		}
		containers, docker := portmap.Containers()
		rep := portmap.Build(entries, containers, docker)

		if len(args) == 1 {
			dir := projectDir(args)
			var expected []portmap.Expected
			for _, p := range checks.ExpectedPorts(dir) {
				expected = append(expected, portmap.Expected{Port: p.Port, Source: p.Source})
			}
			rep.Project = &portmap.ProjectInfo{
				Name:      filepath.Base(dir),
				Expected:  len(expected),
				Conflicts: portmap.Conflicts(rep, dir, expected),
			}
		}

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
