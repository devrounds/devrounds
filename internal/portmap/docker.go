package portmap

import (
	"context"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

// dockerBin is the docker command. Tests replace it with a stand-in.
var dockerBin = "docker"

// Container is one running Docker container.
type Container struct {
	Name      string `json:"name"`
	Image     string `json:"image"`
	Project   string `json:"compose_project,omitempty"`
	Service   string `json:"compose_service,omitempty"`
	Ports     string `json:"ports,omitempty"`
	HostPorts []int  `json:"host_ports,omitempty"`
	Status    string `json:"status"`
}

var hostPortRe = regexp.MustCompile(`:(\d+)->`)

// parseHostPorts pulls the published host ports out of docker's Ports column.
func parseHostPorts(s string) []int {
	seen := map[int]bool{}
	var out []int
	for _, m := range hostPortRe.FindAllStringSubmatch(s, -1) {
		n, err := strconv.Atoi(m[1])
		if err != nil || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	sort.Ints(out)
	return out
}

func parseContainers(out string) []Container {
	var cs []Container
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimRight(line, "\r")
		if strings.TrimSpace(line) == "" {
			continue
		}
		p := strings.SplitN(line, "|", 6)
		if len(p) != 6 {
			continue
		}
		cs = append(cs, Container{
			Name: p[0], Image: p[1], Project: p[2], Service: p[3],
			Ports: p[4], HostPorts: parseHostPorts(p[4]), Status: p[5],
		})
	}
	return cs
}

// Containers lists running containers. ok is false when docker is missing or its daemon is not reachable.
func Containers() ([]Container, bool) {
	if _, err := exec.LookPath(dockerBin); err != nil {
		return nil, false
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	format := `{{.Names}}|{{.Image}}|{{.Label "com.docker.compose.project"}}|{{.Label "com.docker.compose.service"}}|{{.Ports}}|{{.Status}}`
	out, err := exec.CommandContext(ctx, dockerBin, "ps", "--format", format).Output()
	if err != nil {
		return nil, false
	}
	return parseContainers(string(out)), true
}
