package cmd

import (
	"os"
	"testing"

	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestTokenSavings_PnpmList(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/pnpm_list_raw.txt")
	if err != nil {
		t.Skip("fixture file not found")
	}
	output := filterPnpmList(string(input), "")

	in := utils.CountTokens(string(input))
	out := utils.CountTokens(output)
	if in == 0 {
		t.Skip("empty fixture")
	}
	savings := 100.0 - float64(out)/float64(in)*100.0
	assert.GreaterOrEqualf(t, savings, 30.0,
		"pnpm list savings %.1f%% < required 30%%", savings)
}

func TestFilterPnpmList_Empty(t *testing.T) {
	out := filterPnpmList("", "")
	// empty or minimal output is acceptable
	assert.NotNil(t, out)
}

func TestFilterPnpmList_StripsTreeChars(t *testing.T) {
	input := `my-project@1.0.0 /home/user/project
├─┬ react@18.2.0
│ ├── loose-envify@1.4.0
│ └── scheduler@0.23.0
└─┬ typescript@5.2.2
  └── (empty)
`
	out := filterPnpmList(input, "")
	assert.NotContains(t, out, "├─")
	assert.NotContains(t, out, "│")
	assert.NotContains(t, out, "└─")
}
