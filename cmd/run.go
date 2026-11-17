package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/devrounds/devrounds/internal/checks"
	"github.com/devrounds/devrounds/internal/envdrift"
	"github.com/devrounds/devrounds/internal/report"
)

var jsonOutput bool

func init() {
	rootCmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, "print the report as JSON")
}

func projectDir(args []string) string {
	dir := "."
	if len(args) > 0 {
		dir = args[0]
	}
	abs, err := filepath.Abs(dir)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(2)
	}
	if info, err := os.Stat(abs); err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "not a directory: %s\n", abs)
		os.Exit(2)
	}
	return abs
}

func runReport(verb string, args []string, withChecks, withEnv bool) {
	dir := projectDir(args)
	var results []report.Result
	if withChecks {
		results = append(results, checks.Run(dir)...)
	}
	if withEnv {
		results = append(results, envdrift.Run(dir)...)
	}
	title := fmt.Sprintf("%s on %s", verb, filepath.Base(dir))
	var ok bool
	if jsonOutput {
		ok = report.PrintJSON(title, results)
	} else {
		ok = report.Print(title, results)
	}
	if !ok {
		os.Exit(1)
	}
}
