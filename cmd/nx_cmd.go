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

var nxCmd = &cobra.Command{
	Use:   "nx",
	Short: "Token-optimized nx commands",
}

var nxTestCmd = &cobra.Command{
	Use:                "test [args...]",
	Short:              "Compact nx test output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNxSubcmd("nx test", append([]string{"test"}, args...), filterNxTest)
	},
}

var nxBuildCmd = &cobra.Command{
	Use:                "build [args...]",
	Short:              "Compact nx build output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNxSubcmd("nx build", append([]string{"build"}, args...), filterNxBuild)
	},
}

func runNxSubcmd(cmdName string, args []string, fn func(string, string) string) error {
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

	exitCode, err := runner.RunWithFilter(cmdName, "nx", args, fn)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

// filterNxTest keeps NX status lines, failures, and summary.
func filterNxTest(stdout, stderr string) string {
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
		trimmed := strings.TrimSpace(l)
		lLow := strings.ToLower(trimmed)

		// NX status banner lines
		if strings.Contains(trimmed, "NX") || strings.HasPrefix(trimmed, ">") {
			kept = append(kept, l)
			continue
		}
		// FAIL/PASS file lines
		if strings.HasPrefix(trimmed, "FAIL ") || strings.HasPrefix(trimmed, "PASS ") {
			kept = append(kept, l)
			inFailure = strings.HasPrefix(trimmed, "FAIL ")
			continue
		}
		// Summary lines
		if strings.HasPrefix(lLow, "tests:") || strings.HasPrefix(lLow, "test suites:") ||
			strings.HasPrefix(lLow, "passed") || strings.HasPrefix(lLow, "failed") {
			kept = append(kept, l)
			inFailure = false
			continue
		}
		// Failure details
		if strings.HasPrefix(trimmed, "●") {
			inFailure = true
			kept = append(kept, l)
			continue
		}
		// Skip passing test lines
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

// filterNxBuild keeps NX status lines and errors.
func filterNxBuild(stdout, stderr string) string {
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
		trimmed := strings.TrimSpace(l)
		lLow := strings.ToLower(trimmed)
		if trimmed == "" {
			continue
		}
		if strings.Contains(trimmed, "NX") || strings.HasPrefix(trimmed, ">") ||
			strings.Contains(lLow, "error") || strings.Contains(lLow, "warning") ||
			strings.Contains(lLow, "successfully") || strings.Contains(lLow, "failed") {
			kept = append(kept, l)
		}
	}

	if len(kept) == 0 {
		return combined
	}
	return strings.TrimSpace(strings.Join(kept, "\n")) + "\n"
}

func init() {
	nxCmd.AddCommand(nxTestCmd)
	nxCmd.AddCommand(nxBuildCmd)
	rootCmd.AddCommand(nxCmd)
}
