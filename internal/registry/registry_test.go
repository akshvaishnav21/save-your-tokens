package registry

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRewriteCommand_GitLog(t *testing.T) {
	// Positive
	assert.Equal(t, "syt git log", RewriteCommand("git log", nil))
	assert.Equal(t, "syt git log -10", RewriteCommand("git log -10", nil))
	assert.Equal(t, "syt git log --oneline", RewriteCommand("git log --oneline", nil))

	// Negative: should not be empty (git status rewrites to syt git status)
	assert.NotEqual(t, "syt git log", RewriteCommand("git status", nil))
	// "git logfile" doesn't match "git log" pattern (which requires space or end)
	// but could match other patterns; just verify it's not "syt git log"
	result := RewriteCommand("git logfile", nil)
	assert.NotEqual(t, "syt git log", result)

	// Exclusion
	assert.Equal(t, "", RewriteCommand("git log", []string{"git log"}))
}

func TestRewriteCommand_GitStatus(t *testing.T) {
	assert.Equal(t, "syt git status", RewriteCommand("git status", nil))
	assert.Equal(t, "syt git status --short", RewriteCommand("git status --short", nil))
	assert.Equal(t, "", RewriteCommand("git statusbar", nil))
}

func TestRewriteCommand_GoTest(t *testing.T) {
	assert.Equal(t, "syt go test", RewriteCommand("go test", nil))
	assert.Equal(t, "syt go test ./...", RewriteCommand("go test ./...", nil))
	assert.Equal(t, "syt go test -v ./...", RewriteCommand("go test -v ./...", nil))
	// go build rewrites to syt go build (not same as go test)
	assert.Equal(t, "syt go build", RewriteCommand("go build", nil))
	// Verify the go test result is not go build result
	assert.NotEqual(t, RewriteCommand("go test", nil), RewriteCommand("go build", nil))
}

func TestRewriteCommand_CargoTest(t *testing.T) {
	assert.Equal(t, "syt cargo test", RewriteCommand("cargo test", nil))
	assert.Equal(t, "syt cargo test --release", RewriteCommand("cargo test --release", nil))
	// cargo clippy rewrites to syt cargo clippy (different from cargo test)
	assert.Equal(t, "syt cargo clippy", RewriteCommand("cargo clippy", nil))
	assert.NotEqual(t, RewriteCommand("cargo test", nil), RewriteCommand("cargo clippy", nil))
}

func TestRewriteCommand_NeverRewrite(t *testing.T) {
	// Already syt
	assert.Equal(t, "", RewriteCommand("syt git log", nil))
	assert.Equal(t, "", RewriteCommand("syt rewrite foo", nil))

	// Comments
	assert.Equal(t, "", RewriteCommand("# some comment", nil))

	// Shell builtins
	assert.Equal(t, "", RewriteCommand("cd /tmp", nil))
	assert.Equal(t, "", RewriteCommand("pwd", nil))
	assert.Equal(t, "", RewriteCommand("echo hello", nil))
	assert.Equal(t, "", RewriteCommand("export FOO=bar", nil))
	assert.Equal(t, "", RewriteCommand("source ~/.bashrc", nil))

	// Empty
	assert.Equal(t, "", RewriteCommand("", nil))
	assert.Equal(t, "", RewriteCommand("  ", nil))

	// Multiline
	assert.Equal(t, "", RewriteCommand("git log\ngit status", nil))
}

func TestRewriteCommand_AllCategories(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"git diff", "syt git diff"},
		{"git add .", "syt git add ."},
		{"git commit -m 'foo'", "syt git commit -m 'foo'"},
		{"git push", "syt git push"},
		{"git pull", "syt git pull"},
		{"git fetch", "syt git fetch"},
		{"git branch", "syt git branch"},
		{"git stash", "syt git stash"},
		{"git worktree list", "syt git worktree list"},
		{"go build ./...", "syt go build ./..."},
		{"go vet ./...", "syt go vet ./..."},
		{"go run main.go", "syt go run main.go"},
		{"cargo build", "syt cargo build"},
		{"cargo clippy", "syt cargo clippy"},
		{"cargo check", "syt cargo check"},
		{"cargo run", "syt cargo run"},
		{"pnpm list", "syt pnpm list"},
		{"pnpm install", "syt pnpm install"},
		{"pnpm outdated", "syt pnpm outdated"},
		{"pnpm add react", "syt pnpm add react"},
		{"npm install", "syt npm install"},
		{"npm run build", "syt npm run build"},
		{"npm test", "syt npm test"},
		{"pip install flask", "syt pip install flask"},
		{"pip list", "syt pip list"},
		{"tsc --noEmit", "syt tsc --noEmit"},
		{"eslint src/", "syt lint eslint src/"},
		{"biome check .", "syt lint biome ."},
		{"vitest run", "syt vitest"},
		{"vitest", "syt vitest"},
		{"next build", "syt next build"},
		{"next dev", "syt next dev"},
		{"prisma migrate deploy", "syt prisma migrate deploy"},
		{"prisma generate", "syt prisma generate"},
		{"pytest", "syt pytest"},
		{"pytest -v tests/", "syt pytest -v tests/"},
		{"ruff check .", "syt ruff check ."},
		{"ruff format .", "syt ruff format ."},
		{"mypy src/", "syt lint mypy src/"},
		{"gh pr view", "syt gh pr view"},
		{"gh pr list", "syt gh pr list"},
		{"gh issue view 123", "syt gh issue view 123"},
		{"gh issue list", "syt gh issue list"},
		{"gh run view", "syt gh run view"},
		{"gh run list", "syt gh run list"},
		{"grep -r foo .", "syt grep -r foo ."},
		{"rg pattern src/", "syt rg pattern src/"},
		{"ls", "syt ls"},
		{"ls -la", "syt ls"},
		{"cat file.txt", "syt read file.txt"},
		{"bat file.go", "syt read file.go"},
		{"docker ps", "syt docker ps"},
		{"docker build .", "syt docker build ."},
		{"docker logs mycontainer", "syt docker logs mycontainer"},
		{"uv pip install flask", "syt pip install flask"},
		{"uv pip list", "syt pip list"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := RewriteCommand(tt.input, nil)
			assert.Equal(t, tt.expected, got, "input: %q", tt.input)
		})
	}
}

func TestClassifyCommand(t *testing.T) {
	c := ClassifyCommand("git log -10")
	assert.Equal(t, "supported", c.Kind)
	assert.Equal(t, "syt git log -10", c.SytCmd)
	assert.Equal(t, CategoryGit, c.Category)
	assert.Equal(t, 80, c.SavesPct)

	c2 := ClassifyCommand("syt git log")
	assert.Equal(t, "ignored", c2.Kind)

	c3 := ClassifyCommand("some-unknown-tool --flag")
	assert.Equal(t, "unsupported", c3.Kind)

	c4 := ClassifyCommand("")
	assert.Equal(t, "ignored", c4.Kind)

	c5 := ClassifyCommand("cd /tmp")
	assert.Equal(t, "ignored", c5.Kind)
}

func TestClassifyCommand_SavesPct(t *testing.T) {
	tests := []struct {
		cmd      string
		minSaves int
	}{
		{"cargo test", 90},
		{"git log", 80},
		{"vitest", 99},
		{"pytest", 90},
	}
	for _, tt := range tests {
		c := ClassifyCommand(tt.cmd)
		assert.Equal(t, "supported", c.Kind)
		assert.GreaterOrEqual(t, c.SavesPct, tt.minSaves)
	}
}
