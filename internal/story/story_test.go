package story

import (
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func TestSinceArg(t *testing.T) {
	cases := map[string]string{
		"7d":         "7 days ago",
		"2w":         "2 weeks ago",
		"1d":         "1 days ago",
		"2026-11-01": "2026-11-01",
	}
	for in, want := range cases {
		if got := SinceArg(in); got != want {
			t.Errorf("SinceArg(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestPeriod(t *testing.T) {
	cases := []struct{ since, until, want string }{
		{"7d", "", "last 7 days"},
		{"1w", "", "last 1 week"},
		{"2w", "", "last 2 weeks"},
		{"2026-11-01", "", "since 2026-11-01"},
		{"2026-11-01", "2026-11-08", "since 2026-11-01 until 2026-11-08"},
	}
	for _, c := range cases {
		if got := Period(c.since, c.until); got != c.want {
			t.Errorf("Period(%q, %q) = %q, want %q", c.since, c.until, got, c.want)
		}
	}
}

func TestAreaOf(t *testing.T) {
	cases := []struct {
		files []string
		want  string
	}{
		{[]string{"cmd/a.go", "cmd/b.go", "internal/c.go"}, "cmd"},
		{[]string{"README.md"}, "(root)"},
		{[]string{"a/x", "b/y"}, "a"},
		{nil, "(no files)"},
	}
	for _, c := range cases {
		if got := areaOf(c.files); got != c.want {
			t.Errorf("areaOf(%v) = %q, want %q", c.files, got, c.want)
		}
	}
}

func TestGroupBy(t *testing.T) {
	commits := []Commit{
		{Hash: "1", Date: "2026-11-02", Subject: "a", Area: "cmd"},
		{Hash: "2", Date: "2026-11-02", Subject: "b", Area: "internal"},
		{Hash: "3", Date: "2026-11-01", Subject: "c", Area: "internal"},
		{Hash: "4", Date: "2026-11-01", Subject: "d", Area: "internal"},
	}
	byArea := GroupBy(commits, "area")
	if len(byArea) != 2 || byArea[0].Name != "internal" || len(byArea[0].Commits) != 3 || byArea[1].Name != "cmd" {
		t.Errorf("by area = %+v", byArea)
	}
	byDay := GroupBy(commits, "day")
	if len(byDay) != 2 || byDay[0].Name != "2026-11-02" || byDay[1].Name != "2026-11-01" {
		t.Errorf("by day = %+v", byDay)
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

func TestCommits(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	dir := t.TempDir()
	gitRun(t, dir, "init")
	write := func(rel string) {
		full := filepath.Join(dir, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(full, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	commit := func(name, email, msg string) {
		gitRun(t, dir, "add", ".")
		gitRun(t, dir, "-c", "user.name="+name, "-c", "user.email="+email,
			"-c", "commit.gpgsign=false", "commit", "-m", msg)
	}
	write("cmd/a.go")
	write("cmd/b.go")
	commit("Ann", "ann@example.com", "add cmd files")
	write("docs/x.md")
	commit("Bob", "bob@example.com", "add docs")

	all, err := Commits(dir, "7d", "", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 2 {
		t.Fatalf("got %d commits, want 2: %+v", len(all), all)
	}
	if all[0].Subject != "add docs" || all[0].Area != "docs" || all[0].Files != 1 {
		t.Errorf("newest commit = %+v", all[0])
	}
	if all[1].Subject != "add cmd files" || all[1].Area != "cmd" || all[1].Files != 2 {
		t.Errorf("oldest commit = %+v", all[1])
	}

	ann, err := Commits(dir, "7d", "", "ann@example.com")
	if err != nil {
		t.Fatal(err)
	}
	var subjects []string
	for _, c := range ann {
		subjects = append(subjects, c.Subject)
	}
	if !reflect.DeepEqual(subjects, []string{"add cmd files"}) {
		t.Errorf("author filter gave %v", subjects)
	}

	none, _ := Commits(dir, "2000-01-01", "2000-01-02", "")
	if len(none) != 0 {
		t.Errorf("an old date range should be empty, got %+v", none)
	}

	if _, err := Commits(t.TempDir(), "7d", "", ""); err == nil {
		t.Error("expected an error outside a git repository")
	}
}
