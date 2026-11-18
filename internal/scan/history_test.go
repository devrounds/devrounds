package scan

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestMergeContributors(t *testing.T) {
	in := []Count{{"Ann", 5}, {"Bob", 3}, {"ann", 2}, {"Cy", 1}, {"Di", 1}}
	got := mergeContributors(in)
	want := []Count{{"Ann", 7}, {"Bob", 3}, {"Cy", 1}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("mergeContributors = %v, want %v", got, want)
	}
}

func gitRun(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func TestGitHistory(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	gitRun(t, dir, "init")
	commit := func(author, file string) {
		if err := os.WriteFile(filepath.Join(dir, file), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		gitRun(t, dir, "add", ".")
		gitRun(t, dir, "-c", "user.name="+author, "-c", "user.email=a@example.com",
			"-c", "commit.gpgsign=false", "commit", "-m", "add "+file)
	}
	commit("Ann", "a.txt")
	commit("ann", "b.txt")

	h := Git(dir)
	if h == nil {
		t.Fatal("Git returned nil for a repo with commits")
	}
	if h.Commits != 2 {
		t.Errorf("Commits = %d, want 2", h.Commits)
	}
	if len(h.Contributors) != 1 || h.Contributors[0].Count != 2 {
		t.Errorf("Contributors = %v, want one merged entry with 2 commits", h.Contributors)
	}
	if h.LastActivity == "" {
		t.Error("LastActivity is empty")
	}
	if len(h.ActiveAreas) != 1 || h.ActiveAreas[0].Name != "(root files)" || h.ActiveAreas[0].Count != 2 {
		t.Errorf("ActiveAreas = %v, want (root files) with 2", h.ActiveAreas)
	}
	if Git(t.TempDir()) != nil {
		t.Error("expected nil outside a git repo")
	}
}
