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

var bunCmd = &cobra.Command{
	Use:   "bun",
	Short: "Token-optimized bun commands",
}

var bunInstallCmd = &cobra.Command{
	Use:                "install [args...]",
	Short:              "Compact bun install output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBunSubcmd("bun install", append([]string{"install"}, args...), filterBunInstall)
	},
}

var bunTestCmd = &cobra.Command{
	Use:                "test [args...]",
	Short:              "Compact bun test output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBunSubcmd("bun test", append([]string{"test"}, args...), filterBunTest)
	},
}

var bunRunCmd = &cobra.Command{
	Use:                "run [args...]",
	Short:              "bun run with tracking",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runBunSubcmd("bun run", append([]string{"run"}, args...), filterBunRun)
	},
}

func runBunSubcmd(cmdName string, args []string, fn func(string, string) string) error {
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

	exitCode, err := runner.RunWithFilter(cmdName, "bun", args, fn)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

// filterBunInstall strips bun install progress, keeps summary.
func filterBunInstall(stdout, stderr string) string {
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
		if strings.Contains(lLow, "installed") || strings.Contains(lLow, "packages") ||
			strings.Contains(lLow, "resolved") || strings.Contains(lLow, "total") ||
			strings.Contains(lLow, "done") || strings.Contains(lLow, "warn") {
			kept = append(kept, trim)
		}
	}
	if len(kept) == 0 {
		return "install ok ✓\n"
	}
	return strings.Join(kept, "\n") + "\n"
}

// filterBunTest keeps failures + summary, strips passing test lines.
func filterBunTest(stdout, stderr string) string {
	combined := stdout
	if stderr != "" {
		if combined != "" {
			combined += "\n" + stderr
		} else {
			combined = stderr
		}
	}
	combined = utils.StripANSI(combined)
	lines := strings.Split(combined, "\n")
	var kept []string
	inFailure := false
	for _, l := range lines {
		lLow := strings.ToLower(strings.TrimSpace(l))
		// Track failure blocks
		if strings.Contains(lLow, "fail") || strings.Contains(lLow, "error") {
			inFailure = true
		}
		// Summary lines always kept
		if strings.HasPrefix(lLow, "tests:") || strings.HasPrefix(lLow, "test suites:") ||
			strings.HasPrefix(lLow, "pass:") || strings.HasPrefix(lLow, "fail:") ||
			strings.Contains(lLow, "passed") || strings.Contains(lLow, "failed") {
			kept = append(kept, l)
			inFailure = false
			continue
		}
		// Skip individual passing test lines (✓ or ✔ prefix)
		trimmed := strings.TrimSpace(l)
		if strings.HasPrefix(trimmed, "✓") || strings.HasPrefix(trimmed, "✔") ||
			strings.HasPrefix(trimmed, "√") {
			continue
		}
		if inFailure {
			kept = append(kept, l)
		}
	}
	if len(kept) == 0 {
		return combined
	}
	return strings.TrimSpace(strings.Join(kept, "\n")) + "\n"
}

// filterBunRun passes through program output unchanged.
func filterBunRun(stdout, stderr string) string {
	combined := stdout
	if stderr != "" {
		if combined != "" {
			combined += "\n" + stderr
		} else {
			combined = stderr
		}
	}
	return combined
}

func init() {
	bunCmd.AddCommand(bunInstallCmd)
	bunCmd.AddCommand(bunTestCmd)
	bunCmd.AddCommand(bunRunCmd)
	rootCmd.AddCommand(bunCmd)
}
