# SaveYourTokens — Implementation Plan

**Product**: SaveYourTokens (`syt`)
**Language**: Go
**Date**: 2026-03-12

This document describes the step-by-step implementation of SaveYourTokens. Follow phases in order. Each phase ends with a working, testable state.

---

## Pre-Flight Checklist

Before starting, verify your environment:

```bash
go version          # Must be ≥ 1.22
jq --version        # Required by hook script
git --version       # For testing git filters
```

Working directory for all commands: the `syt/` project root.

---

## Phase 1: Project Scaffold

**Goal**: A compilable Go project with CLI skeleton and shared utilities.

### 1.1 Initialize Module

```bash
mkdir syt && cd syt
go mod init github.com/saveyourtokens/syt
```

### 1.2 Add Dependencies

```bash
go get github.com/spf13/cobra@v1.8.0
go get github.com/BurntSushi/toml@v1.3.2
go get modernc.org/sqlite@v1.29.0
go get github.com/stretchr/testify@v1.9.0
go get github.com/fatih/color@v1.16.0
```

### 1.3 Create `main.go`

```go
package main

import (
    "os"
    "github.com/saveyourtokens/syt/cmd"
)

func main() {
    if err := cmd.Execute(); err != nil {
        os.Exit(1)
    }
}
```

### 1.4 Create `cmd/root.go`

Cobra root command with global flags:
- `-v, --verbose` count flag (0-3 verbosity levels)
- `--ultra-compact` bool flag (icon-only output)
- Version subcommand (`syt version` prints semver)

### 1.5 Create `internal/utils/utils.go`

Implement:
- `StripANSI(s string) string` — remove `\x1b[...m` escape sequences
- `Truncate(s string, max int) string` — truncate with "..." suffix
- `CountTokens(s string) int` — whitespace-split word count
- `FormatTokens(n int) string` — "1.2M", "59.2K", "694"
- `FormatSavingsPct(pct float64) string` — "87.3%"
- `HomeDir() string` — cached `os.UserHomeDir()`
- `DataDir() string` — `~/.local/share/syt` (Linux), `~/Library/Application Support/syt` (macOS), `%APPDATA%\syt` (Windows)
- `ConfigDir() string` — `~/.config/syt` (Linux), `~/Library/Application Support/syt` (macOS), `%APPDATA%\syt` (Windows)

### 1.6 Create Makefile

```makefile
.PHONY: build test install lint clean

build:
	go build -ldflags="-s -w" -o bin/syt .

test:
	go test ./...

install:
	go install .

lint:
	go vet ./...
	gofmt -l .

clean:
	rm -rf bin/
```

### 1.7 Verification

```bash
make build
./bin/syt --help          # Should show usage
./bin/syt version         # Should print version string
go test ./...             # Should pass (no tests yet, but should compile)
```

---

## Phase 2: Core Filter Infrastructure

**Goal**: The `Filter` interface and fallback-safe execution wrapper that all command modules will use.

### 2.1 Create `internal/filter/filter.go`

Define the execution runner:

```go
type Runner struct {
    Verbose int
}

// RunWithFilter executes `binary` with `args`, applies filterFn to captured output,
// prints the result, and returns the command's exit code.
// If filterFn panics or errors, falls back to raw output.
// If exitCode != 0 and rawOutput >= 500 bytes, calls onFailure(rawOutput).
func (r *Runner) RunWithFilter(
    binary string,
    args []string,
    filterFn func(stdout, stderr string) string,
    onSuccess func(filtered, raw string, inputTok, outputTok int),
    onFailure func(raw string),
) (exitCode int, err error)
```

Key implementation details:
- Use `exec.Command(binary, args...)` — never `exec.Command("bash", "-c", ...)`
- Capture stdout and stderr separately with `bytes.Buffer`
- Wrap `filterFn` in a `recover()` panic handler
- Return the process exit code via `cmd.ProcessState.ExitCode()`

### 2.2 Create `internal/filter/language.go`

Language detection by file extension for the `read` and `smart` commands:

```go
type Language string
type FilterLevel int

const (
    LevelNone      FilterLevel = 0
    LevelMinimal   FilterLevel = 1
    LevelAggressive FilterLevel = 2
)

func DetectLanguage(filename string) Language
func StripComments(code string, lang Language, level FilterLevel) string
```

Support: Go, Python, JavaScript, TypeScript, Rust, Java, C, C++, Ruby, Shell.

### 2.3 Verification

```bash
go build ./...    # Must compile
go test ./internal/filter/...
```

---

## Phase 3: First High-Value Filters

