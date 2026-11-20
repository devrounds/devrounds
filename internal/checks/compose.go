package checks

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"gopkg.in/yaml.v3"

	"github.com/devrounds/devrounds/internal/report"
)

// dockerBin is the docker command. Tests replace it with a fake.
var dockerBin = "docker"

func composeFile(dir string) string {
	for _, f := range []string{"compose.yaml", "compose.yml", "docker-compose.yaml", "docker-compose.yml"} {
		if exists(dir, f) {
			return f
		}
	}
	return ""
}

// composeServices lists service names, leaving out services behind a profile.
func composeServices(path string) []string {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cf struct {
		Services map[string]struct {
			Profiles []string `yaml:"profiles"`
		} `yaml:"services"`
	}
	if yaml.Unmarshal(data, &cf) != nil {
		return nil
	}
	var names []string
	for name, svc := range cf.Services {
		if len(svc.Profiles) == 0 {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

func runDocker(dir string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, dockerBin, args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return string(out), err
}

func splitLines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}

func runningServices(dir, file string) ([]string, error) {
	out, err := runDocker(dir, "compose", "-f", file, "ps", "--services", "--status", "running")
	if err != nil {
		out, err = runDocker(dir, "compose", "-f", file, "ps", "--services", "--filter", "status=running")
		if err != nil {
			return nil, err
		}
	}
	return splitLines(out), nil
}

func composeResult(defined, running []string) report.Result {
	runSet := map[string]bool{}
	for _, r := range running {
		runSet[r] = true
	}
	var down []string
	up := 0
	for _, d := range defined {
		if runSet[d] {
			up++
		} else {
			down = append(down, d)
		}
	}
	r := report.Result{Name: "Docker"}
	switch {
	case len(down) == 0:
		r.Status = report.OK
		r.Message = fmt.Sprintf("all %d compose services running: %s", len(defined), nameList(defined))
	case up == 0:
		r.Status = report.Fail
		r.Message = fmt.Sprintf("no compose services running (%d defined): %s", len(defined), nameList(defined))
		r.Next = "docker compose up -d"
	default:
		r.Status = report.Warn
		r.Message = fmt.Sprintf("%d of %d compose services not running: %s", len(down), len(defined), nameList(down))
		r.Next = "docker compose up -d"
	}
	return r
}

// Compose checks whether the project's Docker Compose services are running.
// It does nothing when Docker is not installed, because Binaries already reports that.
func Compose(dir string) []report.Result {
	file := composeFile(dir)
	if file == "" {
		return nil
	}
	if _, err := exec.LookPath(dockerBin); err != nil {
		return nil
	}
	defined := composeServices(filepath.Join(dir, file))
	if len(defined) == 0 {
		return nil
	}
	if _, err := runDocker(dir, "info", "--format", "{{.ServerVersion}}"); err != nil {
		return []report.Result{{
			Status:  report.Fail,
			Name:    "Docker",
			Message: "Docker is installed but its daemon is not reachable, so compose services cannot run",
			Next:    "start Docker",
		}}
	}
	running, err := runningServices(dir, file)
	if err != nil {
		return []report.Result{{
			Status:  report.Warn,
			Name:    "Docker",
			Message: "could not ask docker compose which services are running",
		}}
	}
	return []report.Result{composeResult(defined, running)}
}
