package portmap

import (
	"fmt"
	"io"
	"text/tabwriter"
)

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// PrintText writes the ports table, the containers table when there are any,
// the project's port conflicts when a project was given, and a note when Docker is missing.
func PrintText(w io.Writer, r Report) {
	tw := tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)
	fmt.Fprintln(tw, "PORT\tPROCESS\tPID\tFOLDER\tCONTAINER")
	for _, p := range r.Ports {
		fmt.Fprintf(tw, "%d\t%s\t%d\t%s\t%s\n", p.Port, dash(p.Process), p.PID, dash(p.Folder), dash(p.Container))
	}
	tw.Flush()

	if len(r.Containers) > 0 {
		fmt.Fprintln(w)
		tw = tabwriter.NewWriter(w, 0, 8, 2, ' ', 0)
		fmt.Fprintln(tw, "CONTAINER\tIMAGE\tCOMPOSE\tPORTS\tSTATUS")
		for _, c := range r.Containers {
			compose := "-"
			if c.Project != "" {
				compose = c.Project + "/" + c.Service
			}
			fmt.Fprintf(tw, "%s\t%s\t%s\t%s\t%s\n", c.Name, c.Image, compose, dash(c.Ports), c.Status)
		}
		tw.Flush()
	}

	if p := r.Project; p != nil {
		fmt.Fprintln(w)
		switch {
		case p.Expected == 0:
			fmt.Fprintf(w, "%s names no specific ports (looked in Compose files and .env.example).\n", p.Name)
		case len(p.Conflicts) == 0:
			fmt.Fprintf(w, "No port conflicts for %s (%d expected ports checked).\n", p.Name, p.Expected)
		default:
			fmt.Fprintf(w, "Port conflicts for %s:\n", p.Name)
			for _, c := range p.Conflicts {
				fmt.Fprintf(w, "  %d (%s) is held by %s\n", c.Port, c.Source, c.Holder)
			}
		}
	}

	if !r.Docker {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Docker was not found or is not running, so containers are not listed.")
	}
}
