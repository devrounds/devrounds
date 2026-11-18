package scan

import (
	"sort"
	"strings"
)

// mergeContributors combines names that differ only by case or spacing and keeps the top three.
func mergeContributors(in []Count) []Count {
	byKey := map[string]*Count{}
	var order []string
	for _, c := range in {
		key := strings.ToLower(strings.Join(strings.Fields(c.Name), " "))
		if e, ok := byKey[key]; ok {
			e.Count += c.Count
			continue
		}
		cc := c
		byKey[key] = &cc
		order = append(order, key)
	}
	out := make([]Count, 0, len(order))
	for _, k := range order {
		out = append(out, *byKey[k])
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Count > out[j].Count })
	if len(out) > 3 {
		out = out[:3]
	}
	return out
}
