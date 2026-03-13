package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStripANSI(t *testing.T) {
	assert.Equal(t, "hello world", StripANSI("\x1b[32mhello\x1b[0m world"))
	assert.Equal(t, "plain", StripANSI("plain"))
	assert.Equal(t, "", StripANSI(""))
	assert.Equal(t, "test", StripANSI("\x1b[1;32mtest\x1b[0m"))
}

func TestTruncate(t *testing.T) {
	assert.Equal(t, "hello", Truncate("hello", 10))
	assert.Equal(t, "hel...", Truncate("hello world", 6))
	assert.Equal(t, "hel", Truncate("hello", 3))
	assert.Equal(t, "", Truncate("", 10))
	// Unicode
	assert.Equal(t, "héllo...", Truncate("héllo world", 8))
}

func TestCountTokens(t *testing.T) {
	assert.Equal(t, 3, CountTokens("hello world foo"))
	assert.Equal(t, 0, CountTokens(""))
	assert.Equal(t, 0, CountTokens("   "))
	assert.Equal(t, 1, CountTokens("single"))
}

func TestFormatTokens(t *testing.T) {
	assert.Equal(t, "694", FormatTokens(694))
	assert.Equal(t, "59.2K", FormatTokens(59200))
	assert.Equal(t, "1.2M", FormatTokens(1200000))
	assert.Equal(t, "1.0K", FormatTokens(1000))
}

func TestFormatSavingsPct(t *testing.T) {
	assert.Equal(t, "87.3%", FormatSavingsPct(87.3))
	assert.Equal(t, "100.0%", FormatSavingsPct(100.0))
	assert.Equal(t, "0.0%", FormatSavingsPct(0.0))
}

func TestDataDir(t *testing.T) {
	dir := DataDir()
	assert.NotEmpty(t, dir)
	assert.Contains(t, dir, "syt")
}

func TestConfigDir(t *testing.T) {
	dir := ConfigDir()
	assert.NotEmpty(t, dir)
	assert.Contains(t, dir, "syt")
}
