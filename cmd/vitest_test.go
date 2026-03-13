package cmd

import (
	"os"
	"testing"

	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestTokenSavings_Vitest(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/vitest_raw.txt")
	if err != nil {
		t.Skip("fixture file not found")
	}
	output := filterVitest(string(input), "")

	in := utils.CountTokens(string(input))
	out := utils.CountTokens(output)
	if in == 0 {
		t.Skip("empty fixture")
	}
	savings := 100.0 - float64(out)/float64(in)*100.0
	assert.GreaterOrEqualf(t, savings, 80.0,
		"vitest savings %.1f%% < required 80%%", savings)
}

func TestFilterVitest_Empty(t *testing.T) {
	out := filterVitest("", "")
	assert.NotEmpty(t, out)
}

func TestFilterVitest_AllPass(t *testing.T) {
	input := ` ✓ src/math.test.ts (5)
 ✓ src/utils.test.ts (8)
 ✓ src/api.test.ts (12)

 Test Files  3 passed (3)
      Tests  25 passed (25)
   Start at  10:00:00
   Duration  1.23s
`
	out := filterVitest(input, "")
	assert.Contains(t, out, "passed")
	assert.NotContains(t, out, "FAIL")
}

func TestFilterVitest_WithFailure(t *testing.T) {
	input := ` ✓ src/math.test.ts (5)
 × src/utils.test.ts (1)
   × formatDate > should format correctly
     AssertionError: expected 'Jan 1' to equal 'January 1'

 Test Files  1 failed | 1 passed (2)
      Tests  1 failed | 5 passed (6)
   Duration  0.89s
`
	out := filterVitest(input, "")
	assert.Contains(t, out, "failed")
}
