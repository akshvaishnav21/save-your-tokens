package cmd

import (
	"fmt"
	"os"

	"github.com/saveyourtokens/syt/internal/config"
	"github.com/saveyourtokens/syt/internal/registry"
	"github.com/spf13/cobra"
)

var rewriteCmd = &cobra.Command{
	Use:   "rewrite <command>",
	Short: "Rewrite a command to its syt equivalent (used by hook)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		input := args[0]

		// Lazy-load config only for exclusion list (avoid SQLite open)
		cfg := config.Load()

		result := registry.RewriteCommand(input, cfg.Hooks.ExcludeCommands)
		if result == "" {
			os.Exit(1)
		}
		fmt.Print(result)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rewriteCmd)
}
