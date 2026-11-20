package portmap

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestComposeProjectName(t *testing.T) {
	cases := map[string]string{
		filepath.Join("x", "Shop API!"): "shopapi",
		filepath.Join("x", "my_app-2"):  "my_app-2",
	}
	for in, want := range cases {
		if got := composeProjectName(in); got != want {
			t.Errorf("composeProjectName(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestWithin(t *testing.T) {
	dir := t.TempDir()
	if !within(dir, dir) || !within(dir, filepath.Join(dir, "a", "b")) {
		t.Error("the folder itself and its subfolders are within")
	}
	if within(dir, filepath.Dir(dir)) || within(dir, t.TempDir()) || within(dir, dir+"-other") {
		t.Error("the parent, other folders and look-alike names are not within")
	}
}

func TestConflicts(t *testing.T) {
	dir := t.TempDir()
	other := t.TempDir()
	proj := composeProjectName(dir)
	rep := Report{
		Ports: []Row{
			{Entry: Entry{Port: 3000, PID: 11, Process: "node", Folder: filepath.Join(dir, "api")}},
			{Entry: Entry{Port: 4000, PID: 12, Process: "node", Folder: other}},
			{Entry: Entry{Port: 5000, PID: 13}},
			{Entry: Entry{Port: 6000, PID: 14}, Container: "ours"},
			{Entry: Entry{Port: 7000, PID: 15}, Container: "theirs"},
		},
		Containers: []Container{{Name: "ours", Project: proj}, {Name: "theirs", Project: "elsewhere"}},
	}
	expected := []Expected{{3000, "a"}, {4000, "b"}, {5000, "c"}, {6000, "d"}, {7000, "e"}, {8000, "f"}}
	got := Conflicts(rep, dir, expected)
	if len(got) != 3 {
		t.Fatalf("got %d conflicts, want 3: %+v", len(got), got)
	}
	if got[0].Port != 4000 || got[0].Holder != "node (pid 12) from "+other {
		t.Errorf("foreign process: %+v", got[0])
	}
	if got[1].Port != 5000 || got[1].Holder != "pid 13" {
		t.Errorf("unknown process: %+v", got[1])
	}
	if got[2].Port != 7000 || got[2].Holder != "container theirs" {
		t.Errorf("foreign container: %+v", got[2])
	}
}

func TestPrintTextProject(t *testing.T) {
	var buf bytes.Buffer
	rep := Build(nil, nil, true)
	rep.Project = &ProjectInfo{Name: "demo", Expected: 2, Conflicts: []Conflict{{Port: 4000, Source: ".env.example", Holder: "pid 9"}}}
	PrintText(&buf, rep)
	if out := buf.String(); !strings.Contains(out, "Port conflicts for demo:") || !strings.Contains(out, "4000 (.env.example) is held by pid 9") {
		t.Errorf("conflicts missing:\n%s", out)
	}

	buf.Reset()
	rep.Project = &ProjectInfo{Name: "demo", Expected: 2, Conflicts: []Conflict{}}
	PrintText(&buf, rep)
	if !strings.Contains(buf.String(), "No port conflicts for demo") {
		t.Errorf("clean case missing:\n%s", buf.String())
	}

	buf.Reset()
	rep.Project = &ProjectInfo{Name: "demo", Conflicts: []Conflict{}}
	PrintText(&buf, rep)
	if !strings.Contains(buf.String(), "names no specific ports") {
		t.Errorf("no-ports case missing:\n%s", buf.String())
	}
}
