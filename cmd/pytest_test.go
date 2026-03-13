package cmd

import (
	"os"
	"testing"

	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestTokenSavings_Pytest(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/pytest_raw.txt")
	if err != nil {
		t.Skip("fixture file not found")
	}
	output := filterPytest(string(input), "")

	in := utils.CountTokens(string(input))
	out := utils.CountTokens(output)
	if in == 0 {
		t.Skip("empty fixture")
	}
	savings := 100.0 - float64(out)/float64(in)*100.0
	assert.GreaterOrEqualf(t, savings, 40.0,
		"pytest savings %.1f%% < required 40%%", savings)
}

func TestFilterPytest_Empty(t *testing.T) {
	out := filterPytest("", "")
	assert.NotEmpty(t, out)
}

func TestFilterPytest_AllPass(t *testing.T) {
	input := `============================= test session starts ==============================
platform linux -- Python 3.11.0, pytest-7.4.0
collected 10 items

tests/test_math.py ..........                                            [100%]

============================== 10 passed in 0.42s ==============================
`
	out := filterPytest(input, "")
	assert.Contains(t, out, "passed")
	assert.NotContains(t, out, "FAILED")
}

func TestFilterPytest_WithFailures(t *testing.T) {
	input := `============================= test session starts ==============================
collected 5 items

tests/test_math.py .F...                                                 [100%]

=================================== FAILURES ===================================
_______________________ test_divide_by_zero ________________________

    def test_divide_by_zero():
>       assert divide(1, 0) == float('inf')
E       ZeroDivisionError: division by zero

tests/test_math.py:15: ZeroDivisionError
=========================== short test summary info ============================
FAILED tests/test_math.py::test_divide_by_zero - ZeroDivisionError: division by zero
========================= 1 failed, 4 passed in 0.23s ==========================
`
	out := filterPytest(input, "")
	assert.Contains(t, out, "FAILED")
}
