package scan

import (
	"bytes"
	"strings"
	"testing"
)

func render(b Briefing) string {
	var buf bytes.Buffer
	PrintMarkdown(&buf, b)
	return buf.String()
}

func TestPrintMarkdownSections(t *testing.T) {
	b := Brief(Complete{
		Full: Full{
			Report: Report{
				Project:  "demo",
				Types:    []string{"Python"},
				Signals:  []string{"requirements.txt"},
				Commands: []string{"pip install -r requirements.txt"},
				Missing:  []string{"No README"},
			},
			Risks:   []string{"debug flag on: settings.py:1"},
			History: &History{Commits: 2, LastActivity: "2026-01-02", Contributors: []Count{{"Ann", 2}}},
		},
		Deps: []string{"Requirements without a version (1): flask"},
	})
	out := render(b)
	for _, want := range []string{
		"# Takeover report: demo\n",
		"- **Type:** Python (requirements.txt)\n",
		"- **Runtime:** -\n",
		"## Run commands\n```\npip install -r requirements.txt\n```\n",
		"## Gaps\n- No README\n",
		"## History\n- 2 commits, last activity 2026-01-02\n- Contributors: Ann (2)\n",
		"## Risks\n- debug flag on: settings.py:1\n",
		"## Dependencies\n- Requirements without a version (1): flask\n",
		"## Start here\n```\n",
		"## Questions for the previous maintainer\n- Where is this project documented, and how is it deployed?\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output is missing %q\n%s", want, out)
		}
	}
	if n := strings.Count(out, "```"); n%2 != 0 {
		t.Errorf("unbalanced code fences: %d", n)
	}
}

func TestPrintMarkdownEmptyAndNoHistory(t *testing.T) {
	out := render(Brief(Complete{Full: Full{Report: Report{
		Project: "x",
		Types:   []string{"Other (generic fallback)"},
	}}}))
	for _, want := range []string{
		"- **Type:** Other (generic fallback)\n",
		"- **Entry points:** -\n",
		"## Run commands\n- none found\n",
		"## Gaps\n- nothing obvious\n",
		"## Risks\n- nothing obvious\n",
		"## Dependencies\n- nothing obvious\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output is missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "## History") {
		t.Error("History section should be left out when there is no git history")
	}
}
