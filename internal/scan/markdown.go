package scan

import (
	"fmt"
	"io"
	"strings"
)

const fence = "```"

func mdBullets(w io.Writer, items []string, empty string) {
	if len(items) == 0 {
		fmt.Fprintf(w, "- %s\n", empty)
		return
	}
	for _, it := range items {
		fmt.Fprintf(w, "- %s\n", it)
	}
}

func mdCode(w io.Writer, lines []string) {
	fmt.Fprintln(w, fence)
	for _, l := range lines {
		fmt.Fprintln(w, l)
	}
	fmt.Fprintln(w, fence)
}

func orDash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// PrintMarkdown writes the whole briefing as a Markdown document.
func PrintMarkdown(w io.Writer, b Briefing) {
	fmt.Fprintf(w, "# Takeover report: %s\n\n", b.Project)

	fmt.Fprintln(w, "## Overview")
	typ := strings.Join(b.Types, ", ")
	if len(b.Signals) > 0 {
		typ += " (" + strings.Join(b.Signals, ", ") + ")"
	}
	fmt.Fprintf(w, "- **Type:** %s\n", typ)
	fmt.Fprintf(w, "- **Runtime:** %s\n", orDash(b.Runtime))
	fmt.Fprintf(w, "- **Entry points:** %s\n\n", orDash(strings.Join(b.Entry, ", ")))

	fmt.Fprintln(w, "## Run commands")
	if len(b.Commands) == 0 {
		fmt.Fprintln(w, "- none found")
	} else {
		mdCode(w, b.Commands)
	}
	fmt.Fprintln(w)

	fmt.Fprintln(w, "## Gaps")
	mdBullets(w, b.Missing, "nothing obvious")
	fmt.Fprintln(w)

	if h := b.History; h != nil {
		fmt.Fprintln(w, "## History")
		noun := "commits"
		if h.Commits == 1 {
			noun = "commit"
		}
		fmt.Fprintf(w, "- %d %s, last activity %s\n", h.Commits, noun, h.LastActivity)
		if len(h.Contributors) > 0 {
			fmt.Fprintf(w, "- Contributors: %s\n", joinCounts(h.Contributors))
		}
		if len(h.ActiveAreas) > 0 {
			fmt.Fprintf(w, "- Active areas (file changes, 90 days): %s\n", joinCounts(h.ActiveAreas))
		}
		fmt.Fprintln(w)
	}

	fmt.Fprintln(w, "## Risks")
	mdBullets(w, b.Risks, "nothing obvious")
	fmt.Fprintln(w)

	fmt.Fprintln(w, "## Dependencies")
	mdBullets(w, b.Deps, "nothing obvious")
	fmt.Fprintln(w)

	fmt.Fprintln(w, "## Start here")
	mdCode(w, b.FirstCommands)
	fmt.Fprintln(w)

	fmt.Fprintln(w, "## Questions for the previous maintainer")
	mdBullets(w, b.Questions, "none")
}
