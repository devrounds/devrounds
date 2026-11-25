package flow

import (
	"path/filepath"
	"regexp"
	"strings"
)

const redacted = "<redacted>"

const secretWords = `(?:key|token|secret|passw(?:or)?d|pwd|credential|auth)`

var (
	// https://user:password@host
	urlCredsRe = regexp.MustCompile(`(?i)([a-z][a-z0-9+.-]*://[^/\s:@]+:)([^@/\s]+)(@)`)
	// Authorization: Bearer ..., X-Api-Key: ..., Cookie: ...
	headerRe = regexp.MustCompile(`(?i)((?:proxy-authorization|authorization|x-api-key|api-key|x-auth-token|set-cookie|cookie)\s*:\s*)([^"'\r\n]+)`)
	// $env:API_KEY = "value"
	psEnvRe = regexp.MustCompile(`(?i)(\$env:[A-Za-z0-9_]*` + secretWords + `[A-Za-z0-9_]*\s*=\s*)("[^"]*"|'[^']*'|\S+)`)
	// API_KEY=value, DB_PASSWORD="value", Authorization="Bearer x"
	assignRe = regexp.MustCompile(`(?i)\b([A-Za-z0-9_]*` + secretWords + `[A-Za-z0-9_]*)=("[^"]*"|'[^']*'|\S+)`)
	// --password value, --token=value, -Password value
	flagRe = regexp.MustCompile(`(?i)(^|\s)(--?(?:password|passwd|pwd|pass|token|secret|secret[-_]?key|api[-_]?key|access[-_]?key|auth|authorization|credentials?|bearer|key))([=:\s]+)("[^"]*"|'[^']*'|\S+)`)
	// Well-known token formats, wherever they appear.
	knownTokenRe = regexp.MustCompile(`gh[pousr]_[A-Za-z0-9]{20,}|github_pat_[A-Za-z0-9_]{20,}|npm_[A-Za-z0-9]{20,}|pypi-[A-Za-z0-9_-]{20,}|AKIA[0-9A-Z]{16}|xox[abprs]-[A-Za-z0-9-]{10,}|sk_(?:live|test)_[A-Za-z0-9]{10,}|sk-[A-Za-z0-9_-]{20,}|eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`)
	longWordRe   = regexp.MustCompile(`[A-Za-z0-9+/=_-]{32,}`)
)

func hasUpper(s string) bool { return strings.ToLower(s) != s }
func hasLower(s string) bool { return strings.ToUpper(s) != s }
func hasDigit(s string) bool { return strings.ContainsAny(s, "0123456789") }

// looksLikeToken is true for long words that mix upper case, lower case and digits.
// Pure hex strings, such as git hashes, are left alone.
func looksLikeToken(s string) bool {
	return hasUpper(s) && hasLower(s) && hasDigit(s)
}

func redact(cmd string) string {
	cmd = urlCredsRe.ReplaceAllString(cmd, "${1}"+redacted+"${3}")
	cmd = headerRe.ReplaceAllString(cmd, "${1}"+redacted)
	cmd = psEnvRe.ReplaceAllString(cmd, "${1}"+redacted)
	cmd = assignRe.ReplaceAllString(cmd, "${1}="+redacted)
	cmd = flagRe.ReplaceAllString(cmd, "${1}${2}${3}"+redacted)
	cmd = knownTokenRe.ReplaceAllString(cmd, redacted)
	cmd = longWordRe.ReplaceAllStringFunc(cmd, func(w string) string {
		if looksLikeToken(w) {
			return redacted
		}
		return w
	})
	return cmd
}

func isFlowCommand(cmd string) bool {
	f := strings.Fields(cmd)
	if len(f) < 2 {
		return false
	}
	name := strings.ToLower(strings.TrimSuffix(filepath.Base(f[0]), ".exe"))
	return name == "devrounds" && strings.EqualFold(f[1], "flow")
}

// Sanitize decides whether a command may be recorded and returns it with secrets removed.
// Empty commands, commands that start with a space, and devrounds flow commands
// are never recorded.
func Sanitize(raw string) (string, bool) {
	if raw == "" || raw[0] == ' ' || raw[0] == '\t' {
		return "", false
	}
	cmd := strings.TrimSpace(raw)
	if cmd == "" || isFlowCommand(cmd) {
		return "", false
	}
	return redact(cmd), true
}
