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

var npmCmd = &cobra.Command{
	Use:   "npm",
	Short: "Token-optimized npm commands",
}

var npmInstallCmd = &cobra.Command{
	Use:                "install [args...]",
	Short:              "Compact npm install output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNpmSubcmd("npm install", append([]string{"install"}, args...), filterNpmInstall)
	},
}

var npmRunCmd = &cobra.Command{
	Use:                "run [args...]",
	Short:              "npm run with compact output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNpmSubcmd("npm run", append([]string{"run"}, args...), filterNpmRun)
	},
}

var npmTestCmd = &cobra.Command{
	Use:                "test [args...]",
	Short:              "npm test with compact output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNpmSubcmd("npm test", append([]string{"test"}, args...), filterNpmRun)
	},
}

func runNpmSubcmd(cmdName string, args []string, fn func(string, string) string) error {
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

	exitCode, err := runner.RunWithFilter(cmdName, "npm", args, fn)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

// filterNpmInstall strips verbose npm install output.
func filterNpmInstall(stdout, stderr string) string {
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
	if strings.Contains(lower, "npm err!") || strings.Contains(lower, "npm error") {
		return combined
	}

	lines := strings.Split(combined, "\n")
	var kept []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		lLow := strings.ToLower(l)
		// Skip verbose lines
		if strings.HasPrefix(l, "npm warn") || strings.HasPrefix(l, "npm notice") {
			continue
		}
		if strings.Contains(lLow, "added") || strings.Contains(lLow, "removed") ||
			strings.Contains(lLow, "packages") || strings.Contains(lLow, "audited") ||
			strings.Contains(lLow, "found") {
			kept = append(kept, l)
		}
	}
	if len(kept) == 0 {
		return "install ok ✓\n"
	}
	return strings.Join(kept, "\n") + "\n"
}

// filterNpmRun passes through npm run/test output, stripping boilerplate.
func filterNpmRun(stdout, stderr string) string {
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
		// Skip npm > script lines and blank npm lines
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
	npmCmd.AddCommand(npmInstallCmd)
	npmCmd.AddCommand(npmRunCmd)
	npmCmd.AddCommand(npmTestCmd)
	rootCmd.AddCommand(npmCmd)
}
