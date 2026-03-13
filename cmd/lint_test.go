package cmd

import (
	"os"
	"testing"

	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestTokenSavings_Eslint(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/eslint_raw.txt")
	if err != nil {
		t.Skip("fixture file not found")
	}
	output := filterEslint(string(input), "")

	in := utils.CountTokens(string(input))
	out := utils.CountTokens(output)
	if in == 0 {
		t.Skip("empty fixture")
	}
	savings := 100.0 - float64(out)/float64(in)*100.0
	assert.GreaterOrEqualf(t, savings, 40.0,
		"eslint savings %.1f%% < required 40%%", savings)
}

func TestFilterEslint_Empty(t *testing.T) {
	out := filterEslint("", "")
	assert.NotEmpty(t, out)
}

func TestFilterEslint_WithViolations(t *testing.T) {
	input := `
/home/user/project/src/app.js
  42:15  error    'foo' is defined but never used  no-unused-vars
  87:3   warning  Unexpected console statement      no-console

/home/user/project/src/utils.js
  12:1   error    Missing semicolon                 semi

✖ 3 problems (2 errors, 1 warning)
`
	out := filterEslint(input, "")
	assert.NotEmpty(t, out)
	assert.Contains(t, out, "error")
}

func TestFilterEslint_Clean(t *testing.T) {
	out := filterEslint("", "")
	assert.Contains(t, out, "✓")
}
