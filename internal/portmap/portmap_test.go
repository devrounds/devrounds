package portmap

import (
	"bytes"
	"fmt"
	"os"
	"reflect"
	"strings"
	"testing"
)

// TestMain lets the test binary stand in for docker when DEVROUNDS_FAKE_DOCKER is set.
func TestMain(m *testing.M) {
	if os.Getenv("DEVROUNDS_FAKE_DOCKER") == "1" {
		if os.Getenv("DEVROUNDS_FAKE_DAEMON") == "down" {
			fmt.Fprintln(os.Stderr, "Cannot connect to the Docker daemon")
			os.Exit(1)
		}
		fmt.Print(os.Getenv("DEVROUNDS_FAKE_PS"))
		os.Exit(0)
	}
	os.Exit(m.Run())
}

const fakePS = "pg|postgres:16|shop|db|0.0.0.0:5432->5432/tcp, :::5432->5432/tcp|Up 2 hours\n" +
	"web|nginx|||0.0.0.0:8080->80/tcp|Up 5 minutes\n"

func TestParseHostPorts(t *testing.T) {
	cases := map[string][]int{
		"0.0.0.0:5432->5432/tcp, :::5432->5432/tcp":      {5432},
		"127.0.0.1:8080->80/tcp, 0.0.0.0:9090->9090/tcp": {8080, 9090},
		"5432/tcp": nil,
		"":         nil,
	}
	for in, want := range cases {
		if got := parseHostPorts(in); !reflect.DeepEqual(got, want) {
			t.Errorf("parseHostPorts(%q) = %v, want %v", in, got, want)
		}
	}
}

func TestParseContainers(t *testing.T) {
	cs := parseContainers(fakePS + "broken line\n")
	if len(cs) != 2 {
		t.Fatalf("got %d containers, want 2: %+v", len(cs), cs)
	}
	if cs[0].Name != "pg" || cs[0].Project != "shop" || cs[0].Service != "db" || !reflect.DeepEqual(cs[0].HostPorts, []int{5432}) {
		t.Errorf("first container = %+v", cs[0])
	}
	if cs[1].Project != "" || cs[1].Status != "Up 5 minutes" {
		t.Errorf("second container = %+v", cs[1])
	}
}

func TestBuild(t *testing.T) {
	entries := []Entry{{Port: 3000, PID: 1}, {Port: 5432, PID: 2}}
	rep := Build(entries, parseContainers(fakePS), true)
	if len(rep.Ports) != 2 || rep.Ports[0].Container != "" || rep.Ports[1].Container != "pg" {
		t.Errorf("rows = %+v", rep.Ports)
	}
	empty := Build(nil, nil, false)
	if empty.Ports == nil || empty.Containers == nil {
		t.Error("slices should be empty, not nil, so JSON shows [] instead of null")
	}
}

func TestPrintText(t *testing.T) {
	var buf bytes.Buffer
	PrintText(&buf, Build([]Entry{{Port: 5432, PID: 2, Process: "docker.exe"}}, parseContainers(fakePS), true))
	out := buf.String()
	for _, want := range []string{"PORT", "CONTAINER", "postgres:16", "shop/db", "Up 2 hours"} {
		if !strings.Contains(out, want) {
			t.Errorf("output is missing %q\n%s", want, out)
		}
	}
	if strings.Contains(out, "not listed") {
		t.Error("the Docker-missing note should not appear when Docker answered")
	}

	buf.Reset()
	PrintText(&buf, Build(nil, nil, false))
	if !strings.Contains(buf.String(), "containers are not listed") {
		t.Errorf("missing the Docker-missing note:\n%s", buf.String())
	}
}

func useFakeDocker(t *testing.T, daemon, ps string) {
	t.Helper()
	old := dockerBin
	dockerBin = os.Args[0]
	t.Cleanup(func() { dockerBin = old })
	t.Setenv("DEVROUNDS_FAKE_DOCKER", "1")
	t.Setenv("DEVROUNDS_FAKE_DAEMON", daemon)
	t.Setenv("DEVROUNDS_FAKE_PS", ps)
}

func TestContainersWithFakeDocker(t *testing.T) {
	useFakeDocker(t, "up", fakePS)
	cs, ok := Containers()
	if !ok || len(cs) != 2 {
		t.Errorf("running daemon: ok=%v containers=%+v", ok, cs)
	}

	useFakeDocker(t, "down", "")
	if cs, ok := Containers(); ok || cs != nil {
		t.Errorf("daemon down: ok=%v containers=%+v", ok, cs)
	}
}
