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

var lsCmd = &cobra.Command{
	Use:                "ls [args...]",
	Short:              "Compact ls output",
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

		lsArgs := append([]string{"-la"}, args...)
		exitCode, err := runner.RunWithFilter("ls", "ls", lsArgs, filterLs)
		if err != nil {
			return err
		}
		if exitCode != 0 {
			os.Exit(exitCode)
		}
		return nil
	},
}

// filterLs produces a compact directory listing.
func filterLs(stdout, stderr string) string {
	if stderr != "" && stdout == "" {
		return stderr
	}
	combined := utils.StripANSI(stdout)
	lines := strings.Split(combined, "\n")
	var kept []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" || l == "total 0" || strings.HasPrefix(l, "total ") {
			continue
		}
		// Skip . and .. entries
		if strings.HasSuffix(l, " .") || strings.HasSuffix(l, " ..") {
			continue
		}
		kept = append(kept, l)
	}
	return strings.Join(kept, "\n") + "\n"
}

func init() {
	rootCmd.AddCommand(lsCmd)
}
