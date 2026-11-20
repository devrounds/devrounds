package checks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/devrounds/devrounds/internal/report"
)

func put(t *testing.T, dir, rel, body string) {
	t.Helper()
	full := filepath.Join(dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNodeManager(t *testing.T) {
	cases := map[string]string{"yarn.lock": "yarn", "pnpm-lock.yaml": "pnpm", "package-lock.json": "npm"}
	for file, want := range cases {
		dir := t.TempDir()
		put(t, dir, file, "x")
		if got, _ := nodeManager(dir); got != want {
			t.Errorf("%s: manager = %q, want %q", file, got, want)
		}
	}
	if got, _ := nodeManager(t.TempDir()); got != "npm" {
		t.Errorf("default manager = %q, want npm", got)
	}
}

func TestNodeInstall(t *testing.T) {
	dir := t.TempDir()
	if got := NodeInstall(dir); len(got) != 0 {
		t.Errorf("no package.json should give no result, got %v", got)
	}
	put(t, dir, "package.json", `{"name":"x"}`)
	if got := NodeInstall(dir); len(got) != 0 {
		t.Errorf("no dependencies should give no result, got %v", got)
	}
	put(t, dir, "package.json", `{"dependencies":{"express":"^4.0.0"}}`)
	got := NodeInstall(dir)
	if len(got) != 1 || got[0].Status != report.Fail || got[0].Next != "npm install" {
		t.Errorf("missing node_modules: got %+v", got)
	}
	put(t, dir, "node_modules/express/index.js", "x")
	got = NodeInstall(dir)
	if len(got) != 1 || got[0].Status != report.OK {
		t.Errorf("with node_modules: got %+v", got)
	}
}

func TestPythonEnv(t *testing.T) {
	dir := t.TempDir()
	if got := PythonEnv(dir, ""); len(got) != 0 {
		t.Errorf("not a Python project should give no result, got %v", got)
	}
	put(t, dir, "requirements.txt", "flask\n")
	got := PythonEnv(dir, "")
	if len(got) != 1 || got[0].Status != report.Warn || got[0].Next != "python -m venv .venv" {
		t.Errorf("no venv: got %+v", got)
	}
	got = PythonEnv(dir, filepath.Join("some", "place", "myenv"))
	if len(got) != 1 || got[0].Status != report.OK || !strings.Contains(got[0].Message, "myenv") {
		t.Errorf("active venv: got %+v", got)
	}
	put(t, dir, ".venv/pyvenv.cfg", "home = x")
	got = PythonEnv(dir, "")
	if len(got) != 1 || got[0].Status != report.OK || !strings.Contains(got[0].Message, ".venv") {
		t.Errorf("project venv: got %+v", got)
	}
}
