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

var jestCmd = &cobra.Command{
	Use:                "jest [args...]",
	Short:              "Compact jest output (failures + summary)",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
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

		exitCode, err := runner.RunWithFilter("jest", "jest", args, filterJest)
		if err != nil {
			return err
		}
		if exitCode != 0 {
			os.Exit(exitCode)
		}
		return nil
	},
}

// filterJest keeps FAIL/PASS file lines, failure details, and summary.
// Strips individual passing test lines (✓ test name).
func filterJest(stdout, stderr string) string {
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
	inFailureBlock := false

	for _, l := range lines {
		trimmed := strings.TrimSpace(l)
		lLow := strings.ToLower(trimmed)

		// Always keep summary lines
		if strings.HasPrefix(lLow, "tests:") || strings.HasPrefix(lLow, "test suites:") ||
			strings.HasPrefix(lLow, "snapshots:") || strings.HasPrefix(lLow, "time:") ||
			strings.HasPrefix(lLow, "ran all") {
			kept = append(kept, l)
			inFailureBlock = false
			continue
		}

		// FAIL/PASS file-level lines
		if strings.HasPrefix(trimmed, "FAIL ") || strings.HasPrefix(trimmed, "PASS ") {
			kept = append(kept, l)
			inFailureBlock = strings.HasPrefix(trimmed, "FAIL ")
			continue
		}

		// Failure detail blocks (● Test name)
		if strings.HasPrefix(trimmed, "●") {
			inFailureBlock = true
			kept = append(kept, l)
			continue
		}

		// Skip individual passing test lines
		if strings.HasPrefix(trimmed, "✓") || strings.HasPrefix(trimmed, "✔") ||
			strings.HasPrefix(trimmed, "√") || strings.HasPrefix(trimmed, "○") {
			continue
		}

		if inFailureBlock {
			kept = append(kept, l)
		}
	}

	if len(kept) == 0 {
		return combined
	}
	return strings.TrimSpace(strings.Join(kept, "\n")) + "\n"
}

func init() {
	rootCmd.AddCommand(jestCmd)
}
