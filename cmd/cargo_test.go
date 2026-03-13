package cmd

import (
	"os"
	"testing"

	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestTokenSavings_CargoTest(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/cargo_test_raw.txt")
	if err != nil {
		t.Skip("fixture file not found")
	}
	output := filterCargoTest(string(input), "")

	in := utils.CountTokens(string(input))
	out := utils.CountTokens(output)
	if in == 0 {
		t.Skip("empty fixture")
	}
	savings := 100.0 - float64(out)/float64(in)*100.0
	assert.GreaterOrEqualf(t, savings, 80.0,
		"cargo test savings %.1f%% < required 80%%", savings)
}

func TestTokenSavings_CargoBuild(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/cargo_build_raw.txt")
	if err != nil {
		t.Skip("fixture file not found")
	}
	output := filterCargoBuild(string(input), "")

	in := utils.CountTokens(string(input))
	out := utils.CountTokens(output)
	if in == 0 {
		t.Skip("empty fixture")
	}
	savings := 100.0 - float64(out)/float64(in)*100.0
	assert.GreaterOrEqualf(t, savings, 55.0,
		"cargo build savings %.1f%% < required 55%%", savings)
}

func TestFilterCargoTest_Empty(t *testing.T) {
	out := filterCargoTest("", "")
	assert.NotEmpty(t, out)
}

func TestFilterCargoTest_AllPass(t *testing.T) {
	input := `running 5 tests
test auth::test_login ... ok
test auth::test_logout ... ok
test db::test_connect ... ok
test db::test_query ... ok
test api::test_handler ... ok

test result: ok. 5 passed; 0 failed; 0 ignored; 0 measured; 0 filtered out; finished in 0.45s
`
	out := filterCargoTest(input, "")
	assert.Contains(t, out, "passed")
	assert.NotContains(t, out, "FAILED")
}

func TestFilterCargoBuild_Success(t *testing.T) {
	input := `   Compiling libc v0.2.147
   Compiling proc-macro2 v1.0.67
   Compiling unicode-ident v1.0.12
   Compiling cfg-if v1.0.0
   Compiling myapp v0.1.0 (/home/user/myapp)
    Finished dev [unoptimized + debuginfo] target(s) in 12.34s
`
	out := filterCargoBuild(input, "")
	assert.NotContains(t, out, "Compiling libc")
	assert.NotContains(t, out, "Compiling proc-macro2")
}
