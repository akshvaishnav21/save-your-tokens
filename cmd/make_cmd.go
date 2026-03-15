package cmd

import (
	"os"
	"regexp"
	"strings"

	"github.com/saveyourtokens/syt/internal/config"
	"github.com/saveyourtokens/syt/internal/filter"
	"github.com/saveyourtokens/syt/internal/tee"
	"github.com/saveyourtokens/syt/internal/tracker"
	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/spf13/cobra"
)

var makeCmd = &cobra.Command{
	Use:   "make",
	Short: "Token-optimized make commands",
}

var makeTestCmd = &cobra.Command{
	Use:                "test [args...]",
	Short:              "Compact make test output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMakeSubcmd("make test", append([]string{"test"}, args...), filterMakeTest)
	},
}

var makeBuildCmd = &cobra.Command{
	Use:                "build [args...]",
	Short:              "Compact make build output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMakeSubcmd("make build", append([]string{"build"}, args...), filterMakeBuild)
	},
}

var makeInstallCmd = &cobra.Command{
	Use:                "install [args...]",
	Short:              "Compact make install output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMakeSubcmd("make install", append([]string{"install"}, args...), filterMakeBuild)
	},
}

var makeLintCmd = &cobra.Command{
	Use:                "lint [args...]",
	Short:              "Compact make lint output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runMakeSubcmd("make lint", append([]string{"lint"}, args...), filterMakeBuild)
	},
}

func runMakeSubcmd(cmdName string, args []string, fn func(string, string) string) error {
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

	exitCode, err := runner.RunWithFilter(cmdName, "make", args, fn)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

var makeDirPattern = regexp.MustCompile(`^make\[\d+\]: (Entering|Leaving) directory`)

// stripMakeNoise removes make directory noise from lines.
func stripMakeNoise(lines []string) []string {
	var out []string
	for _, l := range lines {
		if makeDirPattern.MatchString(strings.TrimSpace(l)) {
			continue
		}
		out = append(out, l)
	}
	return out
}

// filterMakeBuild strips make directory lines, keeps errors and real output.
func filterMakeBuild(stdout, stderr string) string {
	combined := stdout
	if stderr != "" {
		if combined != "" {
			combined += "\n" + stderr
		} else {
			combined = stderr
		}
	}
	combined = utils.StripANSI(combined)
	lines := stripMakeNoise(strings.Split(combined, "\n"))
	result := strings.TrimSpace(strings.Join(lines, "\n"))
	if result == "" {
		return "ok ✓\n"
	}
	return result + "\n"
}

// filterMakeTest strips make noise and passing test lines.
func filterMakeTest(stdout, stderr string) string {
	combined := stdout
	if stderr != "" {
		if combined != "" {
			combined += "\n" + stderr
		} else {
			combined = stderr
		}
	}
	combined = utils.StripANSI(combined)
	lines := stripMakeNoise(strings.Split(combined, "\n"))
	// Apply jest-style filtering if output looks like a test runner
	filtered := filterJest(strings.Join(lines, "\n"), "")
	if filtered != strings.Join(lines, "\n")+"\n" {
		return filtered
	}
	result := strings.TrimSpace(strings.Join(lines, "\n"))
	if result == "" {
		return "ok ✓\n"
	}
	return result + "\n"
}

func init() {
	makeCmd.AddCommand(makeTestCmd)
	makeCmd.AddCommand(makeBuildCmd)
	makeCmd.AddCommand(makeInstallCmd)
	makeCmd.AddCommand(makeLintCmd)
	rootCmd.AddCommand(makeCmd)
}
