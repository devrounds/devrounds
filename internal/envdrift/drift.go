package envdrift

import (
	"fmt"
	"sort"

	"github.com/devrounds/devrounds/internal/report"
)

const detailLimit = 10

func describe(name string, locs []Loc) string {
	s := fmt.Sprintf("%s  (%s", name, locs[0])
	if len(locs) > 1 {
		s += fmt.Sprintf(" +%d more", len(locs)-1)
	}
	return s + ")"
}

func details(m map[string][]Loc, names []string) []string {
	var out []string
	for i, n := range names {
		if i == detailLimit {
			out = append(out, fmt.Sprintf("... and %d more", len(names)-detailLimit))
			break
		}
		out = append(out, describe(n, m[n]))
	}
	return out
}

func missingFrom(src map[string][]Loc, others ...map[string][]Loc) []string {
	var out []string
	for name := range src {
		found := false
		for _, o := range others {
			if _, ok := o[name]; ok {
				found = true
				break
			}
		}
		if !found {
			out = append(out, name)
		}
	}
	sort.Strings(out)
	return out
}

// Run compares env var names between code, .env.example, Docker files and CI.
func Run(dir string) []report.Result {
	code, files := ScanCode(dir)
	example := ReadExample(dir)
	docker := ReadDocker(dir)
	ci := ReadCI(dir)

	var results []report.Result
	problems := 0
	warn := func(msg string, m map[string][]Loc, names []string) {
		problems++
		results = append(results, report.Result{
			Status:  report.Warn,
			Name:    "Env",
			Message: msg,
			Details: details(m, names),
		})
	}

	if files == 0 {
		results = append(results, report.Result{
			Status:  report.OK,
			Name:    "Env",
			Message: "no Node or Python source files found, code usage not checked",
		})
	} else {
		if names := missingFrom(code, example, docker); len(names) > 0 {
			warn(fmt.Sprintf("%d used in code but undocumented", len(names)), code, names)
		}
		if names := missingFrom(example, code); len(names) > 0 {
			warn(fmt.Sprintf("%d documented in .env.example but not used in code", len(names)), example, names)
		}
	}
	if names := missingFrom(ci, example, docker); len(names) > 0 {
		warn(fmt.Sprintf("%d referenced in CI but missing locally", len(names)), ci, names)
	}

	if problems == 0 && files > 0 {
		results = append(results, report.Result{
			Status:  report.OK,
			Name:    "Env",
			Message: fmt.Sprintf("no drift (%d names in code, %d in .env.example)", len(code), len(example)),
		})
	}
	return results
}
