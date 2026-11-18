package scan

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// Complete is the full scan report plus dependency findings.
type Complete struct {
	Full
	Deps []string `json:"dependency_risks"`
}

// End-of-life dates from the Node.js and Python release schedules.
var nodeEOL = map[string]string{
	"14": "2023-04-30", "15": "2021-06-01", "16": "2023-09-11", "17": "2022-06-01",
	"18": "2025-04-30", "19": "2023-06-01", "20": "2026-04-30", "21": "2024-06-01",
	"22": "2027-04-30", "23": "2025-06-01", "24": "2028-04-30",
}

var pythonEOL = map[string]string{
	"3.6": "2021-12-23", "3.7": "2023-06-27", "3.8": "2024-10-07", "3.9": "2025-10-31",
	"3.10": "2026-10-31", "3.11": "2027-10-31", "3.12": "2028-10-31", "3.13": "2029-10-31",
}

var (
	nodeMajorRe = regexp.MustCompile(`\d+`)
	pyVerRe     = regexp.MustCompile(`(\d+)\.(\d+)`)
	pyNameRe    = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*`)
)

func listNames(names []string) string {
	if len(names) <= 5 {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s +%d more", strings.Join(names[:5], ", "), len(names)-5)
}

func eolMessage(lang, key, src string, table map[string]string, longPast bool) string {
	if longPast {
		return fmt.Sprintf("%s %s (pinned in %s) is long past end of life", lang, key, src)
	}
	d, ok := table[key]
	if !ok {
		return ""
	}
	t, err := time.Parse("2006-01-02", d)
	if err != nil {
		return ""
	}
	now := time.Now()
	switch {
	case now.After(t):
		return fmt.Sprintf("%s %s (pinned in %s) reached end of life on %s", lang, key, src, d)
	case t.Sub(now) < 180*24*time.Hour:
		return fmt.Sprintf("%s %s (pinned in %s) reaches end of life on %s", lang, key, src, d)
	}
	return ""
}

func runtimeEOL(dir string) []string {
	var out []string
	if f := firstExisting(dir, ".nvmrc", ".node-version"); f != "" {
		if data, err := os.ReadFile(filepath.Join(dir, f)); err == nil {
			if m := nodeMajorRe.FindString(string(data)); m != "" {
				major, _ := strconv.Atoi(m)
				if msg := eolMessage("Node", strconv.Itoa(major), f, nodeEOL, major < 14); msg != "" {
					out = append(out, msg)
				}
			}
		}
	}
	if data, err := os.ReadFile(filepath.Join(dir, ".python-version")); err == nil {
		if m := pyVerRe.FindStringSubmatch(string(data)); m != nil {
			major, _ := strconv.Atoi(m[1])
			minor, _ := strconv.Atoi(m[2])
			old := major < 3 || (major == 3 && minor < 6)
			key := fmt.Sprintf("%d.%d", major, minor)
			if msg := eolMessage("Python", key, ".python-version", pythonEOL, old); msg != "" {
				out = append(out, msg)
			}
		}
	}
	return out
}

func isURLSpec(low string) bool {
	for _, p := range []string{"git+", "git:", "git@", "http:", "https:", "github:", "gitlab:", "bitbucket:", "file:", "link:"} {
		if strings.HasPrefix(low, p) {
			return true
		}
	}
	return strings.Contains(low, "/") && !strings.HasPrefix(low, "npm:") && !strings.Contains(low, "||")
}

func nodeDeps(dir string) []string {
	data, err := os.ReadFile(filepath.Join(dir, "package.json"))
	if err != nil {
		return nil
	}
	var p struct {
		Dependencies         map[string]string `json:"dependencies"`
		DevDependencies      map[string]string `json:"devDependencies"`
		OptionalDependencies map[string]string `json:"optionalDependencies"`
	}
	if json.Unmarshal(data, &p) != nil {
		return nil
	}
	all := map[string]string{}
	for _, m := range []map[string]string{p.Dependencies, p.DevDependencies, p.OptionalDependencies} {
		for k, v := range m {
			all[k] = v
		}
	}
	names := make([]string, 0, len(all))
	for k := range all {
		names = append(names, k)
	}
	sort.Strings(names)

	var anyVer, urls []string
	for _, n := range names {
		s := strings.TrimSpace(all[n])
		low := strings.ToLower(s)
		switch {
		case s == "" || s == "*" || low == "latest" || low == "x":
			anyVer = append(anyVer, n)
		case isURLSpec(low):
			urls = append(urls, n)
		}
	}
	var out []string
	if len(anyVer) > 0 {
		out = append(out, fmt.Sprintf("Dependencies that accept any version (%d): %s", len(anyVer), listNames(anyVer)))
	}
	if len(urls) > 0 {
		out = append(out, fmt.Sprintf("Dependencies installed from a URL or git (%d): %s", len(urls), listNames(urls)))
	}
	if len(all) > 0 && firstExisting(dir, "package-lock.json", "yarn.lock", "pnpm-lock.yaml", "bun.lockb", "npm-shrinkwrap.json") == "" {
		out = append(out, "No lockfile (package-lock.json, yarn.lock or pnpm-lock.yaml), so installs are not reproducible")
	}
	return out
}

func pythonDeps(dir string) []string {
	data, err := os.ReadFile(filepath.Join(dir, "requirements.txt"))
	if err != nil {
		return nil
	}
	var none, minOnly, urls []string
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if i := strings.Index(line, " #"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		if strings.HasPrefix(line, "git+") {
			urls = append(urls, "git URL")
			continue
		}
		name := pyNameRe.FindString(line)
		if name == "" {
			continue
		}
		if strings.Contains(line, "://") {
			urls = append(urls, name)
			continue
		}
		spec := line[len(name):]
		if i := strings.Index(spec, ";"); i >= 0 {
			spec = spec[:i]
		}
		switch {
		case strings.Contains(spec, "==") || strings.Contains(spec, "~=") || strings.Contains(spec, "<"):
		case strings.Contains(spec, ">"):
			minOnly = append(minOnly, name)
		default:
			none = append(none, name)
		}
	}
	var out []string
	if len(none) > 0 {
		out = append(out, fmt.Sprintf("Requirements without a version (%d): %s", len(none), listNames(none)))
	}
	if len(minOnly) > 0 {
		out = append(out, fmt.Sprintf("Requirements with only a minimum version (%d): %s", len(minOnly), listNames(minOnly)))
	}
	if len(urls) > 0 {
		out = append(out, fmt.Sprintf("Requirements installed from a URL or git (%d): %s", len(urls), listNames(urls)))
	}
	return out
}

// Deps finds dependency risks using static rules only. It never contacts a registry.
func Deps(dir string) []string {
	out := []string{}
	out = append(out, runtimeEOL(dir)...)
	out = append(out, nodeDeps(dir)...)
	out = append(out, pythonDeps(dir)...)
	const limit = 12
	if len(out) > limit {
		extra := len(out) - limit
		out = append(out[:limit], fmt.Sprintf("... and %d more", extra))
	}
	return out
}

// PrintComplete writes the full report followed by the dependency findings.
func PrintComplete(w io.Writer, c Complete) {
	PrintFull(w, c.Full)
	if len(c.Deps) == 0 {
		fmt.Fprintf(w, "%-9s nothing obvious\n", "Deps")
		return
	}
	for i, l := range c.Deps {
		label := ""
		if i == 0 {
			label = "Deps"
		}
		fmt.Fprintf(w, "%-9s - %s\n", label, l)
	}
}
