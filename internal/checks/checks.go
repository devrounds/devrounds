package checks

import (
	"os"

	"github.com/devrounds/devrounds/internal/report"
)

// Run executes all readiness checks for the project in dir.
func Run(dir string) []report.Result {
	var results []report.Result
	results = append(results, Runtime(dir)...)
	results = append(results, Binaries(dir)...)
	results = append(results, NodeInstall(dir)...)
	results = append(results, PythonEnv(dir, os.Getenv("VIRTUAL_ENV"))...)
	results = append(results, Ports(dir)...)
	results = append(results, Compose(dir)...)
	results = append(results, EnvFile(dir)...)
	return results
}