**Goal**: The two highest-ROI filters: git and cargo. These alone cover the majority of Claude Code token waste.

### 3.1 Create `cmd/git.go`

Implement `syt git` with subcommands:

**`syt git log [args...]`** — Target: 80% savings
- Capture `git log --oneline --stat [args]`
- Output format: one line per commit: `{short-hash} {message}` + aggregate stats line
- Example output:
  ```
  abc1234 fix: correct auth token refresh logic
  def5678 feat: add user preferences endpoint
  ghi9012 chore: update dependencies
  10 commits · +342 -189 in 8 files
  ```

**`syt git diff [args...]`** — Target: 75% savings
- Capture `git diff --stat [args]`
- Show: changed file list with +/- counts, summary line
- Suppress: actual diff hunks unless `-v` flag

**`syt git status [args...]`** — Target: 80% savings
- Show: counts by category (staged: 3, modified: 2, untracked: 1)
- Show: branch name and upstream sync status

**`syt git push/pull/fetch/add/commit [args...]`** — Target: 90% savings
- Suppress: progress bars, counting objects, remote counts
- Show: one-line confirmation ("ok ✓ pushed to origin/main")
- On error: show full output

**`syt git branch/stash/worktree [args...]`**
- Compact list format

Key implementation requirements:
- Pass ALL user-supplied args through to git unchanged
- Handle global git options: `-C <dir>`, `-c key=val`, `--git-dir`, `--work-tree`
- Preserve exit codes exactly
- Auto-detect `--merges` in args to avoid injecting `--no-merges` when caller specified it

### 3.2 Create `cmd/cargo.go`

Implement `syt cargo` with subcommands:

**`syt cargo test [args...]`** — Target: 90% savings
- Parse test output: detect `test X ... FAILED` lines
- Show only: failed tests with their output blocks
- Show summary: `42 passed, 2 failed, 0 ignored`
- On all-pass: `42 passed ✓`

**`syt cargo build [args...]`** — Target: 80% savings
- Show only: error and warning lines (not "Compiling..." lines)
- Show: final "Finished" line
- On success: `built release in 12.3s ✓`

**`syt cargo clippy [args...]`** — Target: 70% savings
- Group warnings by lint rule
- Show: `{count} {rule}: {files}`
- On no warnings: `no issues ✓`

**`syt cargo check [args...]`** — Similar to build filter.

### 3.3 Create Fixtures

```bash
# Capture real command output for tests
git log -20 > tests/fixtures/git_log_raw.txt
git diff HEAD~5 --stat > tests/fixtures/git_diff_raw.txt
git status > tests/fixtures/git_status_raw.txt
cargo test 2>&1 > tests/fixtures/cargo_test_raw.txt
cargo test 2>&1 > tests/fixtures/cargo_test_failing_raw.txt  # after introducing a test failure
```

### 3.4 Write Tests

For each filter function:

```go
// cmd/git_test.go
func TestFilterGitLog(t *testing.T) {
    input, _ := os.ReadFile("../tests/fixtures/git_log_raw.txt")
    output := filterGitLog(string(input), "")

    // Token savings
    inputTok := utils.CountTokens(string(input))
    outputTok := utils.CountTokens(output)
    savings := 100.0 - float64(outputTok)/float64(inputTok)*100.0
    assert.GreaterOrEqual(t, savings, 80.0, "git log should save ≥80%% tokens")

    // Golden file
    golden := loadGolden(t, "git_log_filtered.txt")
    assert.Equal(t, golden, output)
}
```

### 3.5 Verification

```bash
go test ./cmd/... -v
# Manually verify:
./bin/syt git log -10       # Should show compact format
./bin/syt git status        # Should show file counts
./bin/syt cargo test        # Should show failures-only (run against a project with tests)
```

---

## Phase 4: Token Tracking

**Goal**: Persistent SQLite token metrics and the `syt gain` dashboard.

### 4.1 Create `internal/tracker/tracker.go`

Implement:

```go
type Record struct {
    OriginalCmd  string
    SytCmd       string
    ProjectPath  string
    InputTokens  int
    OutputTokens int
    ExecutionMs  int64
}

type Tracker struct {
    db *sql.DB
}

func NewTracker(dbPath string) (*Tracker, error)
func (t *Tracker) Track(r Record) error
func (t *Tracker) GetSummary(since time.Time) (Summary, error)
func (t *Tracker) GetHistory(limit int) ([]HistoryEntry, error)
func (t *Tracker) GetDailyStats(days int) ([]DayStat, error)
func (t *Tracker) Cleanup(retentionDays int) error
func (t *Tracker) Close() error
```

