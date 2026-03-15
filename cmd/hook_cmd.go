package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/saveyourtokens/syt/internal/config"
	"github.com/saveyourtokens/syt/internal/registry"
	"github.com/spf13/cobra"
)

type hookInput struct {
	ToolName  string       `json:"tool_name"`
	ToolInput hookToolInput `json:"tool_input"`
}

type hookToolInput struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

type hookOutput struct {
	PermissionDecision string          `json:"permissionDecision"`
	UpdatedInput       hookUpdatedInput `json:"updatedInput"`
}

type hookUpdatedInput struct {
	Command     string `json:"command"`
	Description string `json:"description"`
}

// processHookInput contains the core hook logic and is unit-testable without os.Exit.
// Returns the JSON response bytes, or nil if the hook should be a no-op.
func processHookInput(rawInput []byte, excludeCommands []string) []byte {
	var input hookInput
	if err := json.Unmarshal(rawInput, &input); err != nil {
		return nil
	}

	if input.ToolName != "Bash" {
		return nil
	}

	cmd := strings.TrimSpace(input.ToolInput.Command)
	if cmd == "" || strings.HasPrefix(cmd, "syt ") || strings.Contains(cmd, "\n") {
		return nil
	}

	rewritten := registry.RewriteCommand(cmd, excludeCommands)
	if rewritten == "" {
		return nil
	}

	out := hookOutput{
		PermissionDecision: "allow",
		UpdatedInput: hookUpdatedInput{
			Command:     rewritten,
			Description: input.ToolInput.Description,
		},
	}

	data, err := json.Marshal(out)
	if err != nil {
		return nil
	}
	return data
}

func runHook(cmd *cobra.Command, args []string) error {
	defer func() {
		if r := recover(); r != nil {
			os.Exit(0)
		}
	}()

	rawInput, err := io.ReadAll(os.Stdin)
	if err != nil {
		os.Exit(0)
	}

	cfg := config.Load()
	result := processHookInput(rawInput, cfg.Hooks.ExcludeCommands)
	if result == nil {
		os.Exit(0)
	}

	fmt.Println(string(result))
	return nil
}

var hookCmd = &cobra.Command{
	Use:   "hook",
	Short: "Process a Claude Code PreToolUse hook event (reads JSON from stdin)",
	RunE:  runHook,
}

func init() {
	rootCmd.AddCommand(hookCmd)
}
