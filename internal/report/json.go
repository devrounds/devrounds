package report

import (
	"encoding/json"
	"os"
	"strings"
)

type jsonResult struct {
	Status  string   `json:"status"`
	Name    string   `json:"name"`
	Message string   `json:"message"`
	Details []string `json:"details,omitempty"`
	Next    string   `json:"next,omitempty"`
}

type jsonReport struct {
	Title   string       `json:"title"`
	Ready   bool         `json:"ready"`
	Issues  int          `json:"issues"`
	Results []jsonResult `json:"results"`
}

// PrintJSON writes the report as JSON and returns true when there is no FAIL.
func PrintJSON(title string, results []Result) bool {
	rep := jsonReport{Title: title, Ready: true, Results: []jsonResult{}}
	for _, r := range results {
		rep.Results = append(rep.Results, jsonResult{
			Status:  strings.ToLower(r.Status.String()),
			Name:    r.Name,
			Message: r.Message,
			Details: r.Details,
			Next:    r.Next,
		})
		if r.Status != OK {
			rep.Issues++
		}
		if r.Status == Fail {
			rep.Ready = false
		}
	}
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	_ = enc.Encode(rep)
	return rep.Ready
}
