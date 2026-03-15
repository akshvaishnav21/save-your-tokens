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

var yarnCmd = &cobra.Command{
	Use:   "yarn",
	Short: "Token-optimized yarn commands",
}

var yarnInstallCmd = &cobra.Command{
	Use:                "install [args...]",
	Short:              "Compact yarn install output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runYarnSubcmd("yarn install", append([]string{"install"}, args...), filterYarnInstall)
	},
}

var yarnAddCmd = &cobra.Command{
	Use:                "add [args...]",
	Short:              "Compact yarn add output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runYarnSubcmd("yarn add", append([]string{"add"}, args...), filterYarnInstall)
	},
}

var yarnTestCmd = &cobra.Command{
	Use:                "test [args...]",
	Short:              "Compact yarn test output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runYarnSubcmd("yarn test", append([]string{"test"}, args...), filterYarnTest)
	},
}

func runYarnSubcmd(cmdName string, args []string, fn func(string, string) string) error {
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

	exitCode, err := runner.RunWithFilter(cmdName, "yarn", args, fn)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

// filterYarnInstall strips yarn install/add progress noise.
func filterYarnInstall(stdout, stderr string) string {
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

	// Show errors in full
	if strings.Contains(lower, "error ") || strings.Contains(lower, "err!") {
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
		// Skip progress lines: [1/4], info, verbose spinner chars
		if strings.HasPrefix(trim, "[") && strings.Contains(trim, "/") {
			continue
		}
		if strings.HasPrefix(lLow, "info ") || strings.HasPrefix(lLow, "verbose ") {
			continue
		}
		if strings.HasPrefix(trim, "yarn install v") || strings.HasPrefix(trim, "yarn add v") {
			continue
		}
		// Keep summary lines
		if strings.Contains(lLow, "added") || strings.Contains(lLow, "removed") ||
			strings.Contains(lLow, "done") || strings.Contains(lLow, "success") ||
			strings.Contains(lLow, "warning") || strings.Contains(lLow, "package") {
			kept = append(kept, trim)
		}
	}
	if len(kept) == 0 {
		return "install ok ✓\n"
	}
	return strings.Join(kept, "\n") + "\n"
}

// filterYarnTest strips yarn test boilerplate, keeps failures + summary.
func filterYarnTest(stdout, stderr string) string {
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
	for _, l := range lines {
		// Strip yarn > script banner lines
		if strings.HasPrefix(l, "> ") && strings.Contains(l, "@") {
			continue
		}
		kept = append(kept, l)
	}
	result := strings.TrimSpace(strings.Join(kept, "\n"))
	if result == "" {
		return combined
	}
	return result + "\n"
}

func init() {
	yarnCmd.AddCommand(yarnInstallCmd)
	yarnCmd.AddCommand(yarnAddCmd)
	yarnCmd.AddCommand(yarnTestCmd)
	rootCmd.AddCommand(yarnCmd)
}
