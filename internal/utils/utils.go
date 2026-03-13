package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"
)

var ansiRe = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes all ANSI escape sequences from s.
func StripANSI(s string) string {
	return ansiRe.ReplaceAllString(s, "")
}

// Truncate returns s truncated to max runes with "..." suffix if truncated.
func Truncate(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	if max <= 3 {
		return string(runes[:max])
	}
	return string(runes[:max-3]) + "..."
}

// CountTokens approximates token count using whitespace splitting.
func CountTokens(s string) int {
	return len(strings.Fields(s))
}

// FormatTokens formats n as "1.2M", "59.2K", or "694".
func FormatTokens(n int) string {
	switch {
	case n >= 1_000_000:
		return fmt.Sprintf("%.1fM", float64(n)/1_000_000)
	case n >= 1_000:
		return fmt.Sprintf("%.1fK", float64(n)/1_000)
	default:
		return fmt.Sprintf("%d", n)
	}
}

// FormatSavingsPct formats savings percentage as "87.3%".
func FormatSavingsPct(pct float64) string {
	return fmt.Sprintf("%.1f%%", pct)
}

// DataDir returns the platform data directory.
func DataDir() string {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, _ := os.UserHomeDir()
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "syt")
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "syt")
	default:
		xdg := os.Getenv("XDG_DATA_HOME")
		if xdg != "" {
			return filepath.Join(xdg, "syt")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".local", "share", "syt")
	}
}

// ConfigDir returns the platform config directory.
func ConfigDir() string {
	switch runtime.GOOS {
	case "windows":
		appData := os.Getenv("APPDATA")
		if appData == "" {
			home, _ := os.UserHomeDir()
			appData = filepath.Join(home, "AppData", "Roaming")
		}
		return filepath.Join(appData, "syt")
	case "darwin":
		home, _ := os.UserHomeDir()
		return filepath.Join(home, "Library", "Application Support", "syt")
	default:
		xdg := os.Getenv("XDG_CONFIG_HOME")
		if xdg != "" {
			return filepath.Join(xdg, "syt")
		}
		home, _ := os.UserHomeDir()
		return filepath.Join(home, ".config", "syt")
	}
}
