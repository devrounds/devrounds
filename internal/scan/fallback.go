package scan

import (
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

var cmdPrefixes = []string{
	"npm ", "npx ", "yarn ", "pnpm ", "pip ", "pip3 ", "python ", "python3 ", "poetry ", "uv ",
	"make ", "docker ", "go ", "cargo ", "mvn ", "./mvnw ", "gradle ", "./gradlew ", "dotnet ",
	"bundle ", "rake ", "composer ", "php ", "ruby ", "java ", "deno ", "bun ",
}

var ciRunRe = regexp.MustCompile(`^\s*(?:-\s*)?run:\s*(\S.*)$`)

func looksLikeCommand(s string) bool {
	if s == "" || len(s) > 120 || strings.Contains(s, "${{") {
		return false
	}
	low := strings.ToLower(s)
	for _, bad := range []string{"token", "password", "passwd", "secret"} {
		if strings.Contains(low, bad) {
			return false
		}
	}
	if s == "make" {
		return true
	}
	for _, p := range cmdPrefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

func readmeCommands(dir string) []string {
	name := firstExisting(dir, "README.md", "readme.md", "README", "README.rst", "README.txt")
	if name == "" {
		return nil
	}
	data, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		return nil
	}
	var out []string
	inFence := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimRight(raw, "\r")
		trim := strings.TrimSpace(line)
		if strings.HasPrefix(trim, "```") {
			inFence = !inFence
			continue
		}
		indented := strings.HasPrefix(line, "    ") || strings.HasPrefix(line, "\t")
		if !inFence && !indented {
			continue
		}
		trim = strings.TrimPrefix(trim, "$ ")
		if i := strings.Index(trim, "  #"); i > 0 {
			trim = strings.TrimSpace(trim[:i])
		}
		if looksLikeCommand(trim) {
			out = append(out, trim)
		}
	}
	return out
}

func ciCommands(dir string) []string {
	paths, _ := filepath.Glob(filepath.Join(dir, ".github", "workflows", "*.y*ml"))
	sort.Strings(paths)
	var out []string
	for _, p := range paths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			m := ciRunRe.FindStringSubmatch(strings.TrimRight(line, "\r"))
			if m == nil {
				continue
			}
			if c := strings.TrimSpace(m[1]); looksLikeCommand(c) {
				out = append(out, c)
			}
		}
	}
	return out
}

// scanFallback adds run commands found in CI config and the README for
// projects that are not Node or Python.
func scanFallback(dir string, r *Report) {
	const limit = 8
	added := 0
	seen := map[string]bool{}
	for _, c := range r.Commands {
		seen[c] = true
	}
	add := func(cmds []string, src string) {
		for _, c := range cmds {
			if added >= limit || seen[c] {
				continue
			}
			seen[c] = true
			r.Commands = append(r.Commands, c+"   # "+src)
			added++
		}
	}
	add(ciCommands(dir), "from CI")
	add(readmeCommands(dir), "from README")
}