Schema setup in `NewTracker` — create tables if not exist, set `PRAGMA journal_mode=WAL`.

Automatic cleanup: call `Cleanup(90)` during `NewTracker` if the database is more than 1 day old (check a `last_cleanup` key in a `syt_meta` table).

### 4.2 Update Filter Runner

Add tracking to `internal/filter/filter.go`:

```go
type Runner struct {
    Verbose int
    Tracker *tracker.Tracker  // may be nil (disabled)
}
```

In `RunWithFilter`, after filtering, record:
```go
if r.Tracker != nil {
    go r.Tracker.Track(tracker.Record{
        OriginalCmd:  originalCmd,
        SytCmd:       "syt " + binary + " " + strings.Join(args, " "),
        ProjectPath:  mustGetwd(),
        InputTokens:  inputTok,
        OutputTokens: outputTok,
        ExecutionMs:  elapsed.Milliseconds(),
    })
}
```

### 4.3 Create `cmd/gain.go`

Implement `syt gain`:

```
SaveYourTokens Analytics
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Period:     Last 30 days
Commands:   1,247 tracked
Saved:      89,340 tokens
Avg savings: 84.2%

Top commands by tokens saved:
  cargo test    │  412 runs │ 42,830 tokens saved │ 91.3%
  git log       │  287 runs │ 18,655 tokens saved │ 80.1%
  pnpm list     │  156 runs │  9,048 tokens saved │ 76.4%
  vitest        │   89 runs │  8,255 tokens saved │ 99.2%
  tsc --noEmit  │   44 runs │  3,872 tokens saved │ 83.1%
```

Flags: `--history`, `--daily`, `--graph`, `--since <days>`, `--format json`

### 4.4 Verification

```bash
# Run a few syt commands to generate tracking data
./bin/syt git log -5
./bin/syt git status

# Check analytics
./bin/syt gain
./bin/syt gain --history
./bin/syt gain --format json | jq .

# Verify SQLite directly
sqlite3 ~/.local/share/syt/tracking.db "SELECT * FROM syt_log LIMIT 5;"
```

---

## Phase 5: Claude Code Hook System

**Goal**: `syt init` installs the hook and Claude Code automatically rewrites all matching commands.

### 5.1 Create Hook Script Constant

In `cmd/init_cmd.go`, embed the hook script as a string constant:

```go
const hookScript = `#!/usr/bin/env bash
# SaveYourTokens - Claude Code command rewrite hook
# ... (full script content)
`
```

The script content is fixed — it never changes based on user config. When the script needs updating, users run `syt init` again.

### 5.2 Create `cmd/init_cmd.go`

Implement `syt init`:

1. **Resolve paths**:
   - Hook dir: `~/.claude/hooks/`
   - Hook path: `~/.claude/hooks/syt-rewrite.sh`
   - Settings path: `~/.claude/settings.json`

2. **Write hook file**:
   - Create `~/.claude/hooks/` if missing
   - Write hook script content
   - Set permissions 0755 (Unix) or equivalent (Windows)
   - Skip if content unchanged (idempotent)

3. **Store integrity hash** (via `internal/integrity`):
   - Compute SHA-256 of written script
   - Store in `~/.local/share/syt/hook-integrity.json`

4. **Patch settings.json**:
   - Read existing `~/.claude/settings.json` (create `{}` if missing)
   - Parse JSON
   - Check if hook entry already present (idempotent)
   - Add hook entry under `hooks.PreToolUse`
   - Write back atomically (write to tempfile, rename)
   - Prompt user before patching (unless `--auto-patch`)

5. **Create default config**:
   - Write `~/.config/syt/config.toml` with defaults if not exists

6. **Print success summary**:
   ```
   SaveYourTokens initialized ✓

   Hook installed: ~/.claude/hooks/syt-rewrite.sh
   Settings patched: ~/.claude/settings.json
   Config: ~/.config/syt/config.toml

   All matching Claude Code commands will now be automatically optimized.
   Run 'syt gain' after a few sessions to see your savings.
   ```

### 5.3 Create `cmd/uninstall.go`

Implement `syt uninstall`:
- Remove `~/.claude/hooks/syt-rewrite.sh`
- Remove hook entry from `~/.claude/settings.json`
- Remove integrity hash file
- Leave tracking database and config untouched

### 5.4 Create `cmd/rewrite.go`

Implement `syt rewrite "<command>"`:

```go
// Takes the raw command string as a single argument.
// Prints the syt equivalent and exits 0 if a rewrite exists.
// Prints nothing and exits 1 if no rewrite exists.
```

