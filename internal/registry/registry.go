package registry

import (
	"regexp"
	"strings"
)

// Category represents the type of command.
type Category string

const (
	CategoryGit       Category = "git"
	CategoryBuild     Category = "build"
	CategoryTest      Category = "test"
	CategoryLint      Category = "lint"
	CategoryPackage   Category = "package"
	CategoryFile      Category = "file"
	CategoryContainer Category = "container"
	CategoryGitHub    Category = "github"
)

// Rule defines a single rewrite rule.
type Rule struct {
	Pattern  *regexp.Regexp
	Rewrite  func(cmd string) string
	Category Category
	SavesPct int // estimated savings percentage
}

// Classification holds full metadata for a command classification.
type Classification struct {
	Kind     string   // "supported", "unsupported", "ignored"
	SytCmd   string   // rewritten command if Kind == "supported"
	Category Category
	SavesPct int
}

var rules []Rule

// ignoredPrefixes are commands that must never be rewritten.
var ignoredPrefixes = []string{
	"syt", "#", "cd ", "cd\t", "pwd", "echo ", "echo\t", "export ",
	"source ", ". ", "env ", "env\t",
}

var ignoredExact = map[string]bool{
	"cd": true, "pwd": true, "echo": true, "export": true,
	"source": true, ".": true, "env": true,
}

