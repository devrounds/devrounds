package scan

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/devrounds/devrounds/internal/envdrift"
)

// Report is the takeover report for one project folder.
type Report struct {
	Project  string   `json:"project"`
	Types    []string `json:"types"`
	Signals  []string `json:"signals"`
	Runtime  string   `json:"runtime,omitempty"`
	Entry    []string `json:"entry_points"`
	Commands []string `json:"run_commands"`
	Missing  []string `json:"missing"`
}

func firstExisting(dir string, names ...string) string {
	for _, n := range names {
		if _, err := os.Stat(filepath.Join(dir, n)); err == nil {
			return n
		}
	}
	return ""
}

func addUnique(list *[]string, v string) {
	for _, x := range *list {
		if x == v {
			return
		}
	}
	*list = append(*list, v)
}

func scanNode(dir string, r *Report) bool {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return false
	}
	r.Types = append(r.Types, "Node")
	r.Signals = append(r.Signals, "package.json")
	var p struct {
		Main    string            `json:"main"`
		Bin     any               `json:"bin"`
		Scripts map[string]string `json:"scripts"`
		Engines map[string]string `json:"engines"`
	}
	_ = json.Unmarshal(data, &p)

	mgr, install := "npm", "npm install"
	switch {
	case firstExisting(dir, "pnpm-lock.yaml") != "":
		mgr, install = "pnpm", "pnpm install"
	case firstExisting(dir, "yarn.lock") != "":
		mgr, install = "yarn", "yarn install"
	}
	for _, lock := range []string{"package-lock.json", "yarn.lock", "pnpm-lock.yaml"} {
		if firstExisting(dir, lock) != "" {
			r.Signals = append(r.Signals, lock)
		}
	}

	if f := firstExisting(dir, ".nvmrc", ".node-version"); f != "" {
		r.Runtime = "Node pinned by " + f
	} else if v := p.Engines["node"]; v != "" {
		r.Runtime = "Node " + v + " (package.json engines)"
	} else {
		r.Missing = append(r.Missing, "No pinned Node version (.nvmrc, .node-version or engines.node)")
	}

	if p.Main != "" {
		addUnique(&r.Entry, p.Main)
	}
	switch b := p.Bin.(type) {
	case string:
		addUnique(&r.Entry, b)
	case map[string]any:
		keys := make([]string, 0, len(b))
		for k := range b {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			if s, ok := b[k].(string); ok {
				addUnique(&r.Entry, s)
			}
		}
	}
	for _, f := range []string{"index.js", "server.js", "app.js", "main.js", "src/index.js", "src/index.ts", "src/server.ts", "src/main.ts"} {
		if firstExisting(dir, f) != "" {
			addUnique(&r.Entry, f)
		}
	}

	addUnique(&r.Commands, install)
	for _, name := range []string{"start", "dev", "build", "test"} {
		if p.Scripts[name] == "" {
			continue
		}
		cmd := mgr + " " + name
		if mgr == "npm" && name != "start" && name != "test" {
			cmd = "npm run " + name
		}
		addUnique(&r.Commands, cmd)
	}
	if len(p.Scripts) == 0 && p.Main != "" {
		addUnique(&r.Commands, "node "+p.Main)
	}
	return true
}

