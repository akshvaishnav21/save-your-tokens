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

var pipCmd = &cobra.Command{
	Use:   "pip",
	Short: "Token-optimized pip/uv commands",
}

var pipInstallCmd = &cobra.Command{
	Use:                "install [args...]",
	Short:              "Compact pip install output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPipSubcmd("pip install", "pip", append([]string{"install"}, args...), filterPipInstall)
	},
}

var pipListCmd = &cobra.Command{
	Use:                "list [args...]",
	Short:              "Compact pip list output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPipSubcmd("pip list", "pip", append([]string{"list"}, args...), filterPipList)
	},
}

var pipOutdatedCmd = &cobra.Command{
	Use:                "outdated [args...]",
	Short:              "Compact pip list --outdated output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPipSubcmd("pip outdated", "pip", append([]string{"list", "--outdated"}, args...), filterPipList)
	},
}

func runPipSubcmd(cmdName, binary string, args []string, fn func(string, string) string) error {
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

	exitCode, err := runner.RunWithFilter(cmdName, binary, args, fn)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

// filterPipInstall shows errors or compact success.
func filterPipInstall(stdout, stderr string) string {
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

	if strings.Contains(lower, "error") || strings.Contains(lower, "could not") {
		return combined
	}

	lines := strings.Split(combined, "\n")
	var kept []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		lLow := strings.ToLower(l)
		if strings.HasPrefix(lLow, "successfully installed") ||
			strings.HasPrefix(lLow, "requirement already") ||
			strings.HasPrefix(lLow, "collecting") ||
			strings.Contains(lLow, "installed") {
			if l != "" {
				kept = append(kept, l)
			}
		}
	}
	if len(kept) == 0 {
		return "install ok ✓\n"
	}
	return strings.Join(kept, "\n") + "\n"
}

// filterPipList shows a compact package list.
func filterPipList(stdout, stderr string) string {
	if stderr != "" && stdout == "" {
		return stderr
	}
	combined := utils.StripANSI(stdout)
	lines := strings.Split(combined, "\n")
	var kept []string
	for i, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		// Skip header/separator lines (first 2 lines of pip list)
		if i < 2 && (strings.HasPrefix(l, "Package") || strings.HasPrefix(l, "---")) {
			continue
		}
		kept = append(kept, l)
	}
	return strings.Join(kept, "\n") + "\n"
}

func init() {
	pipCmd.AddCommand(pipInstallCmd)
	pipCmd.AddCommand(pipListCmd)
	pipCmd.AddCommand(pipOutdatedCmd)
	rootCmd.AddCommand(pipCmd)
}
