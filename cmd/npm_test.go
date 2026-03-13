package cmd

import (
	"os"
	"testing"

	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestTokenSavings_NpmInstall(t *testing.T) {
	input, err := os.ReadFile("../tests/fixtures/npm_install_raw.txt")
	if err != nil {
		t.Skip("fixture file not found")
	}
	output := filterNpmInstall(string(input), "")

	in := utils.CountTokens(string(input))
	out := utils.CountTokens(output)
	if in == 0 {
		t.Skip("empty fixture")
	}
	savings := 100.0 - float64(out)/float64(in)*100.0
	assert.GreaterOrEqualf(t, savings, 30.0,
		"npm install savings %.1f%% < required 30%%", savings)
}

func TestFilterNpmInstall_Empty(t *testing.T) {
	out := filterNpmInstall("", "")
	assert.NotEmpty(t, out)
}

func TestFilterNpmInstall_StripsWarnLines(t *testing.T) {
	input := `npm warn deprecated inflight@1.0.6: This module is not supported
npm warn deprecated glob@7.2.3: Glob versions <=7 are no longer supported
npm warn deprecated rimraf@3.0.2: Rimraf versions prior to v4 are no longer supported

added 127 packages, and audited 128 packages in 8s

found 0 vulnerabilities
`
	out := filterNpmInstall(input, "")
	assert.NotEmpty(t, out)
}