This is the internal command called by the hook script. It must be fast (<2ms) and produce clean output (no debug info, no newline except the rewritten command).

### 5.5 Create `internal/integrity/integrity.go`

```go
func StoreHash(hookPath, dataDir string) error
func VerifyHash(hookPath, dataDir string) (bool, error)
```

Hash file format (`hook-integrity.json`):
```json
{
  "hook_path": "/Users/user/.claude/hooks/syt-rewrite.sh",
  "sha256": "abc123...",
  "installed_at": "2026-03-12T10:00:00Z",
  "syt_version": "1.0.0"
}
```

### 5.6 Verification

```bash
./bin/syt init --auto-patch

# Verify files created
ls -la ~/.claude/hooks/syt-rewrite.sh
cat ~/.claude/settings.json | jq '.hooks'

# Test the rewrite command
./bin/syt rewrite "git log -10"      # Should print: syt git log -10
./bin/syt rewrite "terraform plan"   # Should print nothing, exit 1
echo $?                               # Should print: 1

# Test hook script directly
echo '{"tool_name":"Bash","tool_input":{"command":"git log -10"}}' \
  | ~/.claude/hooks/syt-rewrite.sh
# Should output hook JSON with "syt git log -10"
```

---

## Phase 6: Rewrite Registry

**Goal**: The authoritative mapping of 60+ commands, used by hook and discover.

### 6.1 Create `internal/registry/registry.go`

Structure:
1. `init()` function that populates `rules []Rule` with all patterns
2. `RewriteCommand(cmd string, excluded []string) string`
3. `ClassifyCommand(cmd string) Classification`
4. `IsIgnored(cmd string) bool`

### 6.2 Implement All Rules

Organize rules into groups. Each group targets one tool category:

**Git (8 rules)**:
- `git log`, `git diff`, `git status`, `git push`, `git pull`, `git fetch`, `git branch`, `git add`, `git commit`, `git stash`, `git worktree`

**Go (4 rules)**:
- `go test`, `go build`, `go vet`, `go run`

**Cargo (5 rules)**:
- `cargo test`, `cargo build`, `cargo clippy`, `cargo check`, `cargo run`

**Package managers (10 rules)**:
- `pnpm list`, `pnpm outdated`, `pnpm install`, `pnpm add`, `pnpm remove`
- `npm install`, `npm run`, `npm test`, `npm list`
- `pip install`, `pip list`, `pip outdated`
- `uv pip install`, `uv pip list`

**TypeScript / JavaScript (8 rules)**:
- `tsc`, `tsc --noEmit`
- `eslint`, `biome check`
- `vitest`, `vitest run`
- `next build`, `next dev`
- `prisma migrate`, `prisma generate`, `prisma studio`
- `prettier --check`

**Test runners (4 rules)**:
- `pytest`, `py.test`
- `playwright test`

**Linters (4 rules)**:
- `ruff check`, `ruff format`
- `mypy`

**GitHub CLI (6 rules)**:
- `gh pr view`, `gh pr list`, `gh pr create`
- `gh issue view`, `gh issue list`
- `gh run view`, `gh run list`

**File operations (6 rules)**:
- `ls`, `ls -la`, `find`
- `grep`, `rg` (ripgrep)
- `cat`, `bat`

**Containers (4 rules)**:
- `docker ps`, `docker build`, `docker logs`
- `docker compose up`, `docker compose logs`

**Ignored prefixes** (never rewrite):
- Lines starting with `#` (comments)
- `cd`, `pwd`, `echo`, `export`, `source`, `.`
- `syt` (already rewritten)
- `curl`, `wget` (user may want raw output by default — configurable)

### 6.3 Write Registry Tests

```go
func TestRewriteCommand(t *testing.T) {
    cases := []struct {
        input    string
        expected string
    }{
        {"git log -10", "syt git log -10"},
        {"cargo test --release", "syt cargo test --release"},
        {"pnpm list --depth 3", "syt pnpm list --depth 3"},
        {"terraform plan", ""},                // no rewrite
        {"syt git status", ""},                // already syt
        {"# comment", ""},                     // ignored
    }
    for _, c := range cases {
        got := registry.RewriteCommand(c.input, nil)
        assert.Equal(t, c.expected, got, "input: %s", c.input)
    }
}
```

### 6.4 Verification

```bash
go test ./internal/registry/... -v
# All rules should have test coverage
# Zero false positives: commands that should not be rewritten must not be
```

---

## Phase 7: Tee Output Recovery

**Goal**: Save full output on failure so Claude can read it without re-running commands.

### 7.1 Create `internal/tee/tee.go`

