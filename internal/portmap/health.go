package portmap

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

// HealthResult is the answer of one local port to a GET on the health path.
type HealthResult struct {
	Port    int    `json:"port"`
	URL     string `json:"url"`
	Status  int    `json:"status,omitempty"`
	Verdict string `json:"verdict"`
	Millis  int64  `json:"millis,omitempty"`
}

// Ports that speak something other than HTTP. They are skipped, never probed.
var nonHTTPPorts = map[int]bool{
	22: true, 25: true, 110: true, 143: true, 389: true, 445: true, 1433: true, 1521: true,
	3306: true, 5432: true, 5672: true, 6379: true, 9042: true, 9092: true, 11211: true, 27017: true,
}

func verdict(status int) string {
	switch {
	case status >= 200 && status < 300:
		return "healthy"
	case status >= 500:
		return "unhealthy"
	}
	return "answered"
}

// HealthTargets picks the project's own ports worth probing: ports it names that are
// listening and are not flagged as conflicts.
func HealthTargets(rep Report, expected []Expected) []int {
	listening := map[int]bool{}
	for _, r := range rep.Ports {
		listening[r.Port] = true
	}
	skip := map[int]bool{}
	if rep.Project != nil {
		for _, c := range rep.Project.Conflicts {
			skip[c.Port] = true
		}
	}
	seen := map[int]bool{}
	var out []int
	for _, e := range expected {
		if listening[e.Port] && !skip[e.Port] && !seen[e.Port] {
			seen[e.Port] = true
			out = append(out, e.Port)
		}
	}
	return out
}

// CheckHealth sends one GET to 127.0.0.1 for each port and records the status code only.
// It ignores proxies, does not follow redirects and never reads the response body.
func CheckHealth(ports []int, path string, timeout time.Duration) []HealthResult {
	if path == "" {
		path = "/health"
	}
	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}
	client := &http.Client{
		Timeout:       timeout,
		Transport:     &http.Transport{DisableKeepAlives: true, Proxy: nil},
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}
	out := []HealthResult{}
	for _, p := range ports {
		r := HealthResult{Port: p, URL: fmt.Sprintf("http://127.0.0.1:%d%s", p, path)}
		if nonHTTPPorts[p] {
			r.Verdict = "skipped (not an HTTP port)"
			out = append(out, r)
			continue
		}
		start := time.Now()
		resp, err := client.Get(r.URL)
		if err != nil {
			r.Verdict = "no HTTP answer"
			out = append(out, r)
			continue
		}
		resp.Body.Close()
		r.Status = resp.StatusCode
		r.Millis = time.Since(start).Milliseconds()
		r.Verdict = verdict(resp.StatusCode)
		out = append(out, r)
	}
	return out
}
