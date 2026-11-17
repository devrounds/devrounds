package envdrift

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Loc is where a variable name was seen. Line 0 means the line is unknown.
type Loc struct {
	File string
	Line int
}

func (l Loc) String() string {
	if l.Line > 0 {
		return fmt.Sprintf("%s:%d", l.File, l.Line)
	}
	return l.File
}

var nameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

var skipDirs = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "dist": true, "build": true,
	".next": true, ".nuxt": true, "out": true, "coverage": true, "venv": true,
	".venv": true, "__pycache__": true, "target": true, ".tox": true,
	".mypy_cache": true, ".idea": true, ".vscode": true, "site-packages": true,
}

// Names that are set by the OS, the runtime or CI, not by the project.
var ignore = map[string]bool{
	"PATH": true, "HOME": true, "USER": true, "USERNAME": true, "USERPROFILE": true,
	"PWD": true, "SHELL": true, "TERM": true, "TMP": true, "TEMP": true, "TMPDIR": true,
	"LANG": true, "CI": true, "NODE_ENV": true, "MODE": true, "DEV": true, "PROD": true,
	"SSR": true, "BASE_URL": true,
}

func ignored(name string) bool {
	return ignore[name] || strings.HasPrefix(name, "GITHUB_") || strings.HasPrefix(name, "RUNNER_")
}

var codePatterns = []*regexp.Regexp{
	regexp.MustCompile(`process\.env\.([A-Za-z_][A-Za-z0-9_]*)`),
	regexp.MustCompile(`process\.env\[\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]\s*\]`),
	regexp.MustCompile(`import\.meta\.env\.([A-Za-z_][A-Za-z0-9_]*)`),
	regexp.MustCompile(`os\.environ\[\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]\s*\]`),
	regexp.MustCompile(`os\.environ\.get\(\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]`),
	regexp.MustCompile(`os\.getenv\(\s*['"]([A-Za-z_][A-Za-z0-9_]*)['"]`),
}

// const { A, B = 1 } = process.env
var destructureRe = regexp.MustCompile(`\{([^}]*)\}\s*=\s*process\.env(?:[^.\w\[]|$)`)

// ScanCode finds every env var name read in Node and Python source files.
// It also returns how many source files it looked at.
func ScanCode(dir string) (map[string][]Loc, int) {
	found := map[string][]Loc{}
	files := 0
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != dir && skipDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		switch strings.ToLower(filepath.Ext(path)) {
		case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs", ".py":
		default:
			return nil
		}
		info, err := d.Info()
		if err != nil || info.Size() > 1<<20 {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}
		rel, err := filepath.Rel(dir, path)
		if err != nil {
			return nil
		}
		files++
		scanFile(found, filepath.ToSlash(rel), string(data))
		return nil
	})
	return found, files
}

func scanFile(found map[string][]Loc, rel, src string) {
	var newlines []int
	for i := 0; i < len(src); i++ {
		if src[i] == '\n' {
			newlines = append(newlines, i)
		}
	}
	add := func(name string, off int) {
		if ignored(name) {
			return
		}
		found[name] = append(found[name], Loc{rel, sort.SearchInts(newlines, off) + 1})
	}
	for _, re := range codePatterns {
		for _, m := range re.FindAllStringSubmatchIndex(src, -1) {
			add(src[m[2]:m[3]], m[0])
		}
	}
	for _, m := range destructureRe.FindAllStringSubmatchIndex(src, -1) {
		for _, part := range strings.Split(src[m[2]:m[3]], ",") {
			part = strings.TrimSpace(part)
			if part == "" || strings.HasPrefix(part, "...") {
				continue
			}
			if i := strings.IndexAny(part, ":="); i >= 0 {
				part = strings.TrimSpace(part[:i])
			}
			if nameRe.MatchString(part) {
				add(part, m[0])
			}
		}
	}
}
