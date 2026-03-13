package cmd

import (
	"fmt"

	"github.com/saveyourtokens/syt/internal/config"
	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/spf13/cobra"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management",
}

var configShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Print the current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Load()
		fmt.Printf("# SaveYourTokens Configuration\n")
		fmt.Printf("# Config dir: %s\n\n", utils.ConfigDir())
		fmt.Printf("[tracking]\n")
		fmt.Printf("  database_path = %q\n", cfg.Tracking.DatabasePath)
		fmt.Printf("  history_days  = %d\n\n", cfg.Tracking.HistoryDays)
		fmt.Printf("[hooks]\n")
		fmt.Printf("  exclude_commands = %v\n\n", cfg.Hooks.ExcludeCommands)
		fmt.Printf("[tee]\n")
		fmt.Printf("  enabled      = %v\n", cfg.Tee.Enabled)
		fmt.Printf("  mode         = %q\n", cfg.Tee.Mode)
		fmt.Printf("  min_size     = %d\n", cfg.Tee.MinSize)
		fmt.Printf("  max_files    = %d\n", cfg.Tee.MaxFiles)
		fmt.Printf("  max_file_size = %d\n", cfg.Tee.MaxFileSize)
		fmt.Printf("  directory    = %q\n\n", cfg.Tee.Directory)
		fmt.Printf("[display]\n")
		fmt.Printf("  colors        = %v\n", cfg.Display.Colors)
		fmt.Printf("  ultra_compact = %v\n", cfg.Display.UltraCompact)
		return nil
	},
}

func init() {
	configCmd.AddCommand(configShowCmd)
	rootCmd.AddCommand(configCmd)
}
