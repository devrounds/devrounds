package scan

import (
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strconv"
	"strings"
)

// Count is a name with a number next to it.
type Count struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// History summarizes a project's git history. Names only, never emails.
type History struct {
	Commits      int     `json:"commits"`
	LastActivity string  `json:"last_activity"`
	Contributors []Count `json:"top_contributors"`
	ActiveAreas  []Count `json:"active_areas_90_days"`
}

func gitOut(dir string, args ...string) (string, bool) {
	full := append([]string{"-C", dir, "-c", "core.quotepath=false"}, args...)
	out, err := exec.Command("git", full...).Output()
	if err != nil {
		return "", false
	}
	return string(out), true
}

// Git summarizes the git history. It returns nil outside a git repo or in an empty one.
func Git(dir string) *History {
	countOut, ok := gitOut(dir, "rev-list", "--count", "HEAD")
	if !ok {
		return nil
	}
	commits, err := strconv.Atoi(strings.TrimSpace(countOut))
	if err != nil || commits == 0 {
		return nil
	}
	h := &History{Commits: commits}

	if out, ok := gitOut(dir, "log", "-1", "--format=%cs"); ok {
		h.LastActivity = strings.TrimSpace(out)
	}

	if out, ok := gitOut(dir, "shortlog", "-sn", "--no-merges", "HEAD"); ok {
		for _, line := range strings.Split(out, "\n") {
			parts := strings.SplitN(strings.TrimSpace(line), "\t", 2)
			if len(parts) != 2 {
				continue
			}
			n, err := strconv.Atoi(strings.TrimSpace(parts[0]))
			if err != nil {
				continue
			}
			h.Contributors = append(h.Contributors, Count{strings.TrimSpace(parts[1]), n})
			if len(h.Contributors) == 100 {
				break
			}
		}
	}

	h.Contributors = mergeContributors(h.Contributors)
	if out, ok := gitOut(dir, "log", "--since=90 days ago", "--name-only", "--pretty=format:", "--no-merges"); ok {
		areas := map[string]int{}
		for _, line := range strings.Split(out, "\n") {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			area := "(root files)"
			if i := strings.Index(line, "/"); i > 0 {
				area = line[:i]
			}
			areas[area]++
		}
		for name, n := range areas {
			h.ActiveAreas = append(h.ActiveAreas, Count{name, n})
		}
		sort.Slice(h.ActiveAreas, func(i, j int) bool {
			if h.ActiveAreas[i].Count != h.ActiveAreas[j].Count {
				return h.ActiveAreas[i].Count > h.ActiveAreas[j].Count
			}
			return h.ActiveAreas[i].Name < h.ActiveAreas[j].Name
		})
		if len(h.ActiveAreas) > 5 {
			h.ActiveAreas = h.ActiveAreas[:5]
		}
	}
	return h
}

func joinCounts(cs []Count) string {
	parts := make([]string, len(cs))
	for i, c := range cs {
		parts[i] = fmt.Sprintf("%s (%d)", c.Name, c.Count)
	}
	return strings.Join(parts, ", ")
}

func printHistory(w io.Writer, h *History) {
	if h == nil {
		return
	}
	noun := "commits"
	if h.Commits == 1 {
		noun = "commit"
	}
	fmt.Fprintf(w, "%-9s %d %s, last activity %s\n", "History", h.Commits, noun, h.LastActivity)
	if len(h.Contributors) > 0 {
		fmt.Fprintf(w, "%-9s Contributors: %s\n", "", joinCounts(h.Contributors))
	}
	if len(h.ActiveAreas) > 0 {
		fmt.Fprintf(w, "%-9s Active areas (file changes, 90 days): %s\n", "", joinCounts(h.ActiveAreas))
	}
}