```go
type Config struct {
    Enabled     bool
    Mode        string  // "failures", "always", "never"
    MinSize     int     // bytes, default 500
    MaxFiles    int     // default 20
    MaxFileSize int64   // bytes, default 1MB
    Directory   string  // default: DataDir()/tee/
}

type Tee struct {
    config Config
}

func New(cfg Config) *Tee

// Save saves raw output to a tee file.
// Returns the file path if saved, "" if not.
func (t *Tee) Save(raw string, cmdSlug string, exitCode int) string

// Hint returns the formatted hint string for a saved tee file.
func (t *Tee) Hint(path string) string

// rotate removes oldest files if count exceeds MaxFiles
func (t *Tee) rotate() error
```

### 7.2 Integrate into Filter Runner

In `internal/filter/filter.go`, after printing filtered output:

```go
if exitCode != 0 && t.Tee != nil {
    if path := t.Tee.Save(rawOutput, cmdSlug, exitCode); path != "" {
        fmt.Println(t.Tee.Hint(path))
    }
}
```

### 7.3 Test Tee Behavior

```go
func TestTeeSavesOnFailure(t *testing.T) {
    teeDir := t.TempDir()
    tee := tee.New(tee.Config{
        Enabled:  true,
        Mode:     "failures",
        MinSize:  10,
        MaxFiles: 5,
        Directory: teeDir,
    })

    // Failure: should save
    path := tee.Save("large output here...", "cargo_test", 1)
    assert.NotEmpty(t, path)
    assert.FileExists(t, path)

    // Success: should not save
    path = tee.Save("output", "git_log", 0)
    assert.Empty(t, path)
}
```

### 7.4 Verification

```bash
# Trigger a failing command through syt
./bin/syt cargo test -- --test nonexistent_test 2>&1
# Should show: [full output: ~/.local/share/syt/tee/...]

ls -la ~/.local/share/syt/tee/
# Should show the saved file
```

---

## Phase 8: Configuration

**Goal**: `~/.config/syt/config.toml` with env var overrides, loaded at startup.

### 8.1 Create `internal/config/config.go`

```go
type TrackingConfig struct {
    DatabasePath string `toml:"database_path"`
    HistoryDays  int    `toml:"history_days"`
}

type HooksConfig struct {
    ExcludeCommands []string `toml:"exclude_commands"`
}

type TeeConfig struct {
    Enabled     bool   `toml:"enabled"`
    Mode        string `toml:"mode"`
    MinSize     int    `toml:"min_size"`
    MaxFiles    int    `toml:"max_files"`
    MaxFileSize int64  `toml:"max_file_size"`
    Directory   string `toml:"directory"`
}

type DisplayConfig struct {
    Colors       bool `toml:"colors"`
    UltraCompact bool `toml:"ultra_compact"`
}

type Config struct {
    Tracking TrackingConfig `toml:"tracking"`
    Hooks    HooksConfig    `toml:"hooks"`
    Tee      TeeConfig      `toml:"tee"`
    Display  DisplayConfig  `toml:"display"`
}

func Load() Config
func Defaults() Config
func WriteDefaults(path string) error
```

### 8.2 Apply Env Var Overrides

In `Load()`, after reading the TOML file, override with env vars:

```go
if v := os.Getenv("SYT_DB_PATH"); v != "" {
    cfg.Tracking.DatabasePath = v
}
if v := os.Getenv("SYT_TEE"); v == "0" {
    cfg.Tee.Enabled = false
}
if v := os.Getenv("SYT_TEE_DIR"); v != "" {
    cfg.Tee.Directory = v
}
// etc.
```

### 8.3 Pass Config Through Root Command

In `cmd/root.go`, load config once and store in cobra context or package-level var. Pass to filter runners, tracker, and tee constructors.

### 8.4 Create `cmd/config_cmd.go`

Implement `syt config show` — prints the effective config (after env var overrides) as TOML.

### 8.5 Verification

```bash
./bin/syt config show
# Should show default config values

SYT_DB_PATH=/tmp/test.db ./bin/syt config show
# Should show updated database_path

echo 'exclude_commands = ["curl"]' >> ~/.config/syt/config.toml
./bin/syt rewrite "curl https://example.com"
# Should exit 1 (no rewrite — excluded)
```

---

## Phase 9: Discover Command

**Goal**: Analyze Claude Code session history to find missed savings opportunities.

### 9.1 Create `internal/discover/provider.go`

