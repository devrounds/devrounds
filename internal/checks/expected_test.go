package checks

import (
	"reflect"
	"testing"
)

func TestExpectedPorts(t *testing.T) {
	dir := t.TempDir()
	put(t, dir, "package.json", `{}`)
	if got := ExpectedPorts(dir); len(got) != 0 {
		t.Errorf("a guessed default must not count: %v", got)
	}
	put(t, dir, "compose.yml", "services:\n  db:\n    image: x\n    ports:\n      - \"5432:5432\"\n")
	put(t, dir, ".env.example", "PORT=4000\n")
	got := ExpectedPorts(dir)
	want := []ExpectedPort{{5432, "compose.yml"}, {4000, ".env.example"}}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("ExpectedPorts = %v, want %v", got, want)
	}
}
