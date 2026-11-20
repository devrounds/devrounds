package checks

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/devrounds/devrounds/internal/report"
)

func TestComposeServices(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, "compose.yml", "services:\n  web:\n    image: x\n  db:\n    image: y\n  tool:\n    image: z\n    profiles: [debug]\n")
	got := composeServices(filepath.Join(dir, "compose.yml"))
	want := []string{"db", "web"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("composeServices = %v, want %v", got, want)
	}
}

func TestComposeFilePrecedence(t *testing.T) {
	dir := t.TempDir()
	if got := composeFile(dir); got != "" {
		t.Errorf("empty dir: got %q", got)
	}
	put(t, dir, "docker-compose.yml", "services: {}")
	if got := composeFile(dir); got != "docker-compose.yml" {
		t.Errorf("got %q", got)
	}
	put(t, dir, "compose.yaml", "services: {}")
	if got := composeFile(dir); got != "compose.yaml" {
		t.Errorf("got %q, want compose.yaml to win", got)
	}
}

func TestComposeResult(t *testing.T) {
	defined := []string{"db", "web"}
	if r := composeResult(defined, []string{"db", "web"}); r.Status != report.OK {
		t.Errorf("all running: %+v", r)
	}
	r := composeResult(defined, nil)
	if r.Status != report.Fail || r.Next != "docker compose up -d" {
		t.Errorf("none running: %+v", r)
	}
	r = composeResult(defined, []string{"db"})
	if r.Status != report.Warn || r.Message != "1 of 2 compose services not running: web" {
		t.Errorf("one down: %+v", r)
	}
}

func useFakeDocker(t *testing.T, daemon, running string) {
	t.Helper()
	old := dockerBin
	dockerBin = os.Args[0]
	t.Cleanup(func() { dockerBin = old })
	t.Setenv("DEVROUNDS_FAKE_DOCKER", "1")
	t.Setenv("DEVROUNDS_FAKE_DAEMON", daemon)
	t.Setenv("DEVROUNDS_FAKE_RUNNING", running)
}

func TestComposeWithFakeDocker(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, "compose.yml", "services:\n  db:\n    image: x\n  web:\n    image: y\n")

	useFakeDocker(t, "up", "db")
	got := Compose(dir)
	if len(got) != 1 || got[0].Status != report.Warn || !strings.Contains(got[0].Message, "web") {
		t.Errorf("one service down: %+v", got)
	}

	useFakeDocker(t, "up", "db,web")
	got = Compose(dir)
	if len(got) != 1 || got[0].Status != report.OK {
		t.Errorf("all up: %+v", got)
	}

	useFakeDocker(t, "down", "")
	got = Compose(dir)
	if len(got) != 1 || got[0].Status != report.Fail || !strings.Contains(got[0].Message, "daemon") {
		t.Errorf("daemon down: %+v", got)
	}
}
