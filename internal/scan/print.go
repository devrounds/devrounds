package scan

import (
	"fmt"
	"io"
	"strings"
)

// PrintText writes the report as plain text.
func PrintText(w io.Writer, r Report) {
	fmt.Fprintf(w, "Scan of %s\n\n", r.Project)
	row := func(label string, lines []string) {
		if len(lines) == 0 {
			fmt.Fprintf(w, "%-9s -\n", label)
			return
		}
		for i, l := range lines {
			if i == 0 {
				fmt.Fprintf(w, "%-9s %s\n", label, l)
			} else {
				fmt.Fprintf(w, "%-9s %s\n", "", l)
			}
		}
	}
	typ := strings.Join(r.Types, ", ")
	if len(r.Signals) > 0 {
		typ += " (" + strings.Join(r.Signals, ", ") + ")"
	}
	row("Type", []string{typ})
	var runtime []string
	if r.Runtime != "" {
		runtime = []string{r.Runtime}
	}
	row("Runtime", runtime)
	var entry []string
	if len(r.Entry) > 0 {
		entry = []string{strings.Join(r.Entry, ", ")}
	}
	row("Entry", entry)
	row("Run", r.Commands)
	if len(r.Missing) == 0 {
		row("Missing", []string{"nothing obvious"})
		return
	}
	lines := make([]string, len(r.Missing))
	for i, m := range r.Missing {
		lines[i] = "- " + m
	}
	row("Missing", lines)
}
