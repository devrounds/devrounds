package checks

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"

	"github.com/devrounds/devrounds/internal/envdrift"
	"github.com/devrounds/devrounds/internal/report"
)

var dotenvLineRe = regexp.MustCompile(`^\s*(?:export\s+)?([A-Za-z_][A-Za-z0-9_]*)\s*=(.*)$`)

func isBlank(raw string) bool {
	v := strings.TrimSpace(raw)
	if strings.HasPrefix(v, "#") {
		return true
	}
	return v == "" || v == `""` || v == "''"
}

// readDotenv returns the names in a .env file and whether each one is empty.
// Values are inspected only to see if they are blank. They are never stored.
func readDotenv(path string) (map[string]bool, bool) {
	f, err := os.Open(path)
	if err != nil {
		return nil, false
	}
	defer f.Close()
	names := map[string]bool{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		if m := dotenvLineRe.FindStringSubmatch(sc.Text()); m != nil {
			names[m[1]] = isBlank(m[2])
		}
	}
	return names, true
}

func exampleFile(dir string) string {
	for _, f := range []string{".env.example", ".env.sample", ".env.template"} {
		if exists(dir, f) {
			return f
		}
	}
	return ".env.example"
}

func nameList(names []string) string {
	const limit = 6
	if len(names) <= limit {
		return strings.Join(names, ", ")
	}
	return fmt.Sprintf("%s +%d more", strings.Join(names[:limit], ", "), len(names)-limit)
}

// EnvFile compares the example file with the real .env. Names only, never values.
func EnvFile(dir string) []report.Result {
	example := envdrift.ReadExample(dir)
	if len(example) == 0 {
		return nil
	}
	names := make([]string, 0, len(example))
	for n := range example {
		names = append(names, n)
	}
	sort.Strings(names)

	dotenv, haveFile := readDotenv(filepath.Join(dir, ".env"))
	var missing, empty []string
	for _, n := range names {
		if blank, ok := dotenv[n]; ok {
			if blank {
				empty = append(empty, n)
			}
			continue
		}
		if v, ok := os.LookupEnv(n); ok {
			if v == "" {
				empty = append(empty, n)
			}
			continue
		}
		missing = append(missing, n)
	}

	ex := exampleFile(dir)
	var results []report.Result
	if len(missing) > 0 {
		msg := fmt.Sprintf("%d missing: %s", len(missing), nameList(missing))
		var next string
		if !haveFile {
			msg = ".env not found, " + msg
			if runtime.GOOS == "windows" {
				next = "copy " + ex + " .env"
			} else {
				next = "cp " + ex + " .env"
			}
		} else {
			next = "add the missing names to .env"
		}
		results = append(results, report.Result{Status: report.Fail, Name: ".env", Message: msg, Next: next})
	}
	if len(empty) > 0 {
		results = append(results, report.Result{
			Status:  report.Warn,
			Name:    ".env",
			Message: fmt.Sprintf("%d empty: %s", len(empty), nameList(empty)),
		})
	}
	if len(results) == 0 {
		msg := fmt.Sprintf("all %d names from %s are set", len(names), ex)
		if !haveFile {
			msg += " (no .env file, values come from the shell)"
		}
		results = append(results, report.Result{Status: report.OK, Name: ".env", Message: msg})
	}
	return results
}
