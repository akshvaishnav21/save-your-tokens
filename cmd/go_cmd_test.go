package cmd

import (
	"os"
	"testing"

	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestTokenSavings_GoTest(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/go_test_raw.txt")
	if err != nil {
		t.Skip("fixture file not found")
	}
	output := filterGoTest(string(input), "")

	in := utils.CountTokens(string(input))
	out := utils.CountTokens(output)
	if in == 0 {
		t.Skip("empty fixture")
	}
	savings := 100.0 - float64(out)/float64(in)*100.0
	assert.GreaterOrEqualf(t, savings, 70.0,
		"go test savings %.1f%% < required 70%%", savings)
}

func TestFilterGoTest_Empty(t *testing.T) {
	out := filterGoTest("", "")
	assert.NotEmpty(t, out)
}

func TestFilterGoTest_AllPass(t *testing.T) {
	input := `{"Time":"2024-01-15T10:00:00Z","Action":"run","Package":"github.com/example/myapp","Test":"TestAdd"}
{"Time":"2024-01-15T10:00:00Z","Action":"output","Package":"github.com/example/myapp","Test":"TestAdd","Output":"=== RUN   TestAdd\n"}
{"Time":"2024-01-15T10:00:00Z","Action":"pass","Package":"github.com/example/myapp","Test":"TestAdd","Elapsed":0.001}
{"Time":"2024-01-15T10:00:00Z","Action":"run","Package":"github.com/example/myapp","Test":"TestSubtract"}
{"Time":"2024-01-15T10:00:00Z","Action":"pass","Package":"github.com/example/myapp","Test":"TestSubtract","Elapsed":0.001}
{"Time":"2024-01-15T10:00:00Z","Action":"pass","Package":"github.com/example/myapp","Elapsed":0.005}
`
	out := filterGoTest(input, "")
	assert.Contains(t, out, "passed")
	assert.NotContains(t, out, "FAIL")
}

func TestFilterGoTest_WithFailure(t *testing.T) {
	input := `{"Time":"2024-01-15T10:00:00Z","Action":"run","Package":"github.com/example/myapp","Test":"TestAdd"}
{"Time":"2024-01-15T10:00:00Z","Action":"pass","Package":"github.com/example/myapp","Test":"TestAdd","Elapsed":0.001}
{"Time":"2024-01-15T10:00:00Z","Action":"run","Package":"github.com/example/myapp","Test":"TestBroken"}
{"Time":"2024-01-15T10:00:00Z","Action":"output","Package":"github.com/example/myapp","Test":"TestBroken","Output":"    math_test.go:42: expected 4, got 5\n"}
{"Time":"2024-01-15T10:00:00Z","Action":"fail","Package":"github.com/example/myapp","Test":"TestBroken","Elapsed":0.002}
{"Time":"2024-01-15T10:00:00Z","Action":"fail","Package":"github.com/example/myapp","Elapsed":0.005}
`
	out := filterGoTest(input, "")
	assert.Contains(t, out, "failed")
}
