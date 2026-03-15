package cmd

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/saveyourtokens/syt/internal/integrity"
	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/spf13/cobra"
)

const hookScript = `#!/usr/bin/env bash
set -euo pipefail
command -v jq >/dev/null 2>&1 || exit 0
command -v syt >/dev/null 2>&1 || exit 0
INPUT=$(cat)
TOOL=$(echo "$INPUT" | jq -r '.tool_name // ""')
CMD=$(echo "$INPUT" | jq -r '.tool_input.command // ""')
[[ "$TOOL" != "Bash" ]] && exit 0
[[ -z "$CMD" ]] && exit 0
[[ "$CMD" == syt\ * ]] && exit 0
[[ "$CMD" == *$'\n'* ]] && exit 0
REWRITTEN=$(syt rewrite "$CMD" 2>/dev/null) || exit 0
DESC=$(echo "$INPUT" | jq -r '.tool_input.description // ""')
jq -n --arg cmd "$REWRITTEN" --arg desc "$DESC" \
  '{permissionDecision:"allow",updatedInput:{command:$cmd,description:$desc}}'
if [[ "${SYT_HOOK_AUDIT:-0}" == "1" ]]; then
  echo "$(date -u +%Y-%m-%dT%H:%M:%SZ) REWRITE: $CMD -> $REWRITTEN" \
    >> "${HOME}/.local/share/syt/hook-audit.log"
fi
`

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Install the Claude Code PreToolUse hook",
	RunE:  runInit,
}

func runInit(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir: %w", err)
	}

	// 1. Write hook script (non-Windows only; on Windows we use "syt hook" directly)
	hooksDir := filepath.Join(home, ".claude", "hooks")
	if err := os.MkdirAll(hooksDir, 0755); err != nil {
		return fmt.Errorf("creating hooks dir: %w", err)
	}

	if runtime.GOOS != "windows" {
		hookPath := filepath.Join(hooksDir, "syt-rewrite.sh")
		if err := os.WriteFile(hookPath, []byte(hookScript), 0644); err != nil {
			return fmt.Errorf("writing hook script: %w", err)
		}
		if err := os.Chmod(hookPath, 0755); err != nil {
			return fmt.Errorf("setting hook permissions: %w", err)
		}

		// 2. Store integrity hash
		dataDir := utils.DataDir()
		if err := integrity.Store(dataDir, hookScript); err != nil {
			return fmt.Errorf("storing integrity hash: %w", err)
		}

		// Also write hooks/syt-rewrite.sh in project
		projectHookDir := "hooks"
		_ = os.MkdirAll(projectHookDir, 0755)
		_ = os.WriteFile(filepath.Join(projectHookDir, "syt-rewrite.sh"), []byte(hookScript), 0755)
	}

	// 3. Patch ~/.claude/settings.json
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := patchSettings(settingsPath, "syt hook"); err != nil {
		return fmt.Errorf("patching settings: %w", err)
	}

	// 4. Create default config if not exists
	configDir := utils.ConfigDir()
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	configPath := filepath.Join(configDir, "config.toml")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		defaultConfig := `# SaveYourTokens configuration
[tracking]
history_days = 90

[tee]
enabled = true
mode = "failures"
min_size = 500
max_files = 20
max_file_size = 1048576

[display]
colors = true
ultra_compact = false
`
		if err := os.WriteFile(configPath, []byte(defaultConfig), 0644); err != nil {
			// Non-fatal
			fmt.Fprintf(os.Stderr, "warning: could not write config: %v\n", err)
		}
	}

	// Print success
	fmt.Println("✓ hook registered: syt hook")
	if runtime.GOOS != "windows" {
		h := sha256.Sum256([]byte(hookScript))
		fmt.Printf("✓ syt-rewrite.sh written to %s\n", filepath.Join(hooksDir, "syt-rewrite.sh"))
		fmt.Printf("✓ Integrity hash: %x\n", h[:8])
	}
	fmt.Printf("✓ Config: %s\n", configPath)
	fmt.Println("✓ Claude Code will now automatically optimize bash commands.")
	return nil
}

// patchSettings adds the syt hook to ~/.claude/settings.json.
func patchSettings(settingsPath, hookCmd string) error {
	// Read existing settings
	var settings map[string]interface{}
	if data, err := os.ReadFile(settingsPath); err == nil {
		_ = json.Unmarshal(data, &settings)
	}
	if settings == nil {
		settings = make(map[string]interface{})
	}

	// Build hook entry
	hookEntry := map[string]interface{}{
		"type":    "command",
		"command": hookCmd,
	}
	hookMatcher := map[string]interface{}{
		"matcher": "Bash",
		"hooks":   []interface{}{hookEntry},
	}

	// Get or create hooks.PreToolUse
	hooks, _ := settings["hooks"].(map[string]interface{})
	if hooks == nil {
		hooks = make(map[string]interface{})
		settings["hooks"] = hooks
	}

	preToolUse, _ := hooks["PreToolUse"].([]interface{})

	// Check if already present
	for _, item := range preToolUse {
		if m, ok := item.(map[string]interface{}); ok {
			if m["matcher"] == "Bash" {
				// Already exists, update it
				if subHooks, ok := m["hooks"].([]interface{}); ok {
					for _, sh := range subHooks {
						if shm, ok := sh.(map[string]interface{}); ok {
							if shm["command"] == hookCmd {
								return nil // Already set correctly
							}
						}
					}
					m["hooks"] = []interface{}{hookEntry}
					return writeSettingsAtomic(settingsPath, settings)
				}
			}
		}
	}

	// Add new entry
	hooks["PreToolUse"] = append(preToolUse, hookMatcher)

	return writeSettingsAtomic(settingsPath, settings)
}

// writeSettingsAtomic writes settings.json atomically via tempfile + rename.
func writeSettingsAtomic(path string, settings map[string]interface{}) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("creating settings dir: %w", err)
	}

	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling settings: %w", err)
	}

	tmpPath := path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0644); err != nil {
		return fmt.Errorf("writing temp settings: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		_ = os.Remove(tmpPath)
		return fmt.Errorf("renaming settings: %w", err)
	}

	return nil
}

func init() {
	rootCmd.AddCommand(initCmd)
}
