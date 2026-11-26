package flow

import (
	"reflect"
	"testing"
	"time"
)

var baseTime = time.Date(2026, 11, 1, 9, 0, 0, 0, time.UTC)

type timeline struct {
	now  time.Time
	recs []Record
}

func newTimeline() *timeline { return &timeline{now: baseTime} }

func (tl *timeline) cmd(project, command string, exit int) {
	tl.now = tl.now.Add(time.Minute)
	tl.recs = append(tl.recs, Record{Time: tl.now, Command: command, Project: project, Dir: "/" + project, ExitCode: exit})
}

// run adds one pass through a workflow, then a long pause so the next pass is a new session.
func (tl *timeline) run(project string, cmds ...string) {
	for _, c := range cmds {
		tl.cmd(project, c, 0)
	}
	tl.now = tl.now.Add(time.Hour)
}

func TestSuggestFindsRepeatedWorkflow(t *testing.T) {
	tl := newTimeline()
	for i := 0; i < 3; i++ {
		tl.run("shop", "go vet ./...", "go build ./...", "go test ./...")
	}
	got := Suggest(tl.recs, DefaultOptions())
	want := []string{"go vet ./...", "go build ./...", "go test ./..."}
	if len(got) != 1 || got[0].Count != 3 || !reflect.DeepEqual(got[0].Steps, want) || got[0].Project != "shop" || got[0].Dir != "/shop" {
		t.Fatalf("got %+v", got)
	}
}

func TestSuggestNeedsEnoughRepeats(t *testing.T) {
	tl := newTimeline()
	for i := 0; i < 2; i++ {
		tl.run("shop", "git add .", "git commit -m x")
	}
	if got := Suggest(tl.recs, DefaultOptions()); len(got) != 0 {
		t.Errorf("two repeats are not enough by default: %+v", got)
	}
	o := DefaultOptions()
	o.MinCount = 2
	if got := Suggest(tl.recs, o); len(got) != 1 {
		t.Errorf("with MinCount 2 there should be one suggestion: %+v", got)
	}
}

func TestSuggestIgnoresTrivialCommands(t *testing.T) {
	tl := newTimeline()
	for i := 0; i < 3; i++ {
		tl.run("shop", "git add .", "cd ..", "git commit -m x")
	}
	got := Suggest(tl.recs, DefaultOptions())
	if len(got) != 1 || !reflect.DeepEqual(got[0].Steps, []string{"git add .", "git commit -m x"}) {
		t.Errorf("got %+v", got)
	}
}

func TestSuggestFailedCommandBreaksTheRun(t *testing.T) {
	tl := newTimeline()
	for i := 0; i < 3; i++ {
		tl.cmd("shop", "make build", 0)
		tl.cmd("shop", "make test", 1)
		tl.cmd("shop", "make deploy", 0)
		tl.now = tl.now.Add(time.Hour)
	}
	if got := Suggest(tl.recs, DefaultOptions()); len(got) != 0 {
		t.Errorf("a failed step must break the sequence: %+v", got)
	}
}

func TestSuggestLongPauseBreaksTheRun(t *testing.T) {
	tl := newTimeline()
	for i := 0; i < 3; i++ {
		tl.cmd("shop", "git pull", 0)
		tl.now = tl.now.Add(30 * time.Minute)
		tl.cmd("shop", "npm install", 0)
		tl.now = tl.now.Add(time.Hour)
	}
	if got := Suggest(tl.recs, DefaultOptions()); len(got) != 0 {
		t.Errorf("commands 30 minutes apart are not one workflow: %+v", got)
	}
}

func TestSuggestProjectsStaySeparate(t *testing.T) {
	tl := newTimeline()
	tl.run("a", "npm install", "npm test")
	tl.run("b", "npm install", "npm test")
	tl.run("a", "npm install", "npm test")
	tl.run("b", "npm install", "npm test")
	if got := Suggest(tl.recs, DefaultOptions()); len(got) != 0 {
		t.Errorf("two repeats per project must not add up across projects: %+v", got)
	}
	tl.run("a", "npm install", "npm test")
	got := Suggest(tl.recs, DefaultOptions())
	if len(got) != 1 || got[0].Project != "a" || got[0].Count != 3 {
		t.Errorf("got %+v", got)
	}
	o := DefaultOptions()
	o.Project = "b"
	if got := Suggest(tl.recs, o); len(got) != 0 {
		t.Errorf("project filter b: %+v", got)
	}
}

func TestSuggestIgnoresOneCommandRepeated(t *testing.T) {
	tl := newTimeline()
	for i := 0; i < 4; i++ {
		tl.run("shop", "go test", "go test")
	}
	if got := Suggest(tl.recs, DefaultOptions()); len(got) != 0 {
		t.Errorf("got %+v", got)
	}
}

func TestSuggestKeepsMoreFrequentShorterSequence(t *testing.T) {
	tl := newTimeline()
	for i := 0; i < 3; i++ {
		tl.run("shop", "A1", "B1", "C1")
	}
	for i := 0; i < 2; i++ {
		tl.run("shop", "A1", "B1")
	}
	got := Suggest(tl.recs, DefaultOptions())
	if len(got) != 2 {
		t.Fatalf("got %d suggestions, want 2: %+v", len(got), got)
	}
	byLen := map[int]int{}
	for _, g := range got {
		byLen[len(g.Steps)] = g.Count
	}
	if byLen[3] != 3 || byLen[2] != 5 {
		t.Errorf("counts by length = %v, want 3 steps x3 and 2 steps x5", byLen)
	}
}

func TestStoreAllIsOldestFirst(t *testing.T) {
	s := openTemp(t)
	now := time.Now()
	mustAdd(t, s, Record{Time: now.Add(-2 * time.Hour), Command: "first"})
	mustAdd(t, s, Record{Time: now.Add(-time.Hour), Command: "second"})
	mustAdd(t, s, Record{Time: now, Command: "third"})
	got, err := s.All()
	if err != nil || len(got) != 3 || got[0].Command != "first" || got[2].Command != "third" {
		t.Errorf("All = %+v, %v", got, err)
	}
}
