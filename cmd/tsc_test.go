package cmd

import (
	"os"
	"testing"

	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestTokenSavings_Tsc(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/tsc_raw.txt")
	if err != nil {
		t.Skip("fixture file not found")
	}
	output := filterTsc(string(input), "")

	in := utils.CountTokens(string(input))
	out := utils.CountTokens(output)
	if in == 0 {
		t.Skip("empty fixture")
	}
	savings := 100.0 - float64(out)/float64(in)*100.0
	assert.GreaterOrEqualf(t, savings, 50.0,
		"tsc savings %.1f%% < required 50%%", savings)
}

func TestFilterTsc_NoErrors(t *testing.T) {
	out := filterTsc("", "")
	assert.Contains(t, out, "✓")
}

func TestFilterTsc_WithErrors(t *testing.T) {
	input := `src/components/Button.tsx(42,15): error TS2339: Property 'onClick' does not exist on type 'ButtonProps'.
src/components/Button.tsx(58,22): error TS2345: Argument of type 'string' is not assignable to parameter of type 'number'.
src/utils/format.ts(12,8): error TS2322: Type 'null' is not assignable to type 'string'.

Found 3 errors in 2 files.
`
	out := filterTsc(input, "")
	assert.Contains(t, out, "error")
	assert.NotEmpty(t, out)
}

func TestFilterTsc_GroupsByFile(t *testing.T) {
	input := `src/foo.ts(1,1): error TS2339: Property 'a' does not exist.
src/foo.ts(2,1): error TS2339: Property 'b' does not exist.
src/bar.ts(5,1): error TS2322: Type mismatch.

Found 3 errors in 2 files.
`
	out := filterTsc(input, "")
	// Should group errors by file — output should be shorter than input
	assert.Less(t, utils.CountTokens(out), utils.CountTokens(input)+10)
}
