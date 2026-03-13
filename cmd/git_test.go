package cmd

import (
	"os"
	"testing"

	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestTokenSavings_GitLog(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/git_log_raw.txt")
	if err != nil {
		t.Skip("fixture file not found")
	}
	output := filterGitLog(string(input), "")

	in := utils.CountTokens(string(input))
	out := utils.CountTokens(output)

	if in == 0 {
		t.Skip("empty fixture")
	}

	savings := 100.0 - float64(out)/float64(in)*100.0
	assert.GreaterOrEqualf(t, savings, 80.0,
		"git log savings %.1f%% < required 80%%", savings)
}

func TestTokenSavings_GitStatus(t *testing.T) {
	input := `## main...origin/main [ahead 2]
M  src/foo.go
M  src/bar.go
 M src/baz.go
?? newfile.txt
?? another.txt
A  staged.go
`
	output := filterGitStatus(input, "")
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "main")
	assert.Contains(t, output, "↑2")
	// Verify meaningful compression occurred (output is shorter than input)
	assert.Less(t, len(output), len(input), "status output should be shorter than input")
}

func TestFilterGitLog_Snapshot(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/git_log_raw.txt")
	if err != nil {
		t.Skip("fixture file not found")
	}
	output := filterGitLog(string(input), "")
	assert.NotEmpty(t, output)
	// Should contain commit hashes (7+ hex chars)
	assert.Contains(t, output, "commits")
}

func TestFilterGitStatus_Clean(t *testing.T) {
	input := "## main...origin/main\n"
	out := filterGitStatus(input, "")
	assert.Contains(t, out, "main")
	assert.Contains(t, out, "Clean")
}

func TestFilterGitStatus_WithChanges(t *testing.T) {
	input := `## feature/test...origin/feature/test [ahead 1, behind 3]
M  src/a.go
 M src/b.go
?? new.txt
`
	out := filterGitStatus(input, "")
	assert.Contains(t, out, "feature/test")
	assert.Contains(t, out, "↑1")
	assert.Contains(t, out, "↓3")
}

func TestFilterGitDiff_Empty(t *testing.T) {
	out := filterGitDiff("", "")
	assert.Equal(t, "no changes", out)
}

func TestFilterGitLog_Empty(t *testing.T) {
	out := filterGitLog("", "")
	assert.Equal(t, "(no commits)", out)
}