```go
type BashEntry struct {
    Command   string
    OutputLen int
    Timestamp time.Time
}

type SessionProvider interface {
    GetEntries(since time.Time, projectPath string) ([]BashEntry, error)
}

type ClaudeCodeProvider struct {
    HomeDir string
}

func (p *ClaudeCodeProvider) GetEntries(since time.Time, project string) ([]BashEntry, error)
```

Implementation details:
- Scan `~/.claude/projects/` for directories
- Filter by project: encode current dir path to Claude Code format (`/path/to/proj` → `-path-to-proj`)
- For each matching directory, read all `.jsonl` files
- Parse each line as JSON, extract `tool_input.command` from `Bash` tool_use entries
- Filter by timestamp (since) from the session metadata

### 9.2 Create `internal/discover/discover.go`

```go
type Result struct {
    Supported   []SupportedEntry
    Unsupported []UnsupportedEntry
    TotalCmds   int
    AlreadySyt  int
}

type SupportedEntry struct {
    Command    string
    SytCmd     string
    Count      int
    Category   registry.Category
    SavesPct   int
    EstSavings int  // estimated tokens saved
}

func Run(provider SessionProvider, since time.Time, project string) (Result, error)
```

### 9.3 Create `internal/discover/report.go`

Format the `Result` as either:
- Human-readable text (default)
- JSON (`--format json`)

### 9.4 Create `cmd/discover.go`

```
syt discover [flags]
  --since <days>    Look back N days (default: 30)
  --project <path>  Filter to specific project (default: current dir)
  --format json     Machine-readable output
  --all             Include all projects, not just current
```

### 9.5 Also Add: `syt init --claude-md`

In `cmd/init_cmd.go`, add `--claude-md` flag:
- Writes a CLAUDE.md-style instruction block to `~/.claude/CLAUDE.md`
- The block tells Claude to use `syt <cmd>` instead of raw commands
- Covers all supported commands with savings percentages

### 9.6 Verification

```bash
./bin/syt discover --since 7
# Should show commands from the last 7 days of Claude Code sessions
# (Requires active Claude Code sessions to have data)

./bin/syt discover --format json | jq .supported[0]
```

---

## Phase 10: Remaining Filter Modules

**Goal**: Implement the remaining high-value filter modules.

Implement in this priority order (highest token savings first):

### 10.1 `cmd/vitest.go` — Target: 99% savings

- Execute: `npx vitest run [args]` or `vitest run [args]`
- Filter: strip all ANSI codes, show only `FAIL` test blocks + summary line
- All-pass format: `47 tests passed ✓ (3.2s)`

### 10.2 `cmd/playwright.go` — Target: 94% savings

- Execute: `npx playwright test [args]`
- Filter: group failures by test suite, show full failure details, suppress passing suites
- Summary: `23 passed, 2 failed ✗`

### 10.3 `cmd/tsc.go` — Target: 83% savings

- Execute: `npx tsc [args]` or `tsc [args]`
- Filter: parse TypeScript error format `file(line,col): error TS####: message`
- Group by file, then by error code
- Summary: `14 errors in 4 files`
- No errors: `no TypeScript errors ✓`

### 10.4 `cmd/lint.go` — Target: 84% savings

Supports ESLint and Biome (auto-detect from project config files):

- Execute: `npx eslint [args]` or `biome check [args]`
- Filter: group violations by rule name
- Format: `{rule}: {count} occurrences in {n} files`
- No violations: `no lint errors ✓`

### 10.5 `cmd/next_cmd.go` — Target: 87% savings

- Execute: `next build [args]`
- Filter: extract route table, bundle size stats, suppress webpack compilation lines
- Format:
  ```
  Route (app)                    Size    First Load JS
  ┌ ○ /                         142 B   89.3 kB
  ├ ○ /about                    142 B   89.3 kB
  └ ƒ /api/users                0 B     84.2 kB

  ○ Static  ƒ Dynamic   build: 23.4s ✓
  ```

### 10.6 `cmd/prisma.go` — Target: 88% savings

- Execute: `npx prisma [subcommand] [args]`
- Filter: strip Prisma ASCII art banner, verbose initialization messages
- Preserve: actual migration output, error messages, schema validation output

### 10.7 `cmd/pnpm.go` — Target: 70-90% savings

- `syt pnpm list [--depth N]`: compact tree, `{name}@{version}` per line
- `syt pnpm outdated`: table of outdated packages (current → wanted → latest)
- `syt pnpm install [--prod]`: suppress progress bars, show "installed X packages in Ys ✓"

### 10.8 `cmd/pytest.go` — Target: 90% savings

State machine parser for pytest output:
- States: COLLECTING → RUNNING → PASSED/FAILED
- Show: FAILED test names + failure details + summary line
- Suppress: dot-per-test progress, passed test names
- All-pass: `47 passed in 3.2s ✓`

