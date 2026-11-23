package portmap

import (
	"bytes"
	"strings"
	"testing"
)

func TestRedactHome(t *testing.T) {
	home := `C:\Users\Ann`
	cases := map[string]string{
		`C:\Users\Ann\`:               `~\`,
		`C:\Users\Ann\AppData\x`:      `~\AppData\x`,
		`node from C:\Users\Ann\proj`: `node from ~\proj`,
		`C:\Users\Ann`:                `~`,
		`C:\Users\Annabel\x`:          `C:\Users\Annabel\x`,
		`D:\work`:                     `D:\work`,
	}
	for in, want := range cases {
		if got := redactHome(in, home); got != want {
			t.Errorf("redactHome(%q) = %q, want %q", in, got, want)
		}
	}
	if got := redactHome("anything", ""); got != "anything" {
		t.Errorf("empty home must change nothing, got %q", got)
	}
}

func TestPrintMarkdown(t *testing.T) {
	home := `C:\Users\Ann`
	rep := Build(
		[]Entry{{Port: 11434, PID: 7, Process: "ollama.exe", Folder: `C:\Users\Ann\AppData\Ollama\`}},
		[]Container{{Name: "pg", Image: "postgres|16", Project: "shop", Service: "db", Ports: "0.0.0.0:5432->5432/tcp", Status: "Up"}},
		true,
	)
	rep.Project = &ProjectInfo{Name: "demo", Expected: 1, Conflicts: []Conflict{{Port: 4000, Source: ".env.example", Holder: `node (pid 9) from C:\Users\Ann\proj`}}}
	rep.Health = []HealthResult{{Port: 8080, URL: "http://127.0.0.1:8080/health", Status: 200, Verdict: "healthy", Millis: 3}}

	var buf bytes.Buffer
	PrintMarkdown(&buf, rep, home)
	out := buf.String()
	for _, want := range []string{
		"# Port map\n",
		"## Listening ports\n",
		"| Port | Process | PID | Folder | Container |\n|---|---|---|---|---|\n",
		`| 11434 | ollama.exe | 7 | ~\AppData\Ollama\ | - |`,
		"## Containers\n",
		`| pg | postgres\|16 | shop/db |`,
		"## Project: demo\n",
		`- **Port 4000** (.env.example) is held by node (pid 9) from ~\proj`,
		"| http://127.0.0.1:8080/health | healthy (200, 3 ms) |",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output is missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "Ann") {
		t.Errorf("the user name must not appear in a shareable report:\n%s", out)
	}
	if strings.Contains(out, "Docker was not found") {
		t.Error("the Docker note should not appear when Docker answered")
	}

	buf.Reset()
	PrintMarkdown(&buf, Build(nil, nil, false), home)
	if !strings.Contains(buf.String(), "> Docker was not found") {
		t.Errorf("missing the Docker note:\n%s", buf.String())
	}
}
