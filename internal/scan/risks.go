package scan

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Full is the scan report plus the risk findings.
type Full struct {
	Report
	Risks []string `json:"risks"`
}

var riskSkip = map[string]bool{
	".git": true, "node_modules": true, "vendor": true, "dist": true, "build": true,
	".venv": true, "venv": true, "__pycache__": true, "target": true, ".next": true,
}

var debugSkip = map[string]bool{
	"tests": true, "test": true, "__tests__": true, "spec": true,
	"node_modules": true, "vendor": true, "dist": true, "build": true,
}

var secretNames = map[string]bool{
	".pypirc": true, ".netrc": true, ".htpasswd": true, "credentials.json": true,
	"secrets.json": true, "secrets.yml": true, "secrets.yaml": true, "terraform.tfstate": true,
	"id_rsa": true, "id_dsa": true, "id_ecdsa": true, "id_ed25519": true,
}

var secretExts = map[string]bool{
	".pem": true, ".key": true, ".p12": true, ".pfx": true, ".jks": true, ".keystore": true,
}

var envSafeSuffix = []string{".example", ".sample", ".template", ".dist"}

var debugPyRe = regexp.MustCompile(`^\s*DEBUG\s*=\s*True\b|\bdebug\s*=\s*True\b`)
var debugJSRe = regexp.MustCompile(`\bdebug\s*:\s*true\b`)

// secretFile matches by file name only. File contents are never read.
func secretFile(rel string) bool {
	base := strings.ToLower(path.Base(rel))
	if secretNames[base] || secretExts[path.Ext(base)] {
		return true
	}
	if base == ".env" || strings.HasPrefix(base, ".env.") {
		for _, s := range envSafeSuffix {
			if strings.HasSuffix(base, s) {
				return false
			}
		}
		return true
	}
	return false
}

// trackedFiles lists the files git tracks in dir. ok is false outside a git repo.
func trackedFiles(dir string) ([]string, bool) {
	out, err := exec.Command("git", "-C", dir, "ls-files", "-z").Output()
	if err != nil {
		return nil, false
	}
	var files []string
	for _, f := range bytes.Split(out, []byte{0}) {
		if len(f) > 0 {
			files = append(files, string(f))
		}
	}
	return files, true
}

func walkFiles(dir string) []string {
	var files []string
	_ = filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if p != dir && riskSkip[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}
		if rel, err := filepath.Rel(dir, p); err == nil {
			files = append(files, filepath.ToSlash(rel))
		}
		return nil
	})
	return files
}

func inSkippedDir(rel string) bool {
	parts := strings.Split(rel, "/")
	for _, p := range parts[:len(parts)-1] {
		if debugSkip[strings.ToLower(p)] {
			return true
		}
	}
	return false
}

func debugFlags(dir string, files []string) []string {
	var out []string
	for _, f := range files {
		var re *regexp.Regexp
		switch strings.ToLower(path.Ext(f)) {
		case ".py":
			re = debugPyRe
		case ".js", ".jsx", ".ts", ".tsx", ".mjs", ".cjs":
			re = debugJSRe
		default:
			continue
		}
		if inSkippedDir(f) {
			continue
		}
		full := filepath.Join(dir, filepath.FromSlash(f))
		info, err := os.Stat(full)
		if err != nil || info.Size() > 1<<20 {
			continue
		}
		data, err := os.ReadFile(full)
		if err != nil {
			continue
		}
		for i, line := range strings.Split(string(data), "\n") {
			if re.MatchString(line) {
				out = append(out, fmt.Sprintf("debug flag on: %s:%d", f, i+1))
				break
			}
		}
	}
	return out
}

// Risks finds secret-looking files by name and debug flags left on.
func Risks(dir string) []string {
	out := []string{}
	files, isGit := trackedFiles(dir)
	if !isGit {
		files = walkFiles(dir)
	}
	var secrets []string
	for _, f := range files {
		if secretFile(f) {
			secrets = append(secrets, f)
		}
	}
	sort.Strings(secrets)
	for _, f := range secrets {
		if isGit {
			out = append(out, "possible secret file committed: "+f)
		} else {
			out = append(out, "possible secret file present: "+f+" (not a git repo, cannot tell if committed)")
		}
	}
	out = append(out, debugFlags(dir, files)...)
	const limit = 12
	if len(out) > limit {
		extra := len(out) - limit
		out = append(out[:limit], fmt.Sprintf("... and %d more", extra))
	}
	return out
}

// PrintFull writes the scan report followed by the risk findings.
func PrintFull(w io.Writer, f Full) {
	PrintText(w, f.Report)
	if len(f.Risks) == 0 {
		fmt.Fprintf(w, "%-9s nothing obvious\n", "Risks")
		return
	}
	for i, l := range f.Risks {
		label := ""
		if i == 0 {
			label = "Risks"
		}
		fmt.Fprintf(w, "%-9s - %s\n", label, l)
	}
}
