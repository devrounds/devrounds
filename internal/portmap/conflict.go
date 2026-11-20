package portmap

import (
	"fmt"
	"path/filepath"
	"regexp"
	"strings"
)

// Expected is a port a project says it will use.
type Expected struct {
	Port   int
	Source string
}

// Conflict is an expected port that something other than the project is holding.
type Conflict struct {
	Port   int    `json:"port"`
	Source string `json:"expected_by"`
	Holder string `json:"held_by"`
}

// ProjectInfo is the project-aware part of the map report.
type ProjectInfo struct {
	Name      string     `json:"name"`
	Expected  int        `json:"expected_ports"`
	Conflicts []Conflict `json:"conflicts"`
}

var badNameRe = regexp.MustCompile(`[^a-z0-9_-]`)

// composeProjectName is the name Docker Compose gives a project that lives in dir.
func composeProjectName(dir string) string {
	return badNameRe.ReplaceAllString(strings.ToLower(filepath.Base(filepath.Clean(dir))), "")
}

// within reports whether folder is dir or somewhere below it.
func within(dir, folder string) bool {
	rel, err := filepath.Rel(filepath.Clean(dir), filepath.Clean(folder))
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
}

func describe(r Row) string {
	if r.Container != "" {
		return "container " + r.Container
	}
	var s string
	switch {
	case r.Process != "" && r.PID > 0:
		s = fmt.Sprintf("%s (pid %d)", r.Process, r.PID)
	case r.PID > 0:
		s = fmt.Sprintf("pid %d", r.PID)
	default:
		s = "an unknown process"
	}
	if r.Folder != "" {
		s += " from " + r.Folder
	}
	return s
}

// Conflicts finds expected ports that are held by something other than the project.
func Conflicts(rep Report, projectDir string, expected []Expected) []Conflict {
	projName := composeProjectName(projectDir)
	containers := map[string]Container{}
	for _, c := range rep.Containers {
		containers[c.Name] = c
	}
	isOurs := func(r Row) bool {
		if r.Container != "" {
			c, ok := containers[r.Container]
			return ok && c.Project != "" && c.Project == projName
		}
		return r.Folder != "" && within(projectDir, r.Folder)
	}
	out := []Conflict{}
	for _, e := range expected {
		holder := ""
		owned := false
		for _, row := range rep.Ports {
			if row.Port != e.Port {
				continue
			}
			if isOurs(row) {
				owned = true
				break
			}
			if holder == "" {
				holder = describe(row)
			}
		}
		if !owned && holder != "" {
			out = append(out, Conflict{Port: e.Port, Source: e.Source, Holder: holder})
		}
	}
	return out
}
