package scan

import (
	"fmt"
	"io"
	"strings"
	"time"
)

// Briefing is the scan report plus a suggested way in.
type Briefing struct {
	Complete
	FirstCommands []string `json:"first_safe_commands"`
	Questions     []string `json:"questions_for_previous_maintainer"`
}

func anyContains(list []string, subs ...string) bool {
	for _, item := range list {
		for _, s := range subs {
			if strings.Contains(item, s) {
				return true
			}
		}
	}
	return false
}

func hasItem(list []string, names ...string) bool {
	for _, item := range list {
		for _, n := range names {
			if item == n {
				return true
			}
		}
	}
	return false
}

// Brief picks up to five read-only commands and up to eight questions from the findings.
func Brief(c Complete) Briefing {
	b := Briefing{Complete: c, FirstCommands: []string{}, Questions: []string{}}
	node := hasItem(c.Types, "Node")
	python := hasItem(c.Types, "Python")
	compose := hasItem(c.Signals, "docker-compose.yml", "docker-compose.yaml", "compose.yml", "compose.yaml")

	var cmds []string
	if c.History != nil {
		cmds = append(cmds,
			"git log --oneline -10   # what changed recently",
			"git status -sb   # current branch and uncommitted work")
	}
	cmds = append(cmds, "devrounds check   # is this machine ready to run it?")
	if compose {
		cmds = append(cmds, "docker compose config --services   # list the services without starting them")
	}
	switch {
	case node:
		cmds = append(cmds, "npm run   # list the project's scripts")
	case python:
		cmds = append(cmds, "pip list   # what is already installed")
	}
	if node || python {
		cmds = append(cmds, "devrounds env   # which environment variables are used and documented")
	}
	cmds = append(cmds, "devrounds map   # what is already listening on which port")
	if len(cmds) > 5 {
		cmds = cmds[:5]
	}
	b.FirstCommands = cmds

	q := func(s string) { b.Questions = append(b.Questions, s) }
	if anyContains(c.Missing, "No README") {
		q("Where is this project documented, and how is it deployed?")
	}
	if anyContains(c.Missing, "No pinned") {
		q("Which runtime version does production use?")
	}
	if anyContains(c.Missing, "environment variables") {
		q("Which environment variables are required, and where do the real values come from?")
	}
	if compose {
		q("Which services (databases, queues, caches) must be running locally, and with what data?")
	}
	if anyContains(c.Missing, "No CI") {
		q("How is this tested and released today?")
	}
	if anyContains(c.Risks, "possible secret file") {
		q("Have the secret files in this project been rotated, and who has had access?")
	}
	if anyContains(c.Risks, "debug flag") {
		q("Is debug mode really off in production?")
	}
	if anyContains(c.Deps, "any version", "No lockfile", "without a version") {
		q("Which dependency versions are known to work together?")
	}
	if c.History != nil {
		if t, err := time.Parse("2006-01-02", c.History.LastActivity); err == nil && time.Since(t) > 180*24*time.Hour {
			q("There has been no activity since " + c.History.LastActivity + ". Is this project still in use?")
		}
	}
	q("What runs in production that is not in this repo (cron jobs, manual steps, hosted settings)?")
	q("What breaks most often, and what would you fix first?")
	if len(b.Questions) > 8 {
		b.Questions = b.Questions[:8]
	}
	return b
}

// PrintBriefing writes the full report, then the suggested commands and questions.
func PrintBriefing(w io.Writer, b Briefing) {
	PrintComplete(w, b.Complete)
	for i, c := range b.FirstCommands {
		label := ""
		if i == 0 {
			label = "Start"
		}
		fmt.Fprintf(w, "%-9s %s\n", label, c)
	}
	for i, qn := range b.Questions {
		label := ""
		if i == 0 {
			label = "Ask"
		}
		fmt.Fprintf(w, "%-9s - %s\n", label, qn)
	}
}
