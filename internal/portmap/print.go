package portmap

import (
	"fmt"
	"io"
	"os"
	"text/tabwriter"
)

func dash(s string) string {
	if s == "" {
		return "-"
	}
	return s
}

// PrintText writes the ports table, then the containers table when there are any.
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
	if !r.Docker {
		fmt.Fprintln(w)
		fmt.Fprintln(w, "Docker was not found or is not running, so containers are not listed.")
	}
}

var _ = os.Stdout
