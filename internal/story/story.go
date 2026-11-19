package story

import (
	"fmt"
	"os/exec"
	"regexp"
	"sort"
	"strings"
)

// Commit is one commit in the work log.
type Commit struct {
	Hash    string `json:"hash"`
	Date    string `json:"date"`
	Subject string `json:"subject"`
	Area    string `json:"area"`
	Files   int    `json:"files"`
}

// Group is a set of commits sharing an area or a day.
type Group struct {
	Name    string   `json:"name"`
	Commits []Commit `json:"commits"`
}

// Log is the whole work log.
type Log struct {
	Project string  `json:"project"`
	Period  string  `json:"period"`
	By      string  `json:"grouped_by"`
	Author  string  `json:"author,omitempty"`
	Total   int     `json:"total_commits"`
	Groups  []Group `json:"groups"`
}

var relRe = regexp.MustCompile(`^(\d+)([dw])$`)

// SinceArg turns 7d or 2w into a git date. Anything else is passed to git as is.
func SinceArg(s string) string {
	s = strings.TrimSpace(s)
	if m := relRe.FindStringSubmatch(s); m != nil {
		unit := "days"
		if m[2] == "w" {
			unit = "weeks"
		}
		return m[1] + " " + unit + " ago"
	}
	return s
}

// Period describes the range in words for the report title.
func Period(since, until string) string {
	since = strings.TrimSpace(since)
	s := "since " + since
	if m := relRe.FindStringSubmatch(since); m != nil {
		unit := "day"
		if m[2] == "w" {
			unit = "week"
		}
		if m[1] != "1" {
			unit += "s"
		}
		s = "last " + m[1] + " " + unit
	}
	if until != "" {
		s += " until " + until
	}
	return s
}

func gitOut(dir string, args ...string) (string, error) {
	full := append([]string{"-C", dir, "-c", "core.quotepath=false"}, args...)
	out, err := exec.Command("git", full...).Output()
	return string(out), err
}

// CurrentUser returns the git user.name and user.email configured for dir.
func CurrentUser(dir string) (string, string) {
	name, _ := gitOut(dir, "config", "user.name")
	email, _ := gitOut(dir, "config", "user.email")
	return strings.TrimSpace(name), strings.TrimSpace(email)
}

// areaOf picks the top-level folder that most of a commit's files are in.
func areaOf(files []string) string {
	if len(files) == 0 {
		return "(no files)"
	}
	count := map[string]int{}
	for _, f := range files {
		seg := "(root)"
		if i := strings.Index(f, "/"); i > 0 {
			seg = f[:i]
		}
		count[seg]++
	}
	best, n := "", 0
	for seg, c := range count {
		if c > n || (c == n && seg < best) {
			best, n = seg, c
		}
	}
	return best
}

// Commits reads non-merge commits in the range, newest first.
func Commits(dir, since, until, author string) ([]Commit, error) {
	if _, err := gitOut(dir, "rev-parse", "--is-inside-work-tree"); err != nil {
		return nil, fmt.Errorf("not a git repository: %s", dir)
	}
	args := []string{"log", "--no-merges", "--since=" + SinceArg(since), "--name-only",
		"--pretty=format:%x1e%h%x1f%cs%x1f%s"}
	if until != "" {
		args = append(args, "--until="+SinceArg(until))
	}
	if author != "" {
		args = append(args, "--regexp-ignore-case", "--author="+author)
	}
	out, err := gitOut(dir, args...)
	if err != nil {
		return nil, nil
	}
	var commits []Commit
	for _, rec := range strings.Split(out, "\x1e") {
		if strings.TrimSpace(rec) == "" {
			continue
		}
		lines := strings.Split(rec, "\n")
		parts := strings.SplitN(strings.TrimRight(lines[0], "\r"), "\x1f", 3)
		if len(parts) != 3 {
			continue
		}
		var files []string
		for _, l := range lines[1:] {
			if l = strings.TrimSpace(l); l != "" {
				files = append(files, l)
			}
		}
		commits = append(commits, Commit{
			Hash: parts[0], Date: parts[1], Subject: parts[2],
			Area: areaOf(files), Files: len(files),
		})
	}
	return commits, nil
}

// GroupBy groups commits by "area" or "day".
func GroupBy(commits []Commit, by string) []Group {
	idx := map[string]int{}
	var groups []Group
	for _, c := range commits {
		key := c.Area
		if by == "day" {
			key = c.Date
		}
		i, ok := idx[key]
		if !ok {
			i = len(groups)
			idx[key] = i
			groups = append(groups, Group{Name: key})
		}
		groups[i].Commits = append(groups[i].Commits, c)
	}
	if by == "day" {
		sort.SliceStable(groups, func(i, j int) bool { return groups[i].Name > groups[j].Name })
	} else {
		sort.SliceStable(groups, func(i, j int) bool {
			if len(groups[i].Commits) != len(groups[j].Commits) {
				return len(groups[i].Commits) > len(groups[j].Commits)
			}
			return groups[i].Name < groups[j].Name
		})
	}
	return groups
}
