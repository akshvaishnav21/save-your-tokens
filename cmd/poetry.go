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

var poetryCmd = &cobra.Command{
	Use:   "poetry",
	Short: "Token-optimized poetry commands",
}

var poetryInstallCmd = &cobra.Command{
	Use:                "install [args...]",
	Short:              "Compact poetry install output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPoetrySubcmd("poetry install", append([]string{"install"}, args...), filterPoetryInstall)
	},
}

var poetryAddCmd = &cobra.Command{
	Use:                "add [args...]",
	Short:              "Compact poetry add output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPoetrySubcmd("poetry add", append([]string{"add"}, args...), filterPoetryInstall)
	},
}

func runPoetrySubcmd(cmdName string, args []string, fn func(string, string) string) error {
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

	exitCode, err := runner.RunWithFilter(cmdName, "poetry", args, fn)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

// filterPoetryInstall strips downloading/progress lines, keeps install summary.
func filterPoetryInstall(stdout, stderr string) string {
	combined := stdout
	if stderr != "" {
		if combined != "" {
			combined += "\n" + stderr
		} else {
			combined = stderr
		}
	}
	combined = utils.StripANSI(combined)
	lower := strings.ToLower(combined)

	if strings.Contains(lower, "error") {
		return combined
	}

	lines := strings.Split(combined, "\n")
	var kept []string
	for _, l := range lines {
		trim := strings.TrimSpace(l)
		if trim == "" {
			continue
		}
		lLow := strings.ToLower(trim)
		// Skip downloading/progress lines
		if strings.HasPrefix(lLow, "downloading") || strings.Contains(lLow, "progress") ||
			strings.HasPrefix(lLow, "  -") {
			continue
		}
		// Keep meaningful lines
		if strings.Contains(lLow, "installing") || strings.Contains(lLow, "updating") ||
			strings.Contains(lLow, "resolving") || strings.Contains(lLow, "warning") ||
			strings.Contains(lLow, "package") || strings.Contains(lLow, "installed") ||
			strings.HasPrefix(lLow, "•") || strings.HasPrefix(trim, "Package operations") {
			kept = append(kept, trim)
		}
	}
	if len(kept) == 0 {
		return "install ok ✓\n"
	}
	return strings.Join(kept, "\n") + "\n"
}

func init() {
	poetryCmd.AddCommand(poetryInstallCmd)
	poetryCmd.AddCommand(poetryAddCmd)
	rootCmd.AddCommand(poetryCmd)
}
