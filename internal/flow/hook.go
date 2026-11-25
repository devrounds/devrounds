package flow

import (
	"fmt"
	"strings"
	"time"
)

const bashHook = `# devrounds flow hook for bash. Remove these lines to stop recording.
__devrounds_hist_num() {
  builtin history 1 | sed -n 's/^ *\([0-9][0-9]*\).*/\1/p'
}
__devrounds_last_num=$(__devrounds_hist_num)
__devrounds_record() {
  local code=$?
  local entry num last
  entry=$(HISTTIMEFORMAT= builtin history 1)
  num=$(printf '%s\n' "$entry" | sed -n 's/^ *\([0-9][0-9]*\).*/\1/p')
  if [ -n "$num" ] && [ "$num" != "$__devrounds_last_num" ]; then
    __devrounds_last_num=$num
    last=$(printf '%s\n' "$entry" | sed 's/^ *[0-9][0-9]*[* ] *//')
    printf '%s' "$last" | devrounds flow record --dir "$PWD" --exit "$code" >/dev/null 2>&1
  fi
  return "$code"
}
case ";$PROMPT_COMMAND;" in
  *";__devrounds_record;"*) ;;
  *) PROMPT_COMMAND="__devrounds_record${PROMPT_COMMAND:+;$PROMPT_COMMAND}" ;;
esac
`

const zshHook = `# devrounds flow hook for zsh. Remove these lines to stop recording.
zmodload zsh/datetime 2>/dev/null
__devrounds_preexec() {
  __devrounds_cmd=$1
  __devrounds_t0=${EPOCHREALTIME:-0}
}
__devrounds_precmd() {
  local code=$?
  [[ -z $__devrounds_cmd ]] && return $code
  local -i ms=0
  if [[ -n $EPOCHREALTIME ]]; then
    ms=$(( (EPOCHREALTIME - __devrounds_t0) * 1000 ))
  fi
  print -rn -- "$__devrounds_cmd" | devrounds flow record --dir "$PWD" --exit "$code" --ms "$ms" >/dev/null 2>&1
  __devrounds_cmd=
  return $code
}
preexec_functions=(${preexec_functions:#__devrounds_preexec} __devrounds_preexec)
precmd_functions=(__devrounds_precmd ${precmd_functions:#__devrounds_precmd})
`

const powershellHook = `# devrounds flow hook for PowerShell. Remove these lines to stop recording.
if (-not (Get-Variable -Name DevroundsOriginalPrompt -Scope Global -ErrorAction SilentlyContinue)) {
    $global:DevroundsOriginalPrompt = $function:prompt
    $global:DevroundsLastId = (Get-History -Count 1 -ErrorAction SilentlyContinue).Id
    function global:prompt {
        $ok = $?
        $code = $global:LASTEXITCODE
        $h = Get-History -Count 1 -ErrorAction SilentlyContinue
        if ($h -and $h.Id -ne $global:DevroundsLastId) {
            $global:DevroundsLastId = $h.Id
            $ms = [int](($h.EndExecutionTime - $h.StartExecutionTime).TotalMilliseconds)
            $exitCode = if ($ok) { 0 } elseif ($code) { $code } else { 1 }
            try {
                $h.CommandLine | & devrounds flow record --dir (Get-Location).Path --exit $exitCode --ms $ms 2>$null | Out-Null
            } catch {}
        }
        & $global:DevroundsOriginalPrompt
    }
}
`

// Hook returns the shell snippet that records commands. devrounds only prints it;
// adding it to a shell profile is up to the user.
func Hook(shell string) (string, error) {
	switch strings.ToLower(shell) {
	case "bash":
		return bashHook, nil
	case "zsh":
		return zshHook, nil
	case "powershell", "pwsh":
		return powershellHook, nil
	}
	return "", fmt.Errorf("unknown shell %q (use bash, zsh or powershell)", shell)
}

// Capture records one command that a shell hook reported. Add removes secrets first.
func Capture(s *Store, command, dir string, exitCode int, ms int64) (bool, error) {
	return s.Add(Record{
		Time:     time.Now(),
		Command:  command,
		Dir:      dir,
		Project:  ProjectOf(dir),
		ExitCode: exitCode,
		Duration: time.Duration(ms) * time.Millisecond,
	})
}
