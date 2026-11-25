package flow

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestHook(t *testing.T) {
	for _, shell := range []string{"bash", "BASH", "zsh", "powershell", "pwsh", "PowerShell"} {
		got, err := Hook(shell)
		if err != nil || !strings.Contains(got, "devrounds flow record") || !strings.Contains(got, "--dir") {
			t.Errorf("Hook(%q) = %d bytes, %v", shell, len(got), err)
		}
	}
	if _, err := Hook("fish"); err == nil {
		t.Error("an unknown shell should be an error")
	}
}

func TestBashHookSyntax(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash on Windows may be the WSL launcher")
	}
	path, err := exec.LookPath("bash")
	if err != nil {
		t.Skip("bash not installed")
	}
	cmd := exec.Command(path, "-n")
	cmd.Stdin = strings.NewReader(bashHook)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Errorf("bash rejected the hook: %v\n%s", err, out)
	}
}

func TestPowerShellHookSyntax(t *testing.T) {
	name := "pwsh"
	if _, err := exec.LookPath(name); err != nil {
		name = "powershell"
	}
	path, err := exec.LookPath(name)
	if err != nil {
		t.Skip("PowerShell not installed")
	}
	file := filepath.Join(t.TempDir(), "hook.ps1")
	if err := os.WriteFile(file, []byte(powershellHook), 0o644); err != nil {
		t.Fatal(err)
	}
	quoted := strings.ReplaceAll(file, "'", "''")
	script := fmt.Sprintf(`$t = $null; $e = $null; [void][System.Management.Automation.Language.Parser]::ParseFile('%s', [ref]$t, [ref]$e); if ($e.Count -gt 0) { $e | ForEach-Object { $_.Message }; exit 1 }`, quoted)
	if out, err := exec.Command(path, "-NoProfile", "-NonInteractive", "-Command", script).CombinedOutput(); err != nil {
		t.Errorf("PowerShell rejected the hook: %v\n%s", err, out)
	}
}

func TestCapture(t *testing.T) {
	s := openTemp(t)
	root := filepath.Join(t.TempDir(), "shop")
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	ok, err := Capture(s, "go test ./...", root, 1, 250)
	if err != nil || !ok {
		t.Fatalf("Capture = %v, %v", ok, err)
	}
	got, err := s.Recent(1)
	if err != nil || len(got) != 1 {
		t.Fatalf("Recent = %v, %v", got, err)
	}
	if got[0].Project != "shop" || got[0].ExitCode != 1 || got[0].Duration != 250*time.Millisecond {
		t.Errorf("stored record = %+v", got[0])
	}
}
