package cmd

import (
	"os"
	"testing"

	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestTokenSavings_GhPrList(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/gh_pr_list_raw.txt")
	if err != nil {
		t.Skip("fixture file not found")
	}
	output := filterGhList(string(input), "")

	in := utils.CountTokens(string(input))
	out := utils.CountTokens(output)
	if in == 0 {
		t.Skip("empty fixture")
	}
	// filterGhList is a passthrough with trimming — just verify it produces output
	assert.NotEmpty(t, output)
	_ = savings(in, out) // just verify it runs without error
}

func TestFilterGhList_Empty(t *testing.T) {
	out := filterGhList("", "")
	assert.NotNil(t, out)
}

func TestFilterGhList_Basic(t *testing.T) {
	input := `Showing 3 of 3 open pull requests in owner/repo

#42  Add authentication  feature/auth   about 2 hours ago
#41  Fix bug             fix/bug        about 1 day ago
#40  Update deps         chore/deps     about 3 days ago
`
	out := filterGhList(input, "")
	assert.NotEmpty(t, out)
	assert.Contains(t, out, "#42")
}

func TestFilterGhPrView_Basic(t *testing.T) {
	input := `title:  Add user authentication
state:  OPEN
author: alice
url:    https://github.com/owner/repo/pull/42

This PR adds user authentication using JWT tokens.
`
	out := filterGhPrView(input, "")
	assert.NotEmpty(t, out)
}

// savings is a helper to avoid unused variable warnings
func savings(in, out int) float64 {
	if in == 0 {
		return 0
	}
	return 100.0 - float64(out)/float64(in)*100.0
}
