package checks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/devrounds/devrounds/internal/report"
)

func exists(dir string, names ...string) bool {
	for _, n := range names {
		if _, err := os.Stat(filepath.Join(dir, n)); err == nil {
			return true
		}
	}
	return false
}

func found(bin string) bool {
	if _, err := exec.LookPath(bin); err == nil {
		return true
	}
	if bin == "python" {
		_, err := exec.LookPath("python3")
		return err == nil
	}
	return false
}

// Binaries checks that the tools this project appears to need are installed.
func Binaries(dir string) []report.Result {
	needed := []string{"git"}
	if exists(dir, "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml", "Dockerfile") {
		needed = append(needed, "docker")
	}
	if exists(dir, "package.json") {
		needed = append(needed, "node", "npm")
	}
	if exists(dir, "pyproject.toml", "requirements.txt", "Pipfile") {
		needed = append(needed, "python")
	}
	var missing []string
	for _, b := range needed {
		if !found(b) {
			missing = append(missing, b)
		}
	}
	if len(missing) > 0 {
		return []report.Result{{
			Status:  report.Fail,
			Name:    "Binaries",
			Message: "missing: " + strings.Join(missing, ", "),
		}}
	}
	return []report.Result{{
		Status:  report.OK,
		Name:    "Binaries",
		Message: fmt.Sprintf("found: %s", strings.Join(needed, ", ")),
	}}
}
