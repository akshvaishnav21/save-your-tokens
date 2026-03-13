package tee

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
)

var slugRe = regexp.MustCompile(`[^a-z0-9]+`)

// Tee handles saving raw command output for failure recovery.
type Tee struct {
	Enabled     bool
	Mode        string // "failures" or "always"
	MinSize     int
	MaxFiles    int
	MaxFileSize int64
	Directory   string
}

// makeSlug creates a safe filename slug from cmd.
func makeSlug(cmd string) string {
	s := strings.ToLower(cmd)
	s = slugRe.ReplaceAllString(s, "_")
	s = strings.Trim(s, "_")
	if len(s) > 40 {
		s = s[:40]
	}
	if s == "" {
		s = "cmd"
	}
	return s
}

// Save writes raw output to a tee file if conditions are met.
// Returns the saved file path, or "" if not saved.
func (t *Tee) Save(raw string, cmdSlug string, exitCode int) string {
	if !t.Enabled {
		return ""
	}

	// Check mode
	switch t.Mode {
	case "failures":
		if exitCode == 0 {
			return ""
		}
	case "always":
		// always save
	default:
		if exitCode == 0 {
			return ""
		}
	}

	if len(raw) < t.MinSize {
		return ""
	}

	if err := os.MkdirAll(t.Directory, 0755); err != nil {
		return ""
	}

	slug := makeSlug(cmdSlug)
	epoch := time.Now().Unix()
	filename := fmt.Sprintf("%d_%s.log", epoch, slug)
	filePath := filepath.Join(t.Directory, filename)

	content := raw
	if int64(len(content)) > t.MaxFileSize {
		content = content[:t.MaxFileSize]
		content += fmt.Sprintf("\n[truncated at %d bytes]", t.MaxFileSize)
	}

	if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
		return ""
	}

	// Rotate: delete oldest if count > MaxFiles
	t.rotate()

	return filePath
}

// rotate deletes oldest tee files if count exceeds MaxFiles.
func (t *Tee) rotate() {
	entries, err := os.ReadDir(t.Directory)
	if err != nil {
		return
	}

	var logFiles []os.DirEntry
	for _, e := range entries {
		if !e.IsDir() && strings.HasSuffix(e.Name(), ".log") {
			logFiles = append(logFiles, e)
		}
	}

	if len(logFiles) <= t.MaxFiles {
		return
	}

	// Sort by name (epoch prefix ensures chronological order)
	sort.Slice(logFiles, func(i, j int) bool {
		return logFiles[i].Name() < logFiles[j].Name()
	})

	// Delete oldest
	toDelete := len(logFiles) - t.MaxFiles
	for i := 0; i < toDelete; i++ {
		_ = os.Remove(filepath.Join(t.Directory, logFiles[i].Name()))
	}
}

// Hint returns the formatted hint string with ~ for home directory.
func (t *Tee) Hint(filePath string) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(filePath, home) {
		filePath = "~" + filePath[len(home):]
	}
	return fmt.Sprintf("[full output: %s]", filePath)
}
