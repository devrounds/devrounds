package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/devrounds/devrounds/internal/checks"
	"github.com/devrounds/devrounds/internal/report"
)

func runChecks(verb string, args []string) {
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
	results := checks.Run(abs)
	if !report.Print(fmt.Sprintf("%s on %s", verb, filepath.Base(abs)), results) {
		os.Exit(1)
	}
}
