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

## Dependencies

**Runtime:**
- `jq` — required by the hook script (`brew install jq` / `apt install jq`)
- `syt` binary on `$PATH`

**Build:**
- Go 1.22+
- No CGO — pure Go build (`modernc.org/sqlite` for SQLite)

## Installation

### Option 1: Build from source

```bash
git clone https://github.com/yourname/save-your-tokens
cd save-your-tokens
make build          # produces dist/syt (or dist/syt.exe on Windows)
make install        # copies to $GOPATH/bin
```

### Option 2: `go install`

```bash
go install github.com/saveyourtokens/syt@latest
```

### Option 3: Download binary

Grab the binary for your platform from the [releases page](https://github.com/yourname/save-your-tokens/releases), put it on your `$PATH`.

### Install the Claude Code hook

```bash
syt init
```

This writes `~/.claude/hooks/syt-rewrite.sh` and patches `~/.claude/settings.json` to register the `PreToolUse` hook. Claude Code will now automatically route bash commands through `syt`.

### Uninstall

```bash
syt uninstall
```

Removes the hook entry from `~/.claude/settings.json`.

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
| `cargo build` | ~55–65% |
| `go test -json` | ~70–80% |
| `pytest` | ~40–60% |
| `vitest` | ~95–99% |
| `playwright` | ~70–90% |
| `tsc` | ~80–85% |
| `eslint` / `biome` | ~75–85% |
| `git log` / `git diff` | ~75–80% |
| `npm install` / `pnpm` | ~60–75% |
| `docker ps` / `docker build` | ~60–70% |
| `gh pr list` / `gh pr view` | pass-through + trim |
| `next build` | ~85–90% |
| `prisma migrate` | ~80% |

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

```bash
make build        # compile binary
make test         # run unit tests
make lint         # run golangci-lint
make clean        # remove dist/

# Run a single test
go test ./cmd/... -run TestFilterCargoTest
go test ./internal/registry/... -v

# Run with race detector
go test ./... -count=1 -race
```

Fixture files for tests live in `tests/fixtures/`. Each filter has a `*_raw.txt` file containing real command output used to verify token savings thresholds.
