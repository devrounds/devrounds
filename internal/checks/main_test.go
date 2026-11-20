package checks

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// TestMain lets the test binary stand in for docker when DEVROUNDS_FAKE_DOCKER is set.
func TestMain(m *testing.M) {
	if os.Getenv("DEVROUNDS_FAKE_DOCKER") == "1" {
		fakeDocker()
	}
	os.Exit(m.Run())
}

func fakeDocker() {
	has := func(word string) bool {
		for _, a := range os.Args[1:] {
			if a == word {
				return true
			}
		}
		return false
	}
	switch {
	case has("info"):
		if os.Getenv("DEVROUNDS_FAKE_DAEMON") == "down" {
			fmt.Fprintln(os.Stderr, "Cannot connect to the Docker daemon")
			os.Exit(1)
		}
		fmt.Println("27.0.0")
	case has("ps"):
		for _, s := range strings.Split(os.Getenv("DEVROUNDS_FAKE_RUNNING"), ",") {
			if s != "" {
				fmt.Println(s)
			}
		}
	}
	os.Exit(0)
}
