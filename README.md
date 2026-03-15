# SaveYourTokens (`syt`)

A transparent CLI proxy for Claude Code that intercepts bash commands, compresses verbose output 30–60%, and tracks cumulative token savings in SQLite.

## How It Works

Once installed, a `PreToolUse` hook fires before every bash command Claude Code runs. The hook rewrites commands like `git log -10` to `syt git log -10`. `syt` runs the real command, strips noise from the output (passing compile lines, passing tests, progress bars), and returns only what matters — failures, errors, and summaries.

```
Claude Code bash call: "cargo test 2>&1"
  → hook rewrites to: "syt cargo test 2>&1"
    → syt runs real cargo test, captures output
      → strips: Compiling/Downloading/Updating lines
      → keeps: test failures, error messages, summary line
        → returns compressed output (~80-90% fewer tokens)
          → logs savings to SQLite in background
```

## Installation

### Step 1: Install prerequisites

**Go 1.22+** (required to build `syt`):

- **macOS:** `brew install go`
- **Linux (Ubuntu/Debian):** `sudo apt install golang-go` or download from [go.dev/dl](https://go.dev/dl/)
- **Windows:** Download the installer from [go.dev/dl](https://go.dev/dl/) and run it

Verify: `go version` should print `go1.22` or higher.

**Git** (to clone the repo):

- **macOS:** `brew install git` or install Xcode Command Line Tools: `xcode-select --install`
- **Linux:** `sudo apt install git`
- **Windows:** Download from [git-scm.com](https://git-scm.com/) or `winget install Git.Git`

**make** (optional, to use `make` shortcuts):

- **macOS:** included with Xcode Command Line Tools (`xcode-select --install`)
- **Linux:** `sudo apt install make`
- **Windows:** not required — use the Go commands directly (see below)

### Step 2: Build from source

```bash
git clone https://github.com/akshvaishnav21/save-your-tokens
cd save-your-tokens
```

**macOS / Linux:**
```bash
make build          # produces dist/syt
make install        # copies binary to $GOPATH/bin
```

**Windows (PowerShell):**
```powershell
mkdir dist
$env:CGO_ENABLED=0; go build -ldflags="-s -w" -o dist/syt.exe .
Copy-Item dist/syt.exe "$env:GOPATH\bin\syt.exe"
```

Make sure `$GOPATH/bin` is on your `PATH`.

**macOS / Linux** — add to `~/.bashrc` or `~/.zshrc`:
```bash
export PATH="$PATH:$(go env GOPATH)/bin"
```

**Windows (PowerShell)** — run once to set permanently:
```powershell
[Environment]::SetEnvironmentVariable("PATH", $env:PATH + ";$(go env GOPATH)\bin", "User")
```

### Step 3: Install the Claude Code hook

```bash
syt init
```

This patches `~/.claude/settings.json` to register `syt hook` as the `PreToolUse` hook. On macOS/Linux it also writes `~/.claude/hooks/syt-rewrite.sh` as a standalone script. Claude Code will now automatically route bash commands through `syt`.

> **Windows note:** The hook runs as `syt hook` (pure Go, no bash or `jq` required). Make sure `syt` is on the PATH that VS Code sees, then restart VS Code after running `syt init`.

### Uninstall

```bash
syt uninstall       # removes hook from ~/.claude/settings.json
```

## Usage

After `syt init`, everything is automatic. You can also invoke filters manually:

```bash
# Pipe output through a filter
git log -20 | syt git log
cargo test 2>&1 | syt cargo test
go test ./... -json | syt go test

# Test the rewrite engine
syt rewrite "git log -10"          # → syt git log -10
syt rewrite "npm install"          # → syt npm install
syt rewrite "go test ./... -json"  # → syt go test ./... -json

# View token savings dashboard
syt gain                    # summary (last 30 days)
syt gain --history          # recent command log
syt gain --daily            # day-by-day breakdown
syt gain --graph            # ASCII bar chart
syt gain --since 7          # last 7 days
syt gain --format json      # machine-readable output

# Discover commands not yet going through syt
syt discover
```

## Supported Commands

| Command | Token reduction |
|---------|----------------|
| `cargo test` | ~80–90% |
| `cargo build` / `cargo clippy` | ~55–75% |
| `go test` | ~70–80% |
| `go build` / `go vet` | ~60–80% |
| `pytest` | ~40–60% |
| `vitest` | ~95–99% |
| `jest` | ~80–90% |
| `playwright` / `npx playwright` | ~70–90% |
| `tsc` | ~80–85% |
| `eslint` / `biome` / `ruff` | ~75–85% |
| `mypy` / `prettier` | ~70–75% |
| `git log` / `git diff` / `git status` | ~75–80% |
| `npm install` / `pnpm install` | ~60–80% |
| `yarn install` / `yarn add` | ~70–80% |
| `bun install` / `bun test` | ~70–90% |
| `poetry install` / `poetry add` | ~70–80% |
| `pip install` / `uv pip install` | ~70–80% |
| `docker ps` / `docker build` | ~60–80% |
| `gh pr list` / `gh pr view` | ~60–70% |
| `next build` | ~85–90% |
| `prisma migrate` / `prisma generate` | ~70–80% |
| `nx test` / `nx build` | ~70–80% |
| `make test` / `make build` | ~60–75% |
| `grep` / `rg` / `ls` / `find` | ~60–70% |

## Configuration

Config file locations:
- Linux: `~/.config/syt/config.toml`
- macOS: `~/Library/Application Support/syt/config.toml`
- Windows: `%APPDATA%\syt\config.toml`

```toml
[tracking]
history_days = 90

[tee]
enabled = true
mode = "failures"    # "always" | "failures" | "never"
min_size = 500
max_files = 20
max_file_size = 1048576

[display]
colors = true
ultra_compact = false
```

When a command fails, `syt` saves the full raw output to `~/.local/share/syt/tee/` so you can inspect what was compressed away.

**Environment variable overrides:**

| Variable | Description |
|----------|-------------|
| `SYT_DB_PATH` | Override SQLite database path |
| `SYT_TEE` | Override tee mode (`always`/`failures`/`never`) |
| `SYT_HOOK_AUDIT=1` | Log every hook rewrite to `~/.local/share/syt/hook-audit.log` |

## Development

**macOS / Linux:**
```bash
make build        # compile binary
make test         # run unit tests
make lint         # run golangci-lint
make clean        # remove dist/
```

**Windows (PowerShell):**
```powershell
$env:CGO_ENABLED=0; go build -ldflags="-s -w" -o dist/syt.exe .
go test ./...
```

```bash
# Run a single test
go test ./cmd/... -run TestFilterCargoTest
go test ./internal/registry/... -v

# Run with race detector
go test ./... -count=1 -race
```

Fixture files for tests live in `tests/fixtures/`. Each filter has a `*_raw.txt` file containing real command output used to verify token savings thresholds.
