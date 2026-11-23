package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/devrounds/devrounds/internal/checks"
	"github.com/devrounds/devrounds/internal/portmap"
	"github.com/spf13/cobra"
)

var (
	mapHealth     bool
	mapHealthPath string
	mapMarkdown   bool
	mapOut        string
)

var mapCmd = &cobra.Command{
	Use:   "map [path]",
	Short: "Show what is listening on which port; with a path, also port conflicts for that project",
	Args:  cobra.MaximumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		if mapHealth && len(args) == 0 {
			fmt.Fprintln(os.Stderr, "devrounds: --health needs a project path, for example: devrounds map . --health")
			os.Exit(2)
		}
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
			if mapHealth {
				targets := portmap.HealthTargets(rep, expected)
				rep.Health = portmap.CheckHealth(targets, mapHealthPath, 2*time.Second)
			}
		}

		var w io.Writer = os.Stdout
		if mapOut != "" {
			f, err := os.Create(mapOut)
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
			enc.SetIndent("", "  ")
			_ = enc.Encode(rep)
		case mapMarkdown:
			home, _ := os.UserHomeDir()
			portmap.PrintMarkdown(w, rep, home)
		default:
			portmap.PrintText(w, rep)
		}
		if mapOut != "" {
			fmt.Fprintln(os.Stderr, "wrote", mapOut)
		}
	},
}

func init() {
	f := mapCmd.Flags()
	f.BoolVar(&mapHealth, "health", false, "GET the health path on the project's own listening ports (needs a project path)")
	f.StringVar(&mapHealthPath, "health-path", "/health", "path to request with --health")
	f.BoolVar(&mapMarkdown, "markdown", false, "print the report as Markdown, with your home folder shown as ~")
	f.StringVarP(&mapOut, "out", "o", "", "write the report to a file instead of the screen")
	rootCmd.AddCommand(mapCmd)
}
