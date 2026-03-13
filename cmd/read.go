package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/saveyourtokens/syt/internal/config"
	"github.com/saveyourtokens/syt/internal/filter"
	"github.com/saveyourtokens/syt/internal/tee"
	"github.com/saveyourtokens/syt/internal/tracker"
	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/spf13/cobra"
)

var readMaxLines int

var readCmd = &cobra.Command{
	Use:   "read [args...]",
	Short: "Read file with line numbers and truncation",
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

		maxLines := readMaxLines
		filterFn := func(stdout, stderr string) string {
			return filterRead(stdout, stderr, maxLines)
		}

		// Build cat args
		catArgs := append([]string{"-n"}, args...)
		exitCode, err := runner.RunWithFilter("read", "cat", catArgs, filterFn)
		if err != nil {
			return err
		}
		if exitCode != 0 {
			os.Exit(exitCode)
		}
		return nil
	},
}

var catCmd = &cobra.Command{
	Use:   "cat [args...]",
	Short: "cat with line numbers and truncation",
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

		maxLines := readMaxLines
		filterFn := func(stdout, stderr string) string {
			return filterRead(stdout, stderr, maxLines)
		}

		catArgs := append([]string{"-n"}, args...)
		exitCode, err := runner.RunWithFilter("read", "cat", catArgs, filterFn)
		if err != nil {
			return err
		}
		if exitCode != 0 {
			os.Exit(exitCode)
		}
		return nil
	},
}

// filterRead adds line numbers and truncates if needed.
func filterRead(stdout, stderr string, maxLines int) string {
	if stderr != "" && stdout == "" {
		return stderr
	}
	lines := strings.Split(stdout, "\n")
	truncated := false
	if maxLines > 0 && len(lines) > maxLines {
		lines = lines[:maxLines]
		truncated = true
	}
	var sb strings.Builder
	for i, l := range lines {
		fmt.Fprintf(&sb, "%4d\t%s\n", i+1, l)
	}
	if truncated {
		fmt.Fprintf(&sb, "... [truncated at %d lines]\n", maxLines)
	}
	return sb.String()
}

func init() {
	readCmd.Flags().IntVar(&readMaxLines, "max-lines", 200, "Maximum lines to display")
	catCmd.Flags().IntVar(&readMaxLines, "max-lines", 200, "Maximum lines to display")
	rootCmd.AddCommand(readCmd)
	rootCmd.AddCommand(catCmd)
}
