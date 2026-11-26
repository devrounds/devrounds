package flow

import (
	"sort"
	"strings"
	"time"
)

// Suggestion is a command sequence the user repeated.
type Suggestion struct {
	Project  string    `json:"project"`
	Dir      string    `json:"dir"`
	Steps    []string  `json:"steps"`
	Count    int       `json:"count"`
	LastSeen time.Time `json:"last_seen"`
}

// Options controls how repeated sequences are found.
type Options struct {
	MinCount int
	MinLen   int
	MaxLen   int
	Gap      time.Duration
	Project  string
}

// DefaultOptions are the settings flow suggest uses.
func DefaultOptions() Options {
	return Options{MinCount: 3, MinLen: 2, MaxLen: 5, Gap: 10 * time.Minute}
}

var trivialCommands = map[string]bool{
	"cd": true, "ls": true, "dir": true, "pwd": true, "clear": true, "cls": true,
	"history": true, "exit": true, "sl": true, "set-location": true, "gci": true,
	"get-childitem": true, "get-location": true, "man": true, "help": true,
}

func isTrivial(cmd string) bool {
	f := strings.Fields(cmd)
	return len(f) == 0 || trivialCommands[strings.ToLower(f[0])]
}

type session struct {
	project string
	cmds    []string
	times   []time.Time
	dirs    []string
}

// buildSessions splits records (oldest first) into runs of successful, non-trivial
// commands in one project. A failed command, a project change or a long pause ends a run.
func buildSessions(records []Record, o Options) []session {
	var sessions []session
	idx := -1
	var lastTime time.Time
	for _, r := range records {
		if o.Project != "" && r.Project != o.Project {
			continue
		}
		if r.ExitCode != 0 {
			idx = -1
			continue
		}
		if idx >= 0 && (sessions[idx].project != r.Project || r.Time.Sub(lastTime) > o.Gap) {
			idx = -1
		}
		lastTime = r.Time
		if isTrivial(r.Command) {
			continue
		}
		if idx < 0 {
			sessions = append(sessions, session{project: r.Project})
			idx = len(sessions) - 1
		}
		s := &sessions[idx]
		s.cmds = append(s.cmds, strings.Join(strings.Fields(r.Command), " "))
		s.times = append(s.times, r.Time)
		s.dirs = append(s.dirs, r.Dir)
	}
	return sessions
}

type candidate struct {
	project string
	steps   []string
	dir     string
	count   int
	last    time.Time
}

func allSame(seq []string) bool {
	for _, c := range seq[1:] {
		if c != seq[0] {
			return false
		}
	}
	return true
}

// contains reports whether b appears inside a as a run of consecutive steps.
func contains(a, b []string) bool {
	for i := 0; i+len(b) <= len(a); i++ {
		match := true
		for j := range b {
			if a[i+j] != b[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// Suggest finds command sequences that were repeated. records must be oldest first.
func Suggest(records []Record, o Options) []Suggestion {
	d := DefaultOptions()
	if o.MinCount < 1 {
		o.MinCount = d.MinCount
	}
	if o.MinLen < 2 {
		o.MinLen = d.MinLen
	}
	if o.MaxLen < o.MinLen {
		o.MaxLen = d.MaxLen
	}
	if o.Gap <= 0 {
		o.Gap = d.Gap
	}

	found := map[string]*candidate{}
	for _, s := range buildSessions(records, o) {
		lastEnd := map[string]int{}
		for n := o.MinLen; n <= o.MaxLen; n++ {
			for i := 0; i+n <= len(s.cmds); i++ {
				seq := s.cmds[i : i+n]
				if allSame(seq) {
					continue
				}
				key := s.project + "\x00" + strings.Join(seq, "\x1f")
				if end, ok := lastEnd[key]; ok && i < end {
					continue
				}
				lastEnd[key] = i + n
				c := found[key]
				if c == nil {
					c = &candidate{project: s.project, steps: append([]string(nil), seq...)}
					found[key] = c
				}
				c.count++
				if t := s.times[i+n-1]; t.After(c.last) {
					c.last = t
					c.dir = s.dirs[i+n-1]
				}
			}
		}
	}

	var list []*candidate
	for _, c := range found {
		if c.count >= o.MinCount {
			list = append(list, c)
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if len(list[i].steps) != len(list[j].steps) {
			return len(list[i].steps) > len(list[j].steps)
		}
		if list[i].count != list[j].count {
			return list[i].count > list[j].count
		}
		return strings.Join(list[i].steps, "\x1f") < strings.Join(list[j].steps, "\x1f")
	})
	var kept []*candidate
	for _, c := range list {
		dup := false
		for _, k := range kept {
			if k.project == c.project && len(k.steps) > len(c.steps) && k.count == c.count && contains(k.steps, c.steps) {
				dup = true
				break
			}
		}
		if !dup {
			kept = append(kept, c)
		}
	}
	sort.Slice(kept, func(i, j int) bool {
		si, sj := kept[i].count*len(kept[i].steps), kept[j].count*len(kept[j].steps)
		if si != sj {
			return si > sj
		}
		if !kept[i].last.Equal(kept[j].last) {
			return kept[i].last.After(kept[j].last)
		}
		return strings.Join(kept[i].steps, "\x1f") < strings.Join(kept[j].steps, "\x1f")
	})

	out := []Suggestion{}
	for _, c := range kept {
		out = append(out, Suggestion{Project: c.project, Dir: c.dir, Steps: c.steps, Count: c.count, LastSeen: c.last})
	}
	return out
}
