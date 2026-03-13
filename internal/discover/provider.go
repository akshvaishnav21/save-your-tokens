package discover

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// SessionProvider is the interface for reading Claude Code session data.
type SessionProvider interface {
	// Sessions returns session file paths matching the criteria.
	Sessions(projectPath string, since time.Time, allProjects bool) ([]string, error)
}

// ClaudeCodeProvider reads sessions from ~/.claude/projects/**/*.jsonl
type ClaudeCodeProvider struct {
	BaseDir string // defaults to ~/.claude/projects
}

// NewClaudeCodeProvider creates a provider with the default base directory.
func NewClaudeCodeProvider() *ClaudeCodeProvider {
	home, _ := os.UserHomeDir()
	return &ClaudeCodeProvider{
		BaseDir: filepath.Join(home, ".claude", "projects"),
	}
}

// Sessions returns JSONL file paths for the given project/time filter.
func (p *ClaudeCodeProvider) Sessions(projectPath string, since time.Time, allProjects bool) ([]string, error) {
	if _, err := os.Stat(p.BaseDir); err != nil {
		return nil, nil // No sessions dir, not an error
	}

	var files []string

	if allProjects || projectPath == "" {
		// Glob all jsonl files
		matches, err := filepath.Glob(filepath.Join(p.BaseDir, "*", "*.jsonl"))
		if err != nil {
			return nil, err
		}
		files = matches
	} else {
		// Encode the project path: replace / (or \) with -
		encoded := encodeProjectPath(projectPath)
		projectDir := filepath.Join(p.BaseDir, encoded)
		matches, err := filepath.Glob(filepath.Join(projectDir, "*.jsonl"))
		if err != nil {
			return nil, err
		}
		files = matches
	}

	// Filter by modification time
	if !since.IsZero() {
		var filtered []string
		for _, f := range files {
			info, err := os.Stat(f)
			if err != nil {
				continue
			}
			if info.ModTime().After(since) {
				filtered = append(filtered, f)
			}
		}
		return filtered, nil
	}

	return files, nil
}

// encodeProjectPath encodes a filesystem path as Claude Code stores it.
func encodeProjectPath(path string) string {
	// Claude Code replaces path separators with -
	path = filepath.ToSlash(path)
	path = strings.ReplaceAll(path, "/", "-")
	path = strings.TrimPrefix(path, "-")
	return path
}

// toolUseEntry represents a single JSONL line from a Claude Code session.
type toolUseEntry struct {
	Type      string `json:"type"`
	Name      string `json:"name"`
	ToolInput struct {
		Command     string `json:"command"`
		Description string `json:"description"`
	} `json:"tool_input"`
	// Also handle nested message format
	Message struct {
		Content []struct {
			Type  string `json:"type"`
			Name  string `json:"name"`
			Input struct {
				Command string `json:"command"`
			} `json:"input"`
		} `json:"content"`
	} `json:"message"`
}

// ExtractBashCommands reads a JSONL session file and returns all Bash commands.
func ExtractBashCommands(filePath string) ([]string, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var commands []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		var entry toolUseEntry
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			continue
		}

		// Direct tool_use format
		if entry.Type == "tool_use" && entry.Name == "Bash" && entry.ToolInput.Command != "" {
			commands = append(commands, entry.ToolInput.Command)
			continue
		}

		// Nested in message.content
		for _, content := range entry.Message.Content {
			if content.Type == "tool_use" && content.Name == "Bash" && content.Input.Command != "" {
				commands = append(commands, content.Input.Command)
			}
		}
	}

	return commands, scanner.Err()
}