### 10.9 `cmd/ruff.go` — Target: 80% savings

- `syt ruff check [args]`: run `ruff check --output-format json`, parse JSON, group by rule code
- `syt ruff format [args]`: run `ruff format`, show only files that were changed

### 10.10 `cmd/pip.go` — Target: 70-85% savings

Auto-detect `uv` (if installed, use `uv pip` instead of `pip`):
- `syt pip install [args]`: suppress download progress, show "installed X packages ✓"
- `syt pip list`: compact `{name}=={version}` per line
- `syt pip outdated`: table of outdated packages

### 10.11 `cmd/docker.go` — Target: 60% savings

- `syt docker ps`: compact table (name, image, status, ports)
- `syt docker build`: suppress layer pull progress, show layer cache hits count, final "built in Xs ✓"
- `syt docker logs [container]`: last 50 lines with timestamp prefix stripped

### 10.12 `cmd/gh.go` — Target: 26-87% savings

- `syt gh pr view [number]`: compact PR summary (title, status, checks, reviewers)
- `syt gh pr list`: compact table (number, title, branch, age)
- `syt gh issue view [number]`: title, status, labels, body (truncated to 500 chars)
- `syt gh run view [id]`: job status summary

### 10.13 `cmd/grep.go`, `cmd/ls.go`, `cmd/read.go`

- `syt grep [args]`: group matches by file, truncate long lines at 120 chars
- `syt ls [args]`: tree format with aggregate counts
- `syt read [file]` / `syt cat [file]`: apply language-aware comment stripping at FilterLevel.Minimal

### Per-Module Test Requirements

For every module above:
1. Create fixture: `tests/fixtures/{cmd}_raw.txt` with real command output
2. Write token savings test: verify ≥ target savings %
3. Write golden file test: output matches `tests/snapshots/{cmd}_filtered.txt`
4. Write edge case tests: empty input, error output, unicode, ANSI codes

---

## Phase 11: Testing

**Goal**: Comprehensive test coverage before release.

### 11.1 Unit Test Coverage

Run `go test -cover ./...` — target ≥80% coverage.

Priority areas:
- `internal/registry`: every rule has a test case
- `internal/tracker`: all CRUD operations
- `internal/tee`: rotation, size limits, mode conditions
- `internal/config`: env var overrides, defaults
- All `cmd/` filter functions with fixtures

### 11.2 Token Accuracy Tests

Every filter must have a test that:
1. Loads its fixture from `tests/fixtures/`
2. Applies the filter
3. Asserts `savings >= target_pct`

```go
func assertTokenSavings(t *testing.T, input, output string, minSavingsPct float64) {
    t.Helper()
    inputTok := utils.CountTokens(input)
    outputTok := utils.CountTokens(output)
    savings := 100.0 - float64(outputTok)/float64(inputTok)*100.0
    assert.GreaterOrEqual(t, savings, minSavingsPct,
        "token savings %.1f%% < required %.1f%%", savings, minSavingsPct)
}
```

### 11.3 Golden File Tests

Use `testify/assert.Equal` with `testdata/` golden files. On first run (or `UPDATE_GOLDEN=1`), write the golden file. On subsequent runs, compare.

```go
func TestGitLogGolden(t *testing.T) {
    input := readFixture(t, "git_log_raw.txt")
    output := filterGitLog(input, "")
    assertGolden(t, "git_log_filtered.txt", output)
}

func assertGolden(t *testing.T, name, got string) {
    t.Helper()
    path := filepath.Join("../tests/snapshots", name)
    if os.Getenv("UPDATE_GOLDEN") == "1" {
        os.WriteFile(path, []byte(got), 0644)
        return
    }
    want := string(must(os.ReadFile(path)))
    assert.Equal(t, want, got)
}
```

### 11.4 Smoke Tests

Create `scripts/test-all.sh` that exercises every `syt` subcommand against real tools:

```bash
#!/usr/bin/env bash
set -e

PASS=0; FAIL=0

check() {
    local desc="$1"; shift
    if "$@" >/dev/null 2>&1; then
        echo "PASS: $desc"; ((PASS++))
    else
        echo "FAIL: $desc"; ((FAIL++))
    fi
}

check "syt git log"      syt git log -5
check "syt git status"   syt git status
check "syt gain"         syt gain
check "syt rewrite git"  syt rewrite "git log -5"
# ... etc for every command

echo "$PASS passed, $FAIL failed"
[[ $FAIL -eq 0 ]]
```

