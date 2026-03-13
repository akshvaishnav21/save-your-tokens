package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall",
	Short: "Remove the Claude Code PreToolUse hook",
	RunE:  runUninstall,
}

func runUninstall(cmd *cobra.Command, args []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir: %w", err)
	}

	// Remove hook script
	hookPath := filepath.Join(home, ".claude", "hooks", "syt-rewrite.sh")
	if err := os.Remove(hookPath); err != nil && !os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "warning: could not remove hook script: %v\n", err)
	} else {
		fmt.Printf("✓ Removed %s\n", hookPath)
	}

	// Remove hook entry from settings.json
	settingsPath := filepath.Join(home, ".claude", "settings.json")
	if err := removeHookFromSettings(settingsPath, hookPath); err != nil {
		fmt.Fprintf(os.Stderr, "warning: could not update settings: %v\n", err)
	} else {
		fmt.Println("✓ Removed hook from settings.json")
	}

	fmt.Println("✓ syt uninstalled. Claude Code will no longer rewrite commands.")
	return nil
}

func removeHookFromSettings(settingsPath, hookPath string) error {
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("reading settings: %w", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return fmt.Errorf("parsing settings: %w", err)
	}

	hooks, ok := settings["hooks"].(map[string]interface{})
	if !ok {
		return nil
	}

	preToolUse, ok := hooks["PreToolUse"].([]interface{})
	if !ok {
		return nil
	}

	// Filter out our hook
	var newPreToolUse []interface{}
	for _, item := range preToolUse {
		m, ok := item.(map[string]interface{})
		if !ok || m["matcher"] != "Bash" {
			newPreToolUse = append(newPreToolUse, item)
			continue
		}
		// Check if it's our hook
		subHooks, _ := m["hooks"].([]interface{})
		var newSubHooks []interface{}
		for _, sh := range subHooks {
			shm, ok := sh.(map[string]interface{})
			if !ok || shm["command"] != hookPath {
				newSubHooks = append(newSubHooks, sh)
			}
		}
		if len(newSubHooks) > 0 {
			m["hooks"] = newSubHooks
			newPreToolUse = append(newPreToolUse, m)
		}
	}

	hooks["PreToolUse"] = newPreToolUse
	return writeSettingsAtomic(settingsPath, settings)
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}
