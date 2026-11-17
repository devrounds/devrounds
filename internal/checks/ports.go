package checks

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	gnet "github.com/shirou/gopsutil/v4/net"
	"github.com/shirou/gopsutil/v4/process"
	"gopkg.in/yaml.v3"

	"github.com/devrounds/devrounds/internal/report"
)

type portRef struct {
	Port   int
	Source string
}

type listener struct {
	PID  int32
	Name string
}

var envPortRe = regexp.MustCompile(`^\s*(?:export\s+)?(PORT|APP_PORT|SERVER_PORT|HTTP_PORT|WEB_PORT)\s*=\s*["']?(\d{1,5})["']?\s*(?:#.*)?$`)

// hostPort extracts the host port from one Compose "ports" entry.
// Entries that use variables or ranges are skipped.
func hostPort(v any) int {
	switch t := v.(type) {
	case int:
		return 0 // container port only, host port is random
	case string:
		s := t
		if strings.HasSuffix(s, "/udp") {
			return 0
		}
		s = strings.TrimSuffix(s, "/tcp")
		parts := strings.Split(s, ":")
		var host string
		switch len(parts) {
		case 2:
			host = parts[0]
		case 3:
			host = parts[1]
		default:
			return 0
		}
		n, err := strconv.Atoi(strings.TrimSpace(host))
		if err != nil {
			return 0
		}
		return n
	case map[string]any:
		if proto, ok := t["protocol"].(string); ok && proto == "udp" {
			return 0
		}
		switch pub := t["published"].(type) {
		case int:
			return pub
		case string:
			if n, err := strconv.Atoi(pub); err == nil {
				return n
			}
		}
	}
	return 0
}

func composePorts(path string) []int {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	var cf struct {
		Services map[string]struct {
			Ports []any `yaml:"ports"`
		} `yaml:"services"`
	}
	if err := yaml.Unmarshal(data, &cf); err != nil {
		return nil
	}
	names := make([]string, 0, len(cf.Services))
	for n := range cf.Services {
		names = append(names, n)
	}
	sort.Strings(names)
	var out []int
	for _, n := range names {
		for _, v := range cf.Services[n].Ports {
			if p := hostPort(v); p > 0 {
				out = append(out, p)
			}
		}
	}
	return out
}

// envPorts reads port numbers only. Other values are never looked at.
func envPorts(path string) []int {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []int
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		m := envPortRe.FindStringSubmatch(sc.Text())
		if m == nil {
			continue
		}
		if n, err := strconv.Atoi(m[2]); err == nil {
			out = append(out, n)
		}
	}
	return out
}

func detectPorts(dir string) []portRef {
	var refs []portRef
	seen := map[int]bool{}
	add := func(p int, src string) {
		if p < 1 || p > 65535 || seen[p] {
			return
		}
		seen[p] = true
		refs = append(refs, portRef{p, src})
	}
	for _, f := range []string{"docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml"} {
		for _, p := range composePorts(filepath.Join(dir, f)) {
			add(p, f)
		}
	}
	for _, p := range envPorts(filepath.Join(dir, ".env.example")) {
		add(p, ".env.example")
	}
	if len(refs) == 0 {
		switch {
		case exists(dir, "package.json"):
			add(3000, "Node default")
		case exists(dir, "pyproject.toml", "requirements.txt", "Pipfile"):
			add(8000, "Python default")
		}
	}
	return refs
}

func listeners() (map[int]listener, error) {
	conns, err := gnet.Connections("tcp")
	if err != nil {
		return nil, err
	}
	m := map[int]listener{}
	for _, c := range conns {
		if c.Status != "LISTEN" {
			continue
		}
		p := int(c.Laddr.Port)
		if _, ok := m[p]; ok {
			continue
		}
		l := listener{PID: c.Pid}
		if c.Pid > 0 {
			if proc, err := process.NewProcess(c.Pid); err == nil {
				if name, err := proc.Name(); err == nil {
					l.Name = name
				}
			}
		}
		m[p] = l
	}
	return m, nil
}

// Ports reports which of the project's expected ports are already taken.
func Ports(dir string) []report.Result {
	refs := detectPorts(dir)
	if len(refs) == 0 {
		return []report.Result{{Status: report.OK, Name: "Ports", Message: "no ports to check"}}
	}
	live, err := listeners()
	if err != nil {
		return []report.Result{{
			Status:  report.Warn,
			Name:    "Ports",
			Message: "could not read listening ports: " + err.Error(),
		}}
	}
	var results []report.Result
	var free []string
	for _, r := range refs {
		l, busy := live[r.Port]
		if !busy {
			free = append(free, strconv.Itoa(r.Port))
			continue
		}
		who := "unknown process"
		switch {
		case l.Name != "" && l.PID > 0:
			who = fmt.Sprintf("%s, pid %d", l.Name, l.PID)
		case l.PID > 0:
			who = fmt.Sprintf("pid %d", l.PID)
		}
		results = append(results, report.Result{
			Status:  report.Warn,
			Name:    "Ports",
			Message: fmt.Sprintf("%d in use (%s), expected by %s", r.Port, who, r.Source),
		})
	}
	if len(free) > 0 {
		results = append(results, report.Result{
			Status:  report.OK,
			Name:    "Ports",
			Message: "free: " + strings.Join(free, ", "),
		})
	}
	return results
}
