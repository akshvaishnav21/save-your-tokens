package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/saveyourtokens/syt/internal/discover"
	"github.com/spf13/cobra"
)

var (
	discoverSince  int
	discoverAll    bool
	discoverFormat string
)

var discoverCmd = &cobra.Command{
	Use:   "discover",
	Short: "Scan Claude Code sessions for optimization opportunities",
	RunE:  runDiscover,
}

func runDiscover(cmd *cobra.Command, args []string) error {
	provider := discover.NewClaudeCodeProvider()

	wd, err := os.Getwd()
	if err != nil {
		wd = ""
	}

	since := time.Now().AddDate(0, 0, -discoverSince)

	opts := discover.Options{
		ProjectPath: wd,
		Since:       since,
		AllProjects: discoverAll,
	}

	result, err := discover.Analyze(provider, opts)
	if err != nil {
		return fmt.Errorf("analyzing sessions: %w", err)
	}

	if discoverFormat == "json" {
		out, err := discover.FormatJSON(result)
		if err != nil {
			return fmt.Errorf("formatting json: %w", err)
		}
		fmt.Println(out)
		return nil
	}

	fmt.Print(discover.FormatText(result))
	return nil
}

func init() {
	discoverCmd.Flags().IntVar(&discoverSince, "since", 30, "Days to look back")
	discoverCmd.Flags().BoolVar(&discoverAll, "all", false, "Scan all projects")
	discoverCmd.Flags().StringVar(&discoverFormat, "format", "", "Output format (json)")
	rootCmd.AddCommand(discoverCmd)
}
