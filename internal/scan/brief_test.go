package scan

import (
	"reflect"
	"testing"
	"time"
)

const (
	qProd   = "What runs in production that is not in this repo (cron jobs, manual steps, hosted settings)?"
	qBreaks = "What breaks most often, and what would you fix first?"
)

func TestBriefBusyNodeProject(t *testing.T) {
	c := Complete{
		Full: Full{
			Report: Report{
				Types:   []string{"Node"},
				Signals: []string{"package.json", "docker-compose.yml"},
				Missing: []string{"No README", "No CI configuration found"},
			},
			Risks:   []string{"possible secret file committed: .env", "debug flag on: app.js:3"},
			History: &History{Commits: 10, LastActivity: "2020-01-01"},
		},
		Deps: []string{"No lockfile (package-lock.json, yarn.lock or pnpm-lock.yaml), so installs are not reproducible"},
	}
	b := Brief(c)

	wantCmds := []string{
		"git log --oneline -10   # what changed recently",
		"git status -sb   # current branch and uncommitted work",
		"devrounds check   # is this machine ready to run it?",
		"docker compose config --services   # list the services without starting them",
		"npm run   # list the project's scripts",
	}
	if !reflect.DeepEqual(b.FirstCommands, wantCmds) {
		t.Errorf("commands = %v, want %v", b.FirstCommands, wantCmds)
	}

	if len(b.Questions) != 8 {
		t.Fatalf("got %d questions, want 8: %v", len(b.Questions), b.Questions)
	}
	if b.Questions[0] != "Where is this project documented, and how is it deployed?" {
		t.Errorf("first question = %q", b.Questions[0])
	}
	if !hasItem(b.Questions, "There has been no activity since 2020-01-01. Is this project still in use?") {
		t.Error("missing the quiet-project question")
	}
	if !hasItem(b.Questions, qProd) {
		t.Error("missing the production question")
	}
	if hasItem(b.Questions, qBreaks) {
		t.Error("the last general question should be cut by the cap of 8")
	}
}

func TestBriefMinimalProject(t *testing.T) {
	c := Complete{Full: Full{Report: Report{Types: []string{"Other (generic fallback)"}}}}
	b := Brief(c)

	wantCmds := []string{
		"devrounds check   # is this machine ready to run it?",
		"devrounds map   # what is already listening on which port",
	}
	if !reflect.DeepEqual(b.FirstCommands, wantCmds) {
		t.Errorf("commands = %v, want %v", b.FirstCommands, wantCmds)
	}
	wantQs := []string{qProd, qBreaks}
	if !reflect.DeepEqual(b.Questions, wantQs) {
		t.Errorf("questions = %v, want %v", b.Questions, wantQs)
	}
}

func TestBriefPythonProject(t *testing.T) {
	c := Complete{Full: Full{Report: Report{
		Types:   []string{"Python"},
		Missing: []string{"No pinned Python version (.python-version or requires-python)"},
	}}}
	b := Brief(c)

	wantCmds := []string{
		"devrounds check   # is this machine ready to run it?",
		"pip list   # what is already installed",
		"devrounds env   # which environment variables are used and documented",
		"devrounds map   # what is already listening on which port",
	}
	if !reflect.DeepEqual(b.FirstCommands, wantCmds) {
		t.Errorf("commands = %v, want %v", b.FirstCommands, wantCmds)
	}
	if !hasItem(b.Questions, "Which runtime version does production use?") {
		t.Errorf("missing the runtime question: %v", b.Questions)
	}
}

func TestBriefRecentActivityAddsNoQuietQuestion(t *testing.T) {
	today := time.Now().Format("2006-01-02")
	c := Complete{Full: Full{History: &History{Commits: 3, LastActivity: today}}}
	b := Brief(c)
	for _, q := range b.Questions {
		if len(q) > 25 && q[:25] == "There has been no activit" {
			t.Errorf("unexpected quiet-project question for recent activity: %q", q)
		}
	}
}
