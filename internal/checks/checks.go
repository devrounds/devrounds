package checks

import "github.com/devrounds/devrounds/internal/report"

// Run executes all readiness checks for the project in dir.
func Run(dir string) []report.Result {
	var results []report.Result
	results = append(results, Runtime(dir)...)
	results = append(results, Binaries(dir)...)
	return results
}
