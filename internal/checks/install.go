package checks

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/devrounds/devrounds/internal/report"
)

func nodeHasDeps(dir string) bool {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return false
	}
	var p struct {
		Dependencies         map[string]string `json:"dependencies"`
		DevDependencies      map[string]string `json:"devDependencies"`
		OptionalDependencies map[string]string `json:"optionalDependencies"`
	}
	if json.Unmarshal(data, &p) != nil {
		return false
	}
	return len(p.Dependencies)+len(p.DevDependencies)+len(p.OptionalDependencies) > 0
}

// nodeManager picks the package manager from the lockfile. npm is the default.
func nodeManager(dir string) (string, string) {
	switch {
	case exists(dir, "pnpm-lock.yaml"):
		return "pnpm", "pnpm-lock.yaml"
	case exists(dir, "yarn.lock"):
		return "yarn", "yarn.lock"
	case exists(dir, "bun.lockb", "bun.lock"):
		return "bun", "bun.lockb"
	case exists(dir, "package-lock.json", "npm-shrinkwrap.json"):
		return "npm", "package-lock.json"
	}
	return "npm", ""
}

// NodeInstall checks that a Node project's dependencies are installed.
func NodeInstall(dir string) []report.Result {
	if !nodeHasDeps(dir) {
		return nil
	}
	mgr, lock := nodeManager(dir)
	var out []report.Result
	if mgr != "npm" {
		if _, err := exec.LookPath(mgr); err != nil {
			out = append(out, report.Result{
				Status:  report.Fail,
				Name:    "Packages",
				Message: fmt.Sprintf("%s is required by %s but was not found on PATH", mgr, lock),
			})
		}
	}
	entries, err := os.ReadDir(filepath.Join(dir, "node_modules"))
	switch {
	case err == nil && len(entries) > 0:
		out = append(out, report.Result{Status: report.OK, Name: "Packages", Message: mgr + ": node_modules present"})
	case exists(dir, ".pnp.cjs", ".pnp.js"):
		out = append(out, report.Result{Status: report.OK, Name: "Packages", Message: mgr + ": Plug'n'Play install present"})
	default:
		out = append(out, report.Result{
			Status:  report.Fail,
			Name:    "Packages",
			Message: "node_modules is missing, dependencies are not installed",
			Next:    mgr + " install",
		})
	}
	return out
}

// PythonEnv checks for a virtual environment. It cannot tell which packages are installed.
func PythonEnv(dir, activeVenv string) []report.Result {
	if !exists(dir, "requirements.txt", "pyproject.toml", "Pipfile", "setup.py") {
		return nil
	}
	tool, lock := "", ""
	switch {
	case exists(dir, "poetry.lock"):
		tool, lock = "poetry", "poetry.lock"
	case exists(dir, "Pipfile.lock", "Pipfile"):
		tool, lock = "pipenv", "Pipfile"
	case exists(dir, "uv.lock"):
		tool, lock = "uv", "uv.lock"
	}
	var out []report.Result
	if tool != "" {
		if _, err := exec.LookPath(tool); err != nil {
			out = append(out, report.Result{
				Status:  report.Fail,
				Name:    "Packages",
				Message: fmt.Sprintf("%s is required by %s but was not found on PATH", tool, lock),
			})
		}
	}
	if tool == "poetry" || tool == "pipenv" {
		out = append(out, report.Result{
			Status:  report.OK,
			Name:    "Packages",
			Message: "Python: environment is managed by " + tool + ", install status not checked",
		})
		return out
	}
	for _, d := range []string{".venv", "venv", "env"} {
		if exists(dir, d+"/pyvenv.cfg") {
			return append(out, report.Result{Status: report.OK, Name: "Packages", Message: "Python: virtual environment found (" + d + ")"})
		}
	}
	if activeVenv != "" {
		return append(out, report.Result{
			Status:  report.OK,
			Name:    "Packages",
			Message: "Python: virtual environment active (" + filepath.Base(activeVenv) + ")",
		})
	}
	next := "python -m venv .venv"
	if tool == "uv" {
		next = "uv sync"
	}
	return append(out, report.Result{
		Status:  report.Warn,
		Name:    "Packages",
		Message: "Python: no virtual environment (.venv or venv) in the project and none active",
		Next:    next,
	})
}
