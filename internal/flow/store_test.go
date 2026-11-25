package flow

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

func openTemp(t *testing.T) *Store {
	t.Helper()
	s, err := Open(filepath.Join(t.TempDir(), "sub", "flow.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { s.Close() })
	return s
}

func mustAdd(t *testing.T, s *Store, r Record) bool {
	t.Helper()
	ok, err := s.Add(r)
	if err != nil {
		t.Fatal(err)
	}
	return ok
}

func TestStoreAddAndRecent(t *testing.T) {
	s := openTemp(t)
	base := time.Now().Add(-time.Hour)
	mustAdd(t, s, Record{Time: base, Command: "git status", Dir: "/p", Project: "p"})
	mustAdd(t, s, Record{Time: base.Add(time.Minute), Command: "go test ./...", Dir: "/p", Project: "p", ExitCode: 1, Duration: 1500 * time.Millisecond})
	mustAdd(t, s, Record{Time: base.Add(2 * time.Minute), Command: "curl --token abc123 https://x", Dir: "/p", Project: "p"})

	got, err := s.Recent(2)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d records, want 2", len(got))
	}
	if strings.Contains(got[0].Command, "abc123") || !strings.Contains(got[0].Command, "<redacted>") {
		t.Errorf("the secret must never be stored: %q", got[0].Command)
	}
	if got[1].Command != "go test ./..." || got[1].ExitCode != 1 || got[1].Duration != 1500*time.Millisecond || got[1].Project != "p" {
		t.Errorf("second record = %+v", got[1])
	}
	if n, _ := s.Count(); n != 3 {
		t.Errorf("Count = %d, want 3", n)
	}
}

func TestStoreSkipsWhatMustNotBeRecorded(t *testing.T) {
	s := openTemp(t)
	for _, c := range []string{"", " echo hidden", "devrounds flow suggest"} {
		if mustAdd(t, s, Record{Command: c}) {
			t.Errorf("%q should not be recorded", c)
		}
	}
	if n, _ := s.Count(); n != 0 {
		t.Errorf("Count = %d, want 0", n)
	}
}

func TestStoreOnOffAndForget(t *testing.T) {
	s := openTemp(t)
	if !s.Enabled() {
		t.Fatal("recording should be on by default")
	}
	if err := s.SetEnabled(false); err != nil {
		t.Fatal(err)
	}
	if s.Enabled() || mustAdd(t, s, Record{Command: "ls"}) {
		t.Error("nothing should be recorded while recording is off")
	}
	if err := s.SetEnabled(true); err != nil {
		t.Fatal(err)
	}
	if !mustAdd(t, s, Record{Command: "ls"}) {
		t.Error("recording should work again after turning it on")
	}
	if err := s.Forget(); err != nil {
		t.Fatal(err)
	}
	if n, _ := s.Count(); n != 0 {
		t.Errorf("Count after Forget = %d, want 0", n)
	}
}

func TestStoreRetention(t *testing.T) {
	s := openTemp(t)
	if mustAdd(t, s, Record{Time: time.Now().Add(-100 * 24 * time.Hour), Command: "old"}) {
		t.Error("a command older than the retention period must not be recorded")
	}
	if !mustAdd(t, s, Record{Time: time.Now().Add(-80 * 24 * time.Hour), Command: "recent enough"}) {
		t.Error("a command inside the retention period should be recorded")
	}
}

func TestStoreTwoShellsAtOnce(t *testing.T) {
	path := filepath.Join(t.TempDir(), "flow.db")
	a, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	b, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { b.Close() })

	var wg sync.WaitGroup
	for i, s := range []*Store{a, b} {
		wg.Add(1)
		go func(i int, s *Store) {
			defer wg.Done()
			for n := 0; n < 20; n++ {
				if _, err := s.Add(Record{Command: fmt.Sprintf("echo %d-%d", i, n)}); err != nil {
					t.Errorf("shell %d: %v", i, err)
				}
			}
		}(i, s)
	}
	wg.Wait()
	if n, _ := a.Count(); n != 40 {
		t.Errorf("Count = %d, want 40", n)
	}
}

func TestDefaultPathOverride(t *testing.T) {
	want := filepath.Join(t.TempDir(), "x.db")
	t.Setenv("DEVROUNDS_FLOW_DB", want)
	if got, err := DefaultPath(); err != nil || got != want {
		t.Errorf("DefaultPath = %q, %v", got, err)
	}
}

func TestProjectOf(t *testing.T) {
	root := filepath.Join(t.TempDir(), "shop")
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, "src", "api")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := ProjectOf(deep); got != "shop" {
		t.Errorf("ProjectOf(%q) = %q, want shop", deep, got)
	}
	plain := filepath.Join(t.TempDir(), "notes")
	if err := os.MkdirAll(plain, 0o755); err != nil {
		t.Fatal(err)
	}
	if got := ProjectOf(plain); got != "notes" {
		t.Errorf("ProjectOf(%q) = %q, want notes", plain, got)
	}
}
