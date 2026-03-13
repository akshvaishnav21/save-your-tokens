package cmd

import (
	"os"
	"testing"

	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestTokenSavings_Playwright(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/playwright_raw.txt")
	if err != nil {
		t.Skip("fixture file not found")
	}
	output := filterPlaywright(string(input), "")

	in := utils.CountTokens(string(input))
	out := utils.CountTokens(output)
	if in == 0 {
		t.Skip("empty fixture")
	}
	savings := 100.0 - float64(out)/float64(in)*100.0
	assert.GreaterOrEqualf(t, savings, 70.0,
		"playwright savings %.1f%% < required 70%%", savings)
}

func TestFilterPlaywright_Empty(t *testing.T) {
	out := filterPlaywright("", "")
	assert.NotEmpty(t, out)
}

func TestFilterPlaywright_AllPass(t *testing.T) {
	input := `Running 5 tests using 1 worker

  ✓  1 [chromium] › tests/login.spec.ts:5:3 › Login › should login successfully (1.2s)
  ✓  2 [chromium] › tests/login.spec.ts:15:3 › Login › should show error on bad password (0.8s)
  ✓  3 [chromium] › tests/nav.spec.ts:5:3 › Navigation › should navigate to home (0.5s)
  ✓  4 [chromium] › tests/nav.spec.ts:12:3 › Navigation › should navigate to profile (0.6s)
  ✓  5 [chromium] › tests/api.spec.ts:8:3 › API › should fetch user data (1.1s)

  5 passed (5.2s)
`
	out := filterPlaywright(input, "")
	assert.Contains(t, out, "passed")
	assert.NotContains(t, out, "failed")
}

func TestFilterPlaywright_WithFailures(t *testing.T) {
	input := `Running 3 tests using 1 worker

  ✓  1 [chromium] › tests/login.spec.ts:5:3 › Login › should login successfully (1.2s)
  ✗  2 [chromium] › tests/login.spec.ts:25:3 › Login › should handle timeout (30.0s)
  ✓  3 [chromium] › tests/nav.spec.ts:5:3 › Navigation › should navigate (0.5s)

  1) [chromium] › tests/login.spec.ts:25:3 › Login › should handle timeout

    Error: Timeout 30000ms exceeded.
    Call log:
      - waiting for selector '#submit-btn'

  1 failed, 2 passed (31.7s)
`
	out := filterPlaywright(input, "")
	assert.Contains(t, out, "failed")
}