### 11.5 Integration Tests

Tag with `//go:build integration` — skipped in normal `go test`, run with `go test -tags integration`:

```go
//go:build integration

func TestRealGitLog(t *testing.T) {
    cmd := exec.Command("syt", "git", "log", "-5")
    out, err := cmd.Output()
    require.NoError(t, err)
    assert.NotEmpty(t, out)
    assert.Less(t, len(out), 2000, "output too large — filter not working")
}
```

### 11.6 Cross-Platform Tests

Add `//go:build` tags for platform-specific assertions where behavior differs:

```go
func TestConfigPath(t *testing.T) {
    path := config.ConfigDir()
    //go:build darwin
    assert.Contains(t, path, "Library/Application Support")
    //go:build linux
    assert.Contains(t, path, ".config")
}
```

---

## Phase 12: Release & CI

**Goal**: Multi-platform binary distribution via GitHub Actions and goreleaser.

### 12.1 Create `.github/workflows/ci.yml`

Run on every push and PR:
```yaml
- go fmt ./... && git diff --exit-code    # format check
- go vet ./...                            # vet
- go test ./...                           # unit tests
- go build .                             # compile check
```

### 12.2 Create `.github/workflows/release.yml`

Run on `v*` tags:
```yaml
matrix:
  include:
    - {os: ubuntu-latest,  GOOS: linux,   GOARCH: amd64}
    - {os: ubuntu-latest,  GOOS: linux,   GOARCH: arm64}
    - {os: macos-latest,   GOOS: darwin,  GOARCH: amd64}
    - {os: macos-latest,   GOOS: darwin,  GOARCH: arm64}
    - {os: windows-latest, GOOS: windows, GOARCH: amd64}

steps:
  - go test ./...
  - go build -ldflags="-s -w" -o syt_{GOOS}_{GOARCH}
  - goreleaser release
```

### 12.3 Create `.goreleaser.yml`

```yaml
project_name: syt
builds:
  - binary: syt
    ldflags: ["-s", "-w"]
    env: [CGO_ENABLED=0]
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]

archives:
  - format: tar.gz
    format_overrides:
      - goos: windows
        format: zip

checksum:
  name_template: checksums.txt

nfpms:  # DEB and RPM packages
  - formats: [deb, rpm]
    homepage: https://github.com/saveyourtokens/syt

brews:  # Homebrew tap
  - tap:
      owner: saveyourtokens
      name: homebrew-tap
```

### 12.4 Version Embedding

Embed version at build time:

```go
// main.go
var Version = "dev"  // overridden by ldflags

// .goreleaser.yml
ldflags: ["-X main.Version={{.Version}}"]
```

### 12.5 Pre-Release Checklist

Before tagging a release:
- [ ] `go test ./...` passes on macOS, Linux, Windows
- [ ] `scripts/test-all.sh` passes with installed binary
- [ ] `syt init` works on clean Claude Code installation
- [ ] Token savings ≥ targets for all filters (run `go test -run TestTokenSavings ./...`)
- [ ] Binary size < 15MB stripped (`ls -lh dist/syt_linux_amd64`)
- [ ] Startup overhead < 10ms (`hyperfine 'syt git status' 'git status' --warmup 5`)

---

## Appendix: Dependency Reference

```
github.com/spf13/cobra             CLI framework
github.com/BurntSushi/toml         TOML config parsing
modernc.org/sqlite                 SQLite (pure Go, no CGO)
github.com/stretchr/testify        Test assertions
github.com/fatih/color             Terminal colors
```

## Appendix: Token Savings Targets Summary

| Command | Target | Phase |
|---------|--------|-------|
| `git log` | ≥80% | Phase 3 |
| `git diff` | ≥75% | Phase 3 |
| `git status` | ≥80% | Phase 3 |
| `cargo test` | ≥90% | Phase 3 |
| `cargo build` | ≥80% | Phase 3 |
| `vitest` | ≥99% | Phase 10 |
| `playwright` | ≥94% | Phase 10 |
| `pytest` | ≥90% | Phase 10 |
| `tsc` | ≥83% | Phase 10 |
| `eslint/biome` | ≥84% | Phase 10 |
| `next build` | ≥87% | Phase 10 |
| `prisma` | ≥88% | Phase 10 |
| `pnpm list` | ≥70% | Phase 10 |
| `ruff check` | ≥80% | Phase 10 |
| `pip list` | ≥70% | Phase 10 |
| `go test` | ≥90% | Phase 10 |
| `docker ps` | ≥60% | Phase 10 |
| `gh pr view` | ≥60% | Phase 10 |
