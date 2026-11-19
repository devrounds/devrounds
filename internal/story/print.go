package story

import (
	"fmt"
	"io"
	"strings"
)

func noun(n int) string {
	if n == 1 {
		return "commit"
	}
	return "commits"
}

func title(l Log) string {
	s := fmt.Sprintf("%s: %s, grouped by %s, %d %s", l.Project, l.Period, l.By, l.Total, noun(l.Total))
	if l.Author != "" {
		s += " by " + l.Author
	}
	return s
}

func detail(l Log, c Commit) string {
	if l.By == "day" {
		return "[" + c.Area + "]"
	}
	return c.Date
}

// PrintText writes the log as plain text.
func PrintText(w io.Writer, l Log) {
	fmt.Fprintln(w, title(l))
	for _, g := range l.Groups {
		fmt.Fprintf(w, "\n%s (%d)\n", g.Name, len(g.Commits))
		for _, c := range g.Commits {
			fmt.Fprintf(w, "  %s  %s\n", detail(l, c), c.Subject)
		}
	}
}

// PrintMarkdown writes the log as a Markdown document.
func PrintMarkdown(w io.Writer, l Log) {
	fmt.Fprintf(w, "# Work log: %s\n\n_%s_\n", l.Project, strings.TrimPrefix(title(l), l.Project+": "))
	for _, g := range l.Groups {
		fmt.Fprintf(w, "\n## %s (%d)\n\n", g.Name, len(g.Commits))
		for _, c := range g.Commits {
			fmt.Fprintf(w, "- %s (%s, `%s`)\n", c.Subject, detail(l, c), c.Hash)
		}
	}
}

// PrintStandup writes a short version: one line per group, up to three subjects each.
func PrintStandup(w io.Writer, l Log) {
	fmt.Fprintln(w, title(l))
	for _, g := range l.Groups {
		var subs []string
		for i, c := range g.Commits {
			if i == 3 {
				break
			}
			subs = append(subs, c.Subject)
		}
		line := fmt.Sprintf("- %s (%d): %s", g.Name, len(g.Commits), strings.Join(subs, "; "))
		if extra := len(g.Commits) - 3; extra > 0 {
			line += fmt.Sprintf(" +%d more", extra)
		}
		fmt.Fprintln(w, line)
	}
}
