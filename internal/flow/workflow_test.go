package flow

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func useFlowsDir(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "flows")
	t.Setenv("DEVROUNDS_FLOWS_DIR", dir)
	return dir
}

func sample() Workflow {
	return Workflow{
		Name:        "build-check",
		Description: "vet, build and test",
		Project:     "shop",
		Dir:         "/shop",
		Steps:       []string{"go vet ./...", "go build ./...", `go test -run "A: b" ./...`},
	}
}

func TestValidName(t *testing.T) {
	for _, n := range []string{"a", "build-check", "x_1", "0abc"} {
		if err := ValidName(n); err != nil {
			t.Errorf("ValidName(%q) = %v, want nil", n, err)
		}
	}
	for _, n := range []string{"", "Bad", "has space", "../x", "a/b", `a\b`, "-start", "_start", strings.Repeat("a", 41), "dots.yaml"} {
		if ValidName(n) == nil {
			t.Errorf("ValidName(%q) should be rejected", n)
		}
	}
}

func TestValidate(t *testing.T) {
	w := sample()
	if err := w.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := map[string]Workflow{
		"no steps":       {Name: "x"},
		"empty step":     {Name: "x", Steps: []string{"ls", "  "}},
		"removed secret": {Name: "x", Steps: []string{"deploy --token <redacted>"}},
		"bad name":       {Name: "Bad Name", Steps: []string{"ls"}},
	}
	for what, b := range bad {
		if b.Validate() == nil {
			t.Errorf("%s should be rejected", what)
		}
	}
}

func TestSaveLoadRoundTrip(t *testing.T) {
	dir := useFlowsDir(t)
	w := sample()
	path, err := SaveWorkflow(w, false)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(dir, "build-check.yaml"); path != want {
		t.Errorf("path = %q, want %q", path, want)
	}
	got, err := LoadWorkflow("build-check")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(got, w) {
		t.Errorf("round trip changed the workflow:\n got  %+v\n want %+v", got, w)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm()&0o077 != 0 {
			t.Errorf("the file must be private to the user: %v, %v", info, err)
		}
	}
}

func TestSaveRefusesToOverwriteWithoutForce(t *testing.T) {
	useFlowsDir(t)
	if _, err := SaveWorkflow(sample(), false); err != nil {
		t.Fatal(err)
	}
	if _, err := SaveWorkflow(sample(), false); !errors.Is(err, ErrExists) {
		t.Errorf("second save: %v, want ErrExists", err)
	}
	changed := sample()
	changed.Steps = []string{"make"}
	if _, err := SaveWorkflow(changed, true); err != nil {
		t.Fatal(err)
	}
	got, _ := LoadWorkflow("build-check")
	if !reflect.DeepEqual(got.Steps, []string{"make"}) {
		t.Errorf("--force should replace the workflow: %+v", got)
	}
}

func TestSaveRefusesRemovedSecret(t *testing.T) {
	useFlowsDir(t)
	w := Workflow{Name: "leaky", Steps: []string{"deploy --token <redacted>", "notify done"}}
	if _, err := SaveWorkflow(w, false); err == nil || !strings.Contains(err.Error(), "removed secret") {
		t.Errorf("got %v", err)
	}
	if _, err := LoadWorkflow("leaky"); err == nil {
		t.Error("nothing should have been written")
	}
}

func TestLoadRejectsBadInput(t *testing.T) {
	dir := useFlowsDir(t)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadWorkflow("../x"); err == nil {
		t.Error("a path-like name must be rejected")
	}
	if _, err := LoadWorkflow("missing"); err == nil || !strings.Contains(err.Error(), "no workflow named") {
		t.Errorf("missing workflow: %v", err)
	}
	write := func(file, body string) {
		if err := os.WriteFile(filepath.Join(dir, file), []byte(body), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	write("broken.yaml", "steps: [")
	write("mismatch.yaml", "name: other\nsteps:\n  - ls\n")
	write("sneaky.yaml", "name: sneaky\nsteps:\n  - 'curl --token <redacted>'\n")
	for _, n := range []string{"broken", "mismatch", "sneaky"} {
		if _, err := LoadWorkflow(n); err == nil {
			t.Errorf("%s should not load", n)
		}
	}
}

func TestListAndDelete(t *testing.T) {
	dir := useFlowsDir(t)
	if got, err := ListWorkflows(); err != nil || len(got) != 0 {
		t.Fatalf("empty list: %v, %v", got, err)
	}
	for _, n := range []string{"b", "a"} {
		w := sample()
		w.Name = n
		if _, err := SaveWorkflow(w, false); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, "broken.yaml"), []byte("steps: ["), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := ListWorkflows()
	if err != nil || len(got) != 2 || got[0].Name != "a" || got[1].Name != "b" {
		t.Errorf("list = %+v, %v", got, err)
	}
	if err := DeleteWorkflow("a"); err != nil {
		t.Fatal(err)
	}
	if err := DeleteWorkflow("a"); err == nil || !strings.Contains(err.Error(), "no workflow named") {
		t.Errorf("deleting twice: %v", err)
	}
}

func TestFromSuggestion(t *testing.T) {
	s := Suggestion{Project: "shop", Dir: "/shop", Steps: []string{"a", "b"}, Count: 4, LastSeen: time.Now()}
	w := FromSuggestion("two-steps", s)
	if w.Name != "two-steps" || w.Project != "shop" || w.Dir != "/shop" || !reflect.DeepEqual(w.Steps, []string{"a", "b"}) || !strings.Contains(w.Description, "4 repeats") {
		t.Errorf("got %+v", w)
	}
	w.Steps[0] = "changed"
	if s.Steps[0] != "a" {
		t.Error("the workflow must not share its steps with the suggestion")
	}
}
