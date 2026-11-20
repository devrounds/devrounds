package checks

import "strings"

// ExpectedPort is a port a project says it will use, and where that is written down.
type ExpectedPort struct {
	Port   int
	Source string
}

// ExpectedPorts lists ports named in Compose files and .env.example.
// Guessed defaults (3000 for Node, 8000 for Python) are left out.
func ExpectedPorts(dir string) []ExpectedPort {
	var out []ExpectedPort
	for _, r := range detectPorts(dir) {
		if strings.HasSuffix(r.Source, "default") {
			continue
		}
		out = append(out, ExpectedPort{r.Port, r.Source})
	}
	return out
}
