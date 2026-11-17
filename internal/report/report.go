package report

import (
	"fmt"
	"strings"
)

type Status int

const (
	OK Status = iota
	Warn
	Fail
)

func (s Status) String() string {
	switch s {
	case OK:
		return "OK"
	case Warn:
		return "WARN"
	}
	return "FAIL"
}

type Result struct {
	Status  Status
	Name    string
	Message string
	Details []string
	Next    string
}

// Print writes the report and returns true when there is no FAIL.
func Print(title string, results []Result) bool {
	fmt.Printf("%s\n\n", title)
	issues, fails := 0, 0
	var hints []string
	for _, r := range results {
		fmt.Printf("%-5s %-9s %s\n", r.Status, r.Name, r.Message)
		for _, d := range r.Details {
			fmt.Printf("%16s- %s\n", "", d)
		}
		if r.Next != "" {
			hints = append(hints, r.Next)
		}
		switch r.Status {
		case Warn:
			issues++
		case Fail:
			issues++
			fails++
		}
	}
	fmt.Println()
	word := "issues"
	if issues == 1 {
		word = "issue"
	}
	switch {
	case fails > 0:
		fmt.Printf("Result: NOT READY (%d %s)\n", issues, word)
	case issues > 0:
		fmt.Printf("Result: READY with %d %s\n", issues, word)
	default:
		fmt.Println("Result: READY")
	}
	if len(hints) > 0 {
		fmt.Printf("Next: %s\n", strings.Join(hints, ", then "))
	}
	return fails == 0
}
