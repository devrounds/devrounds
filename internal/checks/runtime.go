package checks

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/devrounds/devrounds/internal/report"
)

var versionRe = regexp.MustCompile(`\d+(\.\d+)*`)

func firstVersion(s string) string {
	return versionRe.FindString(s)
}

// readPin returns the first existing version file and its first line.
func readPin(dir string, files ...string) (string, string) {
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			continue
		}
		v := strings.TrimSpace(string(data))
		if i := strings.IndexAny(v, "\r\n"); i >= 0 {
			v = strings.TrimSpace(v[:i])
		}
		if v != "" {
			return f, v
		}
	}
	return "", ""
}

// matches reports whether installed starts with every part of required.
func matches(required, installed string) bool {
	req := strings.Split(required, ".")
	inst := strings.Split(installed, ".")
	if len(req) > len(inst) {
		return false
	}
	for i := range req {
		if req[i] != inst[i] {
			return false
		}
	}
	return true
}

func compare(lang, file, pin string, cmds [][]string) report.Result {
	r := report.Result{Name: "Runtime"}
	want := firstVersion(pin)
	if want == "" {
		r.Status = report.Warn
		r.Message = fmt.Sprintf("%s version %q in %s cannot be compared automatically", lang, pin, file)
		return r
	}
	have := ""
	for _, c := range cmds {
		out, err := exec.Command(c[0], c[1:]...).CombinedOutput()
		if err != nil {
			continue
		}
		if v := firstVersion(string(out)); v != "" {
			have = v
			break
		}
	}
	switch {
	case have == "":
		r.Status = report.Fail
		r.Message = fmt.Sprintf("%s %s required by %s, but %s is not installed", lang, want, file, lang)
	case matches(want, have):
		r.Status = report.OK
		r.Message = fmt.Sprintf("%s %s matches %s", lang, have, file)
	default:
		r.Status = report.Fail
		r.Message = fmt.Sprintf("%s %s installed, %s wants %s", lang, have, file, want)
	}
	return r
}

// Runtime compares pinned runtime versions with the installed ones.
func Runtime(dir string) []report.Result {
	var results []report.Result
	if file, pin := readPin(dir, ".nvmrc", ".node-version"); file != "" {
		results = append(results, compare("Node", file, pin, [][]string{{"node", "--version"}}))
	}
	if file, pin := readPin(dir, ".python-version"); file != "" {
		results = append(results, compare("Python", file, pin, [][]string{{"python", "--version"}, {"python3", "--version"}}))
	}
	if len(results) == 0 {
		results = append(results, report.Result{
			Status:  report.OK,
			Name:    "Runtime",
			Message: "no pinned version found (.nvmrc, .node-version, .python-version)",
		})
	}
	return results
}
