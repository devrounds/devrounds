package envdrift

import (
	"bufio"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var exampleFiles = []string{".env.example", ".env.sample", ".env.template"}
var composeFiles = []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"}

var envLineRe = regexp.MustCompile(`^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=`)
var composeVarRe = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)`)
var dockerEnvRe = regexp.MustCompile(`^\s*(?:ENV|ARG)\s+([A-Za-z_][A-Za-z0-9_]*)`)
var ciRefRe = regexp.MustCompile(`(?:secrets|vars)\.([A-Za-z_][A-Za-z0-9_]*)`)

func addName(out map[string][]Loc, name string, loc Loc) {
	if !nameRe.MatchString(name) || ignored(name) {
		return
	}
	appendLoc(out, name, loc)
}

// ReadExample reads names (never values) from .env.example style files.
func ReadExample(dir string) map[string][]Loc {
	out := map[string][]Loc{}
	for _, f := range exampleFiles {
		file, err := os.Open(filepath.Join(dir, f))
		if err != nil {
			continue
		}
		sc := bufio.NewScanner(file)
		n := 0
		for sc.Scan() {
			n++
			if m := envLineRe.FindStringSubmatch(sc.Text()); m != nil {
				addName(out, m[1], Loc{f, n})
			}
		}
		file.Close()
	}
	return out
}

func composeEnv(dir, file string, out map[string][]Loc) {
	data, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return
	}
	var cf struct {
		Services map[string]struct {
			Environment any `yaml:"environment"`
		} `yaml:"services"`
	}
	if yaml.Unmarshal(data, &cf) == nil {
		for _, svc := range cf.Services {
			switch env := svc.Environment.(type) {
			case []any:
				for _, item := range env {
					if s, ok := item.(string); ok {
						name := s
						if i := strings.Index(s, "="); i >= 0 {
							name = s[:i]
						}
						addName(out, strings.TrimSpace(name), Loc{file, 0})
					}
				}
			case map[string]any:
				for k := range env {
					addName(out, k, Loc{file, 0})
				}
			}
		}
	}
	for i, line := range strings.Split(string(data), "\n") {
		for _, m := range composeVarRe.FindAllStringSubmatch(line, -1) {
			addName(out, m[1], Loc{file, i + 1})
		}
	}
}

// ReadDocker reads names from Compose files and the Dockerfile.
func ReadDocker(dir string) map[string][]Loc {
	out := map[string][]Loc{}
	for _, f := range composeFiles {
		composeEnv(dir, f, out)
	}
	if data, err := os.ReadFile(filepath.Join(dir, "Dockerfile")); err == nil {
		for i, line := range strings.Split(string(data), "\n") {
			if m := dockerEnvRe.FindStringSubmatch(line); m != nil {
				addName(out, m[1], Loc{"Dockerfile", i + 1})
			}
		}
	}
	return out
}

func collectEnvKeys(v any, file string, out map[string][]Loc) {
	switch t := v.(type) {
	case map[string]any:
		for k, val := range t {
			if k == "env" {
				if m, ok := val.(map[string]any); ok {
					for name := range m {
						addName(out, name, Loc{file, 0})
					}
				}
			}
			collectEnvKeys(val, file, out)
		}
	case []any:
		for _, item := range t {
			collectEnvKeys(item, file, out)
		}
	}
}

// ReadCI reads names from GitHub Actions workflows: env keys and secrets/vars references.
func ReadCI(dir string) map[string][]Loc {
	out := map[string][]Loc{}
	paths, _ := filepath.Glob(filepath.Join(dir, ".github", "workflows", "*.y*ml"))
	for _, path := range paths {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		for i, line := range strings.Split(string(data), "\n") {
			for _, m := range ciRefRe.FindAllStringSubmatch(line, -1) {
				addName(out, m[1], Loc{rel, i + 1})
			}
		}
		var doc any
		if yaml.Unmarshal(data, &doc) == nil {
			collectEnvKeys(doc, rel, out)
		}
	}
	return out
}
