package scan

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestLooksLikeCommand(t *testing.T) {
	cases := map[string]bool{
		"go test ./...":                  true,
		"make":                           true,
		"docker compose up -d":           true,
		"echo hello":                     false,
		"npm publish --token abc123":     false,
		"go build ${{ matrix.os }}":      false,
		"curl -H 'Authorization: x' url": false,
		"":                               false,
	}
	for in, want := range cases {
		if got := looksLikeCommand(in); got != want {
			t.Errorf("looksLikeCommand(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestReadmeCommands(t *testing.T) {
	dir := t.TempDir()
	readme := "# Demo\n\n" +
		"Run go test in prose, not as a command.\n\n" +
		"```\n" +
		"$ make build   # build it\n" +
		"echo not a build tool\n" +
		"```\n\n" +
		"    docker compose up -d\n"
	if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte(readme), 0o644); err != nil {
		t.Fatal(err)
	}
	got := readmeCommands(dir)
	want := []string{"make build", "docker compose up -d"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("readmeCommands = %v, want %v", got, want)
	}
}

func TestCICommands(t *testing.T) {
	dir := t.TempDir()
	wf := filepath.Join(dir, ".github", "workflows")
	if err := os.MkdirAll(wf, 0o755); err != nil {
		t.Fatal(err)
	}
	ci := "jobs:\n  t:\n    steps:\n" +
		"      - run: go test ./...\n" +
		"      - run: echo hi\n" +
		"      - run: go build ${{ matrix.os }}\n"
	if err := os.WriteFile(filepath.Join(wf, "ci.yml"), []byte(ci), 0o644); err != nil {
		t.Fatal(err)
	}
	got := ciCommands(dir)
	want := []string{"go test ./..."}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ciCommands = %v, want %v", got, want)
	}
}
