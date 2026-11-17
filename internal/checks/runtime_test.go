package checks

import "testing"

func TestMatches(t *testing.T) {
	cases := []struct {
		required, installed string
		want                bool
	}{
		{"18", "18.17.0", true},
		{"18.17", "18.17.0", true},
		{"18.17.0", "18.17.0", true},
		{"18.18", "18.17.0", false},
		{"20", "18.17.0", false},
		{"3.11.4", "3.11", false},
	}
	for _, c := range cases {
		if got := matches(c.required, c.installed); got != c.want {
			t.Errorf("matches(%q, %q) = %v, want %v", c.required, c.installed, got, c.want)
		}
	}
}

func TestFirstVersion(t *testing.T) {
	cases := map[string]string{
		"v18.17.0":      "18.17.0",
		"Python 3.11.4": "3.11.4",
		"node 20":       "20",
		"lts/iron":      "",
	}
	for in, want := range cases {
		if got := firstVersion(in); got != want {
			t.Errorf("firstVersion(%q) = %q, want %q", in, got, want)
		}
	}
}
