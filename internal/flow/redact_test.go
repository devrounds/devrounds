package flow

import (
	"strings"
	"testing"
)

func TestRedact(t *testing.T) {
	cases := []struct{ in, want string }{
		// ordinary commands pass through
		{"git push origin main", "git push origin main"},
		{"npm run build", "npm run build"},
		{`copy C:\keys\a.txt .`, `copy C:\keys\a.txt .`},
		{"git config user.name=Ann", "git config user.name=Ann"},
		{"ssh -i key.pem host", "ssh -i key.pem host"},
		{"mycli --password-file ~/pw.txt", "mycli --password-file ~/pw.txt"},
		{"git checkout 0123456789abcdef0123456789abcdef01234567", "git checkout 0123456789abcdef0123456789abcdef01234567"},

		// flags with values
		{"mysql --password=hunter2 -u root", "mysql --password=<redacted> -u root"},
		{"mycli --token abc123 run", "mycli --token <redacted> run"},
		{"mycli --PASSWORD=x", "mycli --PASSWORD=<redacted>"},
		{"echo my-key value", "echo my-key value"},
		{"git log --key value", "git log --key <redacted>"},
		{`echo "--token abc123"`, `echo "--token <redacted>`},
		{"Connect-Thing -Password hunter2", "Connect-Thing -Password <redacted>"},
		{`tool --api-key "my key" go`, `tool --api-key <redacted> go`},

		// assignments
		{"API_KEY=abc123 npm start", "API_KEY=<redacted> npm start"},
		{`export DB_PASSWORD="my pass"`, "export DB_PASSWORD=<redacted>"},
		{"docker run -e TOKEN=abc img", "docker run -e TOKEN=<redacted> img"},
		{`$env:GITHUB_TOKEN = "abc"`, `$env:GITHUB_TOKEN = <redacted>`},
		{`$env:MY_SECRET="abc"`, `$env:MY_SECRET=<redacted>`},
		{`Invoke-RestMethod -Headers @{Authorization="Bearer x"} http://y`, `Invoke-RestMethod -Headers @{Authorization=<redacted>} http://y`},

		// headers and URLs
		{`curl -H "Authorization: Bearer abc.def" https://x`, `curl -H "Authorization: <redacted>" https://x`},
		{`curl -H 'X-Api-Key: abc' https://x`, `curl -H 'X-Api-Key: <redacted>' https://x`},
		{"git clone https://user:secret@github.com/a/b", "git clone https://user:<redacted>@github.com/a/b"},

		// token formats
		{"echo ghp_abcdefghijklmnopqrstuvwxyz0123456789", "echo <redacted>"},
		{"aws configure set key AKIAIOSFODNN7EXAMPLE", "aws configure set key <redacted>"},
		{"deploy Zk3Qw9XyLm2Pa8Vb7Nc4Rd1Sf6Tg5Uh0Ji", "deploy <redacted>"},
	}
	for _, c := range cases {
		if got := redact(c.in); got != c.want {
			t.Errorf("redact(%q)\n got  %q\n want %q", c.in, got, c.want)
		}
	}
}

func TestSanitizeSkips(t *testing.T) {
	for _, raw := range []string{"", "   ", " echo hi", "\techo hi", "devrounds flow suggest", "./devrounds flow save x", `C:\tools\devrounds.exe FLOW run x`} {
		if got, ok := Sanitize(raw); ok {
			t.Errorf("Sanitize(%q) should not be recorded, got %q", raw, got)
		}
	}
}

func TestSanitizeKeepsCommands(t *testing.T) {
	got, ok := Sanitize("  git status\n"[2:])
	if !ok || got != "git status" {
		t.Errorf("got %q, %v", got, ok)
	}
	got, ok = Sanitize("devrounds check")
	if !ok || got != "devrounds check" {
		t.Errorf("other devrounds commands are recorded: got %q, %v", got, ok)
	}
	got, ok = Sanitize("curl --token abc123 https://x\r\n")
	if !ok || strings.Contains(got, "abc123") || !strings.Contains(got, "<redacted>") {
		t.Errorf("secret not removed: %q, %v", got, ok)
	}
}
