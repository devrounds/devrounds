package envdrift

// appendLoc records a location, merging a lineless entry with a numbered one from the same file.
func appendLoc(m map[string][]Loc, name string, loc Loc) {
	locs := m[name]
	for i, l := range locs {
		if l.File != loc.File {
			continue
		}
		if l.Line == 0 && loc.Line > 0 {
			locs[i] = loc
			return
		}
		if loc.Line == 0 || l.Line == loc.Line {
			return
		}
	}
	m[name] = append(locs, loc)
}