func init() {
	rules = []Rule{
		// ── Git ──────────────────────────────────────────────────────────
		{
			Pattern:  regexp.MustCompile(`^git\s+log(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt git log" + extractArgs(cmd, "git log") },
			Category: CategoryGit,
			SavesPct: 80,
		},
		{
			Pattern:  regexp.MustCompile(`^git\s+diff(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt git diff" + extractArgs(cmd, "git diff") },
			Category: CategoryGit,
			SavesPct: 75,
		},
		{
			Pattern:  regexp.MustCompile(`^git\s+status(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt git status" + extractArgs(cmd, "git status") },
			Category: CategoryGit,
			SavesPct: 80,
		},
		{
			Pattern:  regexp.MustCompile(`^git\s+add(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt git add" + extractArgs(cmd, "git add") },
			Category: CategoryGit,
			SavesPct: 90,
		},
		{
			Pattern:  regexp.MustCompile(`^git\s+commit(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt git commit" + extractArgs(cmd, "git commit") },
			Category: CategoryGit,
			SavesPct: 90,
		},
		{
			Pattern:  regexp.MustCompile(`^git\s+push(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt git push" + extractArgs(cmd, "git push") },
			Category: CategoryGit,
			SavesPct: 90,
		},
		{
			Pattern:  regexp.MustCompile(`^git\s+pull(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt git pull" + extractArgs(cmd, "git pull") },
			Category: CategoryGit,
			SavesPct: 90,
		},
		{
			Pattern:  regexp.MustCompile(`^git\s+fetch(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt git fetch" + extractArgs(cmd, "git fetch") },
			Category: CategoryGit,
			SavesPct: 90,
		},
		{
			Pattern:  regexp.MustCompile(`^git\s+branch(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt git branch" + extractArgs(cmd, "git branch") },
			Category: CategoryGit,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^git\s+stash(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt git stash" + extractArgs(cmd, "git stash") },
			Category: CategoryGit,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^git\s+worktree(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt git worktree" + extractArgs(cmd, "git worktree") },
			Category: CategoryGit,
			SavesPct: 70,
		},

		// ── Go ───────────────────────────────────────────────────────────
		{
			Pattern:  regexp.MustCompile(`^go\s+test(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt go test" + extractArgs(cmd, "go test") },
			Category: CategoryTest,
			SavesPct: 90,
		},
		{
			Pattern:  regexp.MustCompile(`^go\s+build(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt go build" + extractArgs(cmd, "go build") },
			Category: CategoryBuild,
			SavesPct: 80,
		},
		{
			Pattern:  regexp.MustCompile(`^go\s+vet(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt go vet" + extractArgs(cmd, "go vet") },
			Category: CategoryLint,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^go\s+run(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt go run" + extractArgs(cmd, "go run") },
			Category: CategoryBuild,
			SavesPct: 60,
		},

		// ── Cargo ─────────────────────────────────────────────────────────
		{
			Pattern:  regexp.MustCompile(`^cargo\s+test(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt cargo test" + extractArgs(cmd, "cargo test") },
			Category: CategoryTest,
			SavesPct: 90,
		},
		{
			Pattern:  regexp.MustCompile(`^cargo\s+build(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt cargo build" + extractArgs(cmd, "cargo build") },
			Category: CategoryBuild,
			SavesPct: 80,
		},
		{
			Pattern:  regexp.MustCompile(`^cargo\s+clippy(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt cargo clippy" + extractArgs(cmd, "cargo clippy") },
			Category: CategoryLint,
			SavesPct: 75,
		},
		{
			Pattern:  regexp.MustCompile(`^cargo\s+check(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt cargo check" + extractArgs(cmd, "cargo check") },
			Category: CategoryBuild,
			SavesPct: 75,
		},
		{
			Pattern:  regexp.MustCompile(`^cargo\s+run(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt cargo run" + extractArgs(cmd, "cargo run") },
			Category: CategoryBuild,
			SavesPct: 60,
		},

		// ── pnpm ──────────────────────────────────────────────────────────
		{
			Pattern:  regexp.MustCompile(`^pnpm\s+list(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt pnpm list" + extractArgs(cmd, "pnpm list") },
			Category: CategoryPackage,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^pnpm\s+install(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt pnpm install" + extractArgs(cmd, "pnpm install") },
			Category: CategoryPackage,
			SavesPct: 80,
		},
		{
			Pattern:  regexp.MustCompile(`^pnpm\s+outdated(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt pnpm outdated" + extractArgs(cmd, "pnpm outdated") },
			Category: CategoryPackage,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^pnpm\s+add(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt pnpm add" + extractArgs(cmd, "pnpm add") },
			Category: CategoryPackage,
			SavesPct: 80,
		},

		// ── npm ───────────────────────────────────────────────────────────
		{
			Pattern:  regexp.MustCompile(`^npm\s+install(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt npm install" + extractArgs(cmd, "npm install") },
			Category: CategoryPackage,
			SavesPct: 80,
		},
		{
			Pattern:  regexp.MustCompile(`^npm\s+run(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt npm run" + extractArgs(cmd, "npm run") },
			Category: CategoryBuild,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^npm\s+test(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt npm test" + extractArgs(cmd, "npm test") },
			Category: CategoryTest,
			SavesPct: 85,
		},

		// ── pip / uv ──────────────────────────────────────────────────────
		{
			Pattern:  regexp.MustCompile(`^pip\s+install(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt pip install" + extractArgs(cmd, "pip install") },
			Category: CategoryPackage,
			SavesPct: 80,
		},
		{
			Pattern:  regexp.MustCompile(`^pip\s+list(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt pip list" + extractArgs(cmd, "pip list") },
			Category: CategoryPackage,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^pip\s+outdated(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt pip outdated" + extractArgs(cmd, "pip outdated") },
			Category: CategoryPackage,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^uv\s+pip\s+install(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt pip install" + extractArgs(cmd, "uv pip install") },
			Category: CategoryPackage,
			SavesPct: 80,
		},
		{
			Pattern:  regexp.MustCompile(`^uv\s+pip\s+list(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt pip list" + extractArgs(cmd, "uv pip list") },
			Category: CategoryPackage,
			SavesPct: 70,
		},

		// ── TypeScript/JS ─────────────────────────────────────────────────
		{
			Pattern:  regexp.MustCompile(`^tsc(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt tsc" + extractArgs(cmd, "tsc") },
			Category: CategoryBuild,
			SavesPct: 83,
		},
		{
			Pattern:  regexp.MustCompile(`^eslint(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt lint eslint" + extractArgs(cmd, "eslint") },
			Category: CategoryLint,
			SavesPct: 75,
		},
		{
			Pattern:  regexp.MustCompile(`^biome\s+check(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt lint biome" + extractArgs(cmd, "biome check") },
			Category: CategoryLint,
			SavesPct: 75,
		},
		{
			Pattern:  regexp.MustCompile(`^vitest\s+run(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt vitest" + extractArgs(cmd, "vitest run") },
			Category: CategoryTest,
			SavesPct: 99,
		},
		{
			Pattern:  regexp.MustCompile(`^vitest(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt vitest" + extractArgs(cmd, "vitest") },
			Category: CategoryTest,
			SavesPct: 99,
		},
		{
			Pattern:  regexp.MustCompile(`^next\s+build(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt next build" + extractArgs(cmd, "next build") },
			Category: CategoryBuild,
			SavesPct: 85,
		},
		{
			Pattern:  regexp.MustCompile(`^next\s+dev(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt next dev" + extractArgs(cmd, "next dev") },
			Category: CategoryBuild,
			SavesPct: 80,
		},
		{
			Pattern:  regexp.MustCompile(`^prisma\s+migrate(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt prisma migrate" + extractArgs(cmd, "prisma migrate") },
			Category: CategoryBuild,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^prisma\s+generate(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt prisma generate" + extractArgs(cmd, "prisma generate") },
			Category: CategoryBuild,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^prettier\s+--check(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt lint prettier" + extractArgs(cmd, "prettier --check") },
			Category: CategoryLint,
			SavesPct: 75,
		},

		// ── Python ────────────────────────────────────────────────────────
		{
			Pattern:  regexp.MustCompile(`^pytest(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt pytest" + extractArgs(cmd, "pytest") },
			Category: CategoryTest,
			SavesPct: 90,
		},
		{
			Pattern:  regexp.MustCompile(`^ruff\s+check(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt ruff check" + extractArgs(cmd, "ruff check") },
			Category: CategoryLint,
			SavesPct: 75,
		},
		{
			Pattern:  regexp.MustCompile(`^ruff\s+format(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt ruff format" + extractArgs(cmd, "ruff format") },
			Category: CategoryLint,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^mypy(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt lint mypy" + extractArgs(cmd, "mypy") },
			Category: CategoryLint,
			SavesPct: 75,
		},

		// ── GitHub CLI ────────────────────────────────────────────────────
		{
			Pattern:  regexp.MustCompile(`^gh\s+pr\s+view(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt gh pr view" + extractArgs(cmd, "gh pr view") },
			Category: CategoryGitHub,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^gh\s+pr\s+list(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt gh pr list" + extractArgs(cmd, "gh pr list") },
			Category: CategoryGitHub,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^gh\s+issue\s+view(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt gh issue view" + extractArgs(cmd, "gh issue view") },
			Category: CategoryGitHub,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^gh\s+issue\s+list(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt gh issue list" + extractArgs(cmd, "gh issue list") },
			Category: CategoryGitHub,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^gh\s+run\s+view(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt gh run view" + extractArgs(cmd, "gh run view") },
			Category: CategoryGitHub,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^gh\s+run\s+list(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt gh run list" + extractArgs(cmd, "gh run list") },
			Category: CategoryGitHub,
			SavesPct: 70,
		},

		// ── Files ─────────────────────────────────────────────────────────
		{
			Pattern:  regexp.MustCompile(`^grep(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt grep" + extractArgs(cmd, "grep") },
			Category: CategoryFile,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^rg(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt rg" + extractArgs(cmd, "rg") },
			Category: CategoryFile,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^ls\s+-la(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt ls" + extractArgs(cmd, "ls -la") },
			Category: CategoryFile,
			SavesPct: 60,
		},
		{
			Pattern:  regexp.MustCompile(`^ls(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt ls" + extractArgs(cmd, "ls") },
			Category: CategoryFile,
			SavesPct: 60,
		},
		{
			Pattern:  regexp.MustCompile(`^find(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt grep" + extractArgs(cmd, "find") },
			Category: CategoryFile,
			SavesPct: 60,
		},
		{
			Pattern:  regexp.MustCompile(`^cat(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt read" + extractArgs(cmd, "cat") },
			Category: CategoryFile,
			SavesPct: 60,
		},
		{
			Pattern:  regexp.MustCompile(`^bat(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt read" + extractArgs(cmd, "bat") },
			Category: CategoryFile,
			SavesPct: 65,
		},

		// ── Containers ───────────────────────────────────────────────────
		{
			Pattern:  regexp.MustCompile(`^docker\s+ps(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt docker ps" + extractArgs(cmd, "docker ps") },
			Category: CategoryContainer,
			SavesPct: 70,
		},
		{
			Pattern:  regexp.MustCompile(`^docker\s+build(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt docker build" + extractArgs(cmd, "docker build") },
			Category: CategoryContainer,
			SavesPct: 80,
		},
		{
			Pattern:  regexp.MustCompile(`^docker\s+logs(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt docker logs" + extractArgs(cmd, "docker logs") },
			Category: CategoryContainer,
			SavesPct: 70,
		},
		// ── Make ────────────────────────────────────────────────
		{
			Pattern:  regexp.MustCompile(`^make\s+test(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt proxy make test" + extractArgs(cmd, "make test") },
			Category: CategoryTest,
			SavesPct: 80,
		},
		{
			Pattern:  regexp.MustCompile(`^make\s+build(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt proxy make build" + extractArgs(cmd, "make build") },
			Category: CategoryBuild,
			SavesPct: 75,
		},
		{
			Pattern:  regexp.MustCompile(`^make\s+install(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt proxy make install" + extractArgs(cmd, "make install") },
			Category: CategoryBuild,
			SavesPct: 75,
		},
		{
			Pattern:  regexp.MustCompile(`^make\s+lint(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt proxy make lint" + extractArgs(cmd, "make lint") },
			Category: CategoryLint,
			SavesPct: 75,
		},

		// ── Yarn ────────────────────────────────────────────────
		{
			Pattern:  regexp.MustCompile(`^yarn\s+install(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt proxy yarn install" + extractArgs(cmd, "yarn install") },
			Category: CategoryPackage,
			SavesPct: 80,
		},
		{
			Pattern:  regexp.MustCompile(`^yarn\s+add(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt proxy yarn add" + extractArgs(cmd, "yarn add") },
			Category: CategoryPackage,
			SavesPct: 80,
		},
		{
			Pattern:  regexp.MustCompile(`^yarn\s+test(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt proxy yarn test" + extractArgs(cmd, "yarn test") },
			Category: CategoryTest,
			SavesPct: 85,
		},

		// ── Bun ──────────────────────────────────────────────────
		{
			Pattern:  regexp.MustCompile(`^bun\s+install(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt proxy bun install" + extractArgs(cmd, "bun install") },
			Category: CategoryPackage,
			SavesPct: 80,
		},
		{
			Pattern:  regexp.MustCompile(`^bun\s+test(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt proxy bun test" + extractArgs(cmd, "bun test") },
			Category: CategoryTest,
			SavesPct: 90,
		},
		{
			Pattern:  regexp.MustCompile(`^bun\s+run(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt proxy bun run" + extractArgs(cmd, "bun run") },
			Category: CategoryBuild,
			SavesPct: 70,
		},

		// ── Poetry ──────────────────────────────────────────────
		{
			Pattern:  regexp.MustCompile(`^poetry\s+install(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt proxy poetry install" + extractArgs(cmd, "poetry install") },
			Category: CategoryPackage,
			SavesPct: 80,
		},
		{
			Pattern:  regexp.MustCompile(`^poetry\s+add(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt proxy poetry add" + extractArgs(cmd, "poetry add") },
			Category: CategoryPackage,
			SavesPct: 75,
		},

		// ── Jest ────────────────────────────────────────────────
		{
			Pattern:  regexp.MustCompile(`^jest(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt proxy jest" + extractArgs(cmd, "jest") },
			Category: CategoryTest,
			SavesPct: 90,
		},

		// ── Nx ──────────────────────────────────────────────────
		{
			Pattern:  regexp.MustCompile(`^nx\s+test(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt proxy nx test" + extractArgs(cmd, "nx test") },
			Category: CategoryTest,
			SavesPct: 80,
		},
		{
			Pattern:  regexp.MustCompile(`^nx\s+build(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt proxy nx build" + extractArgs(cmd, "nx build") },
			Category: CategoryBuild,
			SavesPct: 80,
		},

		// ── Playwright (direct) ─────────────────────────────────
		{
			Pattern:  regexp.MustCompile(`^npx\s+playwright(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt playwright" + extractArgs(cmd, "npx playwright") },
			Category: CategoryTest,
			SavesPct: 94,
		},
		{
			Pattern:  regexp.MustCompile(`^playwright(\s|$)`),
			Rewrite:  func(cmd string) string { return "syt playwright" + extractArgs(cmd, "playwright") },
			Category: CategoryTest,
			SavesPct: 94,
		},
	}
}

// extractArgs extracts the args portion after the prefix cmd.
func extractArgs(full, prefix string) string {
	if strings.HasPrefix(full, prefix) {
		rest := full[len(prefix):]
		if rest == "" {
			return ""
		}
		return " " + strings.TrimSpace(rest)
	}
	// Try splitting on whitespace and rebuilding
	fields := strings.Fields(full)
	prefixFields := strings.Fields(prefix)
	if len(fields) <= len(prefixFields) {
		return ""
	}
	return " " + strings.Join(fields[len(prefixFields):], " ")
}

// isIgnored returns true if the command should never be rewritten.
func isIgnored(cmd string) bool {
	if cmd == "" {
		return true
	}
	// Check exact matches
	first := strings.Fields(cmd)[0]
	if ignoredExact[first] {
		return true
	}
	// Check prefix
	for _, p := range ignoredPrefixes {
		if cmd == strings.TrimRight(p, " \t") || strings.HasPrefix(cmd, p) {
			return true
		}
	}
	// Multiline commands
	if strings.Contains(cmd, "\n") {
		return true
	}
	return false
}

// RewriteCommand returns the syt equivalent of cmd, or "" if not found.
func RewriteCommand(cmd string, excluded []string) string {
	cmd = strings.TrimSpace(cmd)
	if isIgnored(cmd) {
		return ""
	}
	// Check excluded list
	for _, ex := range excluded {
		if strings.HasPrefix(cmd, ex) {
			return ""
		}
	}
	for _, r := range rules {
		if r.Pattern.MatchString(cmd) {
			return r.Rewrite(cmd)
		}
	}
	return ""
}

// ClassifyCommand returns full classification metadata.
func ClassifyCommand(cmd string) Classification {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return Classification{Kind: "ignored"}
	}
	if strings.HasPrefix(cmd, "syt ") || cmd == "syt" {
		return Classification{Kind: "ignored", SytCmd: cmd}
	}
	if isIgnored(cmd) {
		return Classification{Kind: "ignored"}
	}
	for _, r := range rules {
		if r.Pattern.MatchString(cmd) {
			return Classification{
				Kind:     "supported",
				SytCmd:   r.Rewrite(cmd),
				Category: r.Category,
				SavesPct: r.SavesPct,
			}
		}
	}
	return Classification{Kind: "unsupported"}
}
