package portmap

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func serverPort(t *testing.T, s *httptest.Server) int {
	t.Helper()
	return s.Listener.Addr().(*net.TCPAddr).Port
}

func TestCheckHealth(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) { w.Write([]byte("ok")) })
	mux.HandleFunc("/gone", func(w http.ResponseWriter, r *http.Request) { http.NotFound(w, r) })
	mux.HandleFunc("/broken", func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(500) })
	mux.HandleFunc("/old", func(w http.ResponseWriter, r *http.Request) { http.Redirect(w, r, "/health", http.StatusFound) })
	s := httptest.NewServer(mux)
	defer s.Close()
	port := serverPort(t, s)

	cases := []struct {
		path    string
		status  int
		verdict string
	}{
		{"/health", 200, "healthy"},
		{"gone", 404, "answered"},
		{"/broken", 500, "unhealthy"},
		{"/old", 302, "answered"},
	}
	for _, c := range cases {
		got := CheckHealth([]int{port}, c.path, time.Second)
		if len(got) != 1 || got[0].Status != c.status || got[0].Verdict != c.verdict {
			t.Errorf("path %q: got %+v, want status %d verdict %q", c.path, got, c.status, c.verdict)
		}
	}
	if got := CheckHealth([]int{port}, "", time.Second); got[0].Status != 200 || !strings.HasSuffix(got[0].URL, "/health") {
		t.Errorf("empty path should default to /health: %+v", got)
	}
}

func TestCheckHealthNoAnswerAndSkip(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	closed := l.Addr().(*net.TCPAddr).Port
	l.Close()
	got := CheckHealth([]int{closed}, "/health", 500*time.Millisecond)
	if len(got) != 1 || got[0].Verdict != "no HTTP answer" {
		t.Errorf("closed port: %+v", got)
	}
	got = CheckHealth([]int{5432}, "/health", time.Second)
	if len(got) != 1 || !strings.HasPrefix(got[0].Verdict, "skipped") || got[0].Status != 0 {
		t.Errorf("non-HTTP port should be skipped: %+v", got)
	}
}

func TestHealthTargets(t *testing.T) {
	rep := Report{
		Ports: []Row{
			{Entry: Entry{Port: 3000}}, {Entry: Entry{Port: 4000}}, {Entry: Entry{Port: 5432}},
		},
		Project: &ProjectInfo{Conflicts: []Conflict{{Port: 4000}}},
	}
	expected := []Expected{{3000, "a"}, {4000, "b"}, {5432, "c"}, {9000, "d"}, {3000, "e"}}
	got := HealthTargets(rep, expected)
	if want := []int{3000, 5432}; !reflect.DeepEqual(got, want) {
		t.Errorf("HealthTargets = %v, want %v", got, want)
	}
	if got := HealthTargets(Report{}, expected); len(got) != 0 {
		t.Errorf("without a project nothing is probed, got %v", got)
	}
}

func TestPrintTextHealth(t *testing.T) {
	var buf bytes.Buffer
	rep := Build(nil, nil, true)
	rep.Health = []HealthResult{
		{Port: 8080, URL: "http://127.0.0.1:8080/health", Status: 200, Verdict: "healthy", Millis: 5},
		{Port: 9000, URL: "http://127.0.0.1:9000/health", Verdict: "no HTTP answer"},
	}
	PrintText(&buf, rep)
	out := buf.String()
	for _, want := range []string{"Health checks", "http://127.0.0.1:8080/health  healthy (200, 5 ms)", "9000/health  no HTTP answer"} {
		if !strings.Contains(out, want) {
			t.Errorf("output is missing %q\n%s", want, out)
		}
	}
}
