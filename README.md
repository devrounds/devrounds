# devrounds

One local CLI that does a "round" of checks on any repo or dev machine and gives a single verdict on whether a project is ready to run.

> Early development (v0.2). `round`, `check`, `env`, `map` and `scan` work today. `story` and `flow` are planned.

## Install

Pick one:

    npm install -g devrounds

    pip install devrounds

The npm and pip packages are thin wrappers. They download the prebuilt release binary for your platform and verify its checksum.

Or download a binary for Linux, macOS or Windows from the [Releases page](https://github.com/devrounds/devrounds/releases). Or, with Go installed:

    go install github.com/devrounds/devrounds@latest

## Usage

    devrounds              # same as: devrounds round
    devrounds round [path] # check + env in one pass, one verdict
    devrounds check [path] # readiness checks
    devrounds env [path]   # environment-variable drift
    devrounds map          # what is listening on which port
    devrounds scan [path]  # takeover report for an unfamiliar repo

Example:

    $ devrounds check
    Check on shop-api

    OK    Runtime   Node 20.11.0 matches .nvmrc
    OK    Binaries  found: git, node, npm
    WARN  Ports     3000 in use (node.exe, pid 4121), expected by Node default
    FAIL  .env      .env not found, 3 missing: DATABASE_URL, JWT_SECRET, REDIS_URL

    Result: NOT READY (2 issues)
    Next: copy .env.example .env

The exit code is 1 when anything fails and 0 otherwise, so it works in CI.

## What it checks

`check`
- Pinned runtime version (`.nvmrc`, `.node-version`, `.python-version`) against the installed one
- Required binaries (git, plus docker, node, npm or python when the project needs them)
- Ports the project expects (from Compose files, `PORT`-style names in `.env.example`, or the Node/Python default) and which process holds them
- `.env` against `.env.example`: missing or empty names

`env`
- Names used in code (`process.env.X`, `os.environ["X"]`, `os.getenv("X")`) but missing from `.env.example` and Docker/Compose files
- Names in `.env.example` that no code uses
- Names referenced in GitHub Actions workflows but missing locally

`map`
- Every listening TCP port with its process name, PID and the folder the process was started from (`--json` supported)

`scan`
- Project type, pinned runtime, entry points and likely run commands. Other kinds of project fall back to commands found in Docker, Compose and Makefile files, CI config and the README
- Gaps: missing README, LICENSE, pinned runtime, env documentation or CI
- Risks: secret-looking files that are committed (by file name only, contents are never read) and debug flags left on
- History: commit count, last activity, top contributors (names only, never emails) and the folders changed most in the last 90 days
- `--json` supported

## Rules

- Never prints secret values. Names and file locations only.
- Fully local. No accounts, no network calls.
- Read-only. It never changes your machine.

## Limits

- Node and Python projects only for now.
- Environment variables are found by pattern matching, so dynamically built names are missed, and so are variables read by tools outside the scanned `.js`, `.ts` and `.py` files.
- Compose ports that use variables or ranges are skipped.
- `scan` does not report dependency risks yet.
- Secret detection is by file name only, so a secret inside an ordinarily named file is not found.
- Process names and folders for system services and other users' processes may be hidden (shown as `-`), especially on Windows and macOS.

## License

MIT




