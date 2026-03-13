package cmd

import (
	"os"
	"strings"

	"github.com/saveyourtokens/syt/internal/config"
	"github.com/saveyourtokens/syt/internal/filter"
	"github.com/saveyourtokens/syt/internal/tee"
	"github.com/saveyourtokens/syt/internal/tracker"
	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/spf13/cobra"
)

var ghCmd = &cobra.Command{
	Use:   "gh",
	Short: "Token-optimized GitHub CLI commands",
}

var ghPrCmd = &cobra.Command{
	Use:   "pr",
	Short: "GitHub PR commands",
}

var ghPrViewCmd = &cobra.Command{
	Use:                "view [args...]",
	Short:              "Compact gh pr view",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGhSubcmd("gh pr view", append([]string{"pr", "view"}, args...), filterGhPrView)
	},
}

var ghPrListCmd = &cobra.Command{
	Use:                "list [args...]",
	Short:              "Compact gh pr list",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGhSubcmd("gh pr list", append([]string{"pr", "list"}, args...), filterGhList)
	},
}

var ghIssueCmd = &cobra.Command{
	Use:   "issue",
	Short: "GitHub issue commands",
}

var ghIssueViewCmd = &cobra.Command{
	Use:                "view [args...]",
	Short:              "Compact gh issue view",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGhSubcmd("gh issue view", append([]string{"issue", "view"}, args...), filterGhIssueView)
	},
}

var ghIssueListCmd = &cobra.Command{
	Use:                "list [args...]",
	Short:              "Compact gh issue list",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGhSubcmd("gh issue list", append([]string{"issue", "list"}, args...), filterGhList)
	},
}

var ghRunCmd = &cobra.Command{
	Use:   "run",
	Short: "GitHub Actions run commands",
}

var ghRunViewCmd = &cobra.Command{
	Use:                "view [args...]",
	Short:              "Compact gh run view",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGhSubcmd("gh run view", append([]string{"run", "view"}, args...), filterGhRunView)
	},
}

var ghRunListCmd = &cobra.Command{
	Use:                "list [args...]",
	Short:              "Compact gh run list",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGhSubcmd("gh run list", append([]string{"run", "list"}, args...), filterGhList)
	},
}

func runGhSubcmd(cmdName string, args []string, fn func(string, string) string) error {
	cfg := config.Load()
	var t *tracker.Tracker
	dbPath := cfg.Tracking.DatabasePath
	if dbPath == "" {
		dbPath = utils.DataDir() + "/syt.db"
	}
	if tr, err := tracker.NewTracker(dbPath); err == nil {
		t = tr
		defer t.Close()
	}

	teeCfg := cfg.Tee
	te := &tee.Tee{
		Enabled:     teeCfg.Enabled,
		Mode:        teeCfg.Mode,
		MinSize:     teeCfg.MinSize,
		MaxFiles:    teeCfg.MaxFiles,
		MaxFileSize: teeCfg.MaxFileSize,
		Directory:   teeCfg.Directory,
	}

	runner := &filter.Runner{
		Verbose: verboseCount,
		Tracker: t,
		Tee:     te,
	}

	exitCode, err := runner.RunWithFilter(cmdName, "gh", args, fn)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

// filterGhPrView compacts PR view output.
func filterGhPrView(stdout, stderr string) string {
	if stderr != "" && stdout == "" {
		return stderr
	}
	combined := utils.StripANSI(stdout)
	lines := strings.Split(combined, "\n")
	var kept []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		// Keep: title, number, state, author, labels, url
		lower := strings.ToLower(l)
		if strings.HasPrefix(lower, "title:") || strings.HasPrefix(lower, "state:") ||
			strings.HasPrefix(lower, "author:") || strings.HasPrefix(lower, "labels:") ||
			strings.HasPrefix(lower, "url:") || strings.HasPrefix(lower, "number:") ||
			strings.HasPrefix(lower, "draft:") || strings.HasPrefix(lower, "#") {
			kept = append(kept, l)
		}
	}
	if len(kept) == 0 {
		return combined
	}
	return strings.Join(kept, "\n") + "\n"
}

// filterGhIssueView compacts issue view output.
func filterGhIssueView(stdout, stderr string) string {
	return filterGhPrView(stdout, stderr)
}

// filterGhList compacts list output (PRs, issues, runs).
func filterGhList(stdout, stderr string) string {
	if stderr != "" && stdout == "" {
		return stderr
	}
	combined := utils.StripANSI(stdout)
	lines := strings.Split(combined, "\n")
	var kept []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			kept = append(kept, l)
		}
	}
	return strings.Join(kept, "\n") + "\n"
}

// filterGhRunView compacts run view output, showing status + failures.
func filterGhRunView(stdout, stderr string) string {
	if stderr != "" && stdout == "" {
		return stderr
	}
	combined := utils.StripANSI(stdout)
	lines := strings.Split(combined, "\n")
	var kept []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		lower := strings.ToLower(l)
		if strings.Contains(lower, "status") || strings.Contains(lower, "conclusion") ||
			strings.Contains(lower, "failed") || strings.Contains(lower, "success") ||
			strings.Contains(lower, "title") || strings.Contains(lower, "workflow") ||
			strings.Contains(lower, "error") {
			kept = append(kept, l)
		}
	}
	if len(kept) == 0 {
		return combined
	}
	return strings.Join(kept, "\n") + "\n"
}

func init() {
	ghPrCmd.AddCommand(ghPrViewCmd)
	ghPrCmd.AddCommand(ghPrListCmd)
	ghIssueCmd.AddCommand(ghIssueViewCmd)
	ghIssueCmd.AddCommand(ghIssueListCmd)
	ghRunCmd.AddCommand(ghRunViewCmd)
	ghRunCmd.AddCommand(ghRunListCmd)
	ghCmd.AddCommand(ghPrCmd)
	ghCmd.AddCommand(ghIssueCmd)
	ghCmd.AddCommand(ghRunCmd)
	rootCmd.AddCommand(ghCmd)
}
