package scan

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSecretFile(t *testing.T) {
	cases := map[string]bool{
		".env":                   true,
		"config/.env.production": true,
		".env.local":             true,
		".env.example":           false,
		".env.sample":            false,
		"server.key":             true,
		"certs/site.PEM":         true,
		"id_rsa":                 true,
		"credentials.json":       true,
		"src/keyboard.js":        false,
		"README.md":              false,
		"monkey.go":              false,
	}
	for in, want := range cases {
		if got := secretFile(in); got != want {
			t.Errorf("secretFile(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestDebugFlags(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, body string) {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write("settings.py", "import os\nDEBUG = True\n")
	write("app.js", "const cfg = {\n  debug: true,\n};\n")
	write("tests/test_x.py", "DEBUG = True\n")
	write("ok.py", "DEBUG = False\n")

	files := []string{"settings.py", "app.js", "tests/test_x.py", "ok.py"}
	got := debugFlags(dir, files)
	want := []string{"debug flag on: settings.py:2", "debug flag on: app.js:2"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("debugFlags = %v, want %v", got, want)
	}
}