func scanPython(dir string, r *Report) bool {
	if firstExisting(dir, "pyproject.toml", "requirements.txt", "Pipfile", ".python-version") == "" {
		return false
	}
	r.Types = append(r.Types, "Python")
	for _, f := range []string{"pyproject.toml", "requirements.txt", "Pipfile", "Pipfile.lock", "poetry.lock", "uv.lock", ".python-version"} {
		if firstExisting(dir, f) != "" {
			r.Signals = append(r.Signals, f)
		}
	}
	pp, _ := os.ReadFile(filepath.Join(dir, "pyproject.toml"))
	pyproject := string(pp)

	switch {
	case strings.Contains(pyproject, "[tool.poetry]") || firstExisting(dir, "poetry.lock") != "":
		addUnique(&r.Commands, "poetry install")
	case firstExisting(dir, "uv.lock") != "":
		addUnique(&r.Commands, "uv sync")
	case firstExisting(dir, "Pipfile") != "":
		addUnique(&r.Commands, "pipenv install")
	case firstExisting(dir, "requirements.txt") != "":
		addUnique(&r.Commands, "pip install -r requirements.txt")
	case pyproject != "":
		addUnique(&r.Commands, "pip install -e .")
	}

	if firstExisting(dir, ".python-version") != "" {
		if r.Runtime == "" {
			r.Runtime = "Python pinned by .python-version"
		}
	} else if strings.Contains(pyproject, "requires-python") {
		if r.Runtime == "" {
			r.Runtime = "Python constrained by pyproject.toml (requires-python)"
		}
	} else {
		r.Missing = append(r.Missing, "No pinned Python version (.python-version or requires-python)")
	}

	for _, f := range []string{"main.py", "app.py", "manage.py", "wsgi.py", "asgi.py", "run.py", "__main__.py"} {
		if firstExisting(dir, f) != "" {
			addUnique(&r.Entry, f)
		}
	}
	for _, pattern := range []string{"*/__main__.py", "src/*/__main__.py"} {
		matches, _ := filepath.Glob(filepath.Join(dir, pattern))
		sort.Strings(matches)
		for _, m := range matches {
			rel, err := filepath.Rel(dir, m)
			if err != nil {
				continue
			}
			addUnique(&r.Entry, filepath.ToSlash(rel))
			addUnique(&r.Commands, "python -m "+filepath.Base(filepath.Dir(m)))
		}
	}
	switch {
	case firstExisting(dir, "manage.py") != "":
		addUnique(&r.Commands, "python manage.py runserver")
	case firstExisting(dir, "main.py") != "":
		addUnique(&r.Commands, "python main.py")
	case firstExisting(dir, "app.py") != "":
		addUnique(&r.Commands, "python app.py")
	}
	if info, err := os.Stat(filepath.Join(dir, "tests")); err == nil && info.IsDir() {
		addUnique(&r.Commands, "python -m pytest")
	}
	return true
}

func scanContainers(dir string, r *Report) {
	compose := firstExisting(dir, "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml")
	if compose != "" {
		r.Signals = append(r.Signals, compose)
		addUnique(&r.Commands, "docker compose up -d")
	}
	if firstExisting(dir, "Dockerfile") != "" {
		r.Signals = append(r.Signals, "Dockerfile")
		if compose == "" {
			addUnique(&r.Commands, "docker build -t "+strings.ToLower(r.Project)+" .")
		}
	}
	if firstExisting(dir, "Makefile") != "" {
		r.Signals = append(r.Signals, "Makefile")
		addUnique(&r.Commands, "make")
	}
}

func hasCI(dir string) bool {
	if firstExisting(dir, ".gitlab-ci.yml", "Jenkinsfile", ".travis.yml", "azure-pipelines.yml", ".circleci/config.yml") != "" {
		return true
	}
	matches, _ := filepath.Glob(filepath.Join(dir, ".github", "workflows", "*.y*ml"))
	return len(matches) > 0
}

func checkDocs(dir string, r *Report, hasCode bool) {
	if firstExisting(dir, "README.md", "README", "README.rst", "README.txt", "readme.md") == "" {
		r.Missing = append(r.Missing, "No README")
	}
	if firstExisting(dir, "LICENSE", "LICENSE.md", "LICENSE.txt", "COPYING") == "" {
		r.Missing = append(r.Missing, "No LICENSE file")
	}
	if hasCode {
		names, files := envdrift.ScanCode(dir)
		if files > 0 && len(names) > 0 && len(envdrift.ReadExample(dir)) == 0 && len(envdrift.ReadDocker(dir)) == 0 {
			r.Missing = append(r.Missing, fmt.Sprintf("%d environment variables are read in code but none are documented (.env.example, Docker or Compose files)", len(names)))
		}
	}
	if !hasCI(dir) {
		r.Missing = append(r.Missing, "No CI configuration found")
	}
}

// Run builds the takeover report for the project in dir.
func Run(dir string) Report {
	r := Report{
		Project:  filepath.Base(dir),
		Types:    []string{},
		Signals:  []string{},
		Entry:    []string{},
		Commands: []string{},
		Missing:  []string{},
	}
	node := scanNode(dir, &r)
	py := scanPython(dir, &r)
	scanContainers(dir, &r)
	if !node && !py {
		r.Types = append(r.Types, "Other (generic fallback)")
	}
	checkDocs(dir, &r, node || py)
	return r
}
