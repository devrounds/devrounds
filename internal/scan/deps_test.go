package scan

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
	"time"
)

func writeFile(t *testing.T, dir, name, body string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestNodeDeps(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "package.json", `{
  "dependencies": {
    "express": "^4.18.0",
    "left-pad": "*",
    "mylib": "github:user/repo",
    "alias": "npm:@scope/pkg@1.0.0",
    "range": "^1.0.0 || ^2.0.0",
    "tag": "latest"
  }
}`)
	want := []string{
		"Dependencies that accept any version (2): left-pad, tag",
		"Dependencies installed from a URL or git (1): mylib",
		"No lockfile (package-lock.json, yarn.lock or pnpm-lock.yaml), so installs are not reproducible",
	}
	if got := nodeDeps(dir); !reflect.DeepEqual(got, want) {
		t.Errorf("nodeDeps = %v, want %v", got, want)
	}

	writeFile(t, dir, "package-lock.json", "{}")
	if got := nodeDeps(dir); len(got) != 2 {
		t.Errorf("with a lockfile, got %v, want 2 findings", got)
	}
}

func TestPythonDeps(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, "requirements.txt", "# comment\n"+
		"flask\n"+
		"requests>=2.0\n"+
		"numpy==1.26.0\n"+
		"pandas~=2.0\n"+
		"django>=3,<5\n"+
		"mylib @ https://example.com/mylib.zip\n"+
		"git+https://github.com/x/y.git#egg=y\n"+
		"-r other.txt\n"+
		"pytest ; python_version < \"3.12\"\n")
	want := []string{
		"Requirements without a version (2): flask, pytest",
		"Requirements with only a minimum version (1): requests",
		"Requirements installed from a URL or git (2): mylib, git URL",
	}
	if got := pythonDeps(dir); !reflect.DeepEqual(got, want) {
		t.Errorf("pythonDeps = %v, want %v", got, want)
	}
}

func TestEOLMessage(t *testing.T) {
	if got, want := eolMessage("Node", "14", ".nvmrc", nodeEOL, false),
		"Node 14 (pinned in .nvmrc) reached end of life on 2023-04-30"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got, want := eolMessage("Node", "10", ".nvmrc", nodeEOL, true),
		"Node 10 (pinned in .nvmrc) is long past end of life"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if got := eolMessage("Node", "99", ".nvmrc", nodeEOL, false); got != "" {
		t.Errorf("unknown version should give no message, got %q", got)
	}
	far := map[string]string{"1": "2999-01-01"}
	if got := eolMessage("X", "1", "f", far, false); got != "" {
		t.Errorf("far-future end of life should give no message, got %q", got)
	}
	soonDate := time.Now().AddDate(0, 0, 30).Format("2006-01-02")
	soon := map[string]string{"1": soonDate}
	if got, want := eolMessage("X", "1", "f", soon, false),
		"X 1 (pinned in f) reaches end of life on "+soonDate; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestRuntimeEOL(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, dir, ".nvmrc", "14\n")
	writeFile(t, dir, ".python-version", "2.7.18\n")
	want := []string{
		"Node 14 (pinned in .nvmrc) reached end of life on 2023-04-30",
		"Python 2.7 (pinned in .python-version) is long past end of life",
	}
	if got := runtimeEOL(dir); !reflect.DeepEqual(got, want) {
		t.Errorf("runtimeEOL = %v, want %v", got, want)
	}

	other := t.TempDir()
	writeFile(t, other, ".nvmrc", "lts/iron\n")
	if got := runtimeEOL(other); len(got) != 0 {
		t.Errorf("a named release channel has no number to check, got %v", got)
	}
}
