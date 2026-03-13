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

var (
	pnpmTreeCharRe = regexp.MustCompile(`[├└│─\s]+`)
	pnpmPkgRe      = regexp.MustCompile(`([a-zA-Z@][a-zA-Z0-9/_\-.]*@[\w.\-]+)`)
)

var pnpmCmd = &cobra.Command{
	Use:   "pnpm",
	Short: "Token-optimized pnpm commands",
}

var pnpmListCmd = &cobra.Command{
	Use:                "list [args...]",
	Short:              "Compact pnpm list output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPnpmSubcmd("pnpm list", append([]string{"list"}, args...), filterPnpmList)
	},
}

var pnpmInstallCmd = &cobra.Command{
	Use:                "install [args...]",
	Short:              "Compact pnpm install output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPnpmSubcmd("pnpm install", append([]string{"install"}, args...), filterPnpmInstall)
	},
}

var pnpmOutdatedCmd = &cobra.Command{
	Use:                "outdated [args...]",
	Short:              "Compact pnpm outdated output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPnpmSubcmd("pnpm outdated", append([]string{"outdated"}, args...), filterPnpmOutdated)
	},
}

var pnpmAddCmd = &cobra.Command{
	Use:                "add [args...]",
	Short:              "Compact pnpm add output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPnpmSubcmd("pnpm add", append([]string{"add"}, args...), filterPnpmInstall)
	},
}

func runPnpmSubcmd(cmdName string, args []string, fn func(string, string) string) error {
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

	exitCode, err := runner.RunWithFilter(cmdName, "pnpm", args, fn)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

// filterPnpmList strips tree chars and keeps pkg@version lines.
func filterPnpmList(stdout, stderr string) string {
	if stderr != "" && stdout == "" {
		return stderr
	}
	lines := strings.Split(stdout, "\n")
	var kept []string
	seen := make(map[string]bool)

	for _, line := range lines {
		// Strip ANSI
		line = utils.StripANSI(line)
		// Strip tree drawing characters
		cleaned := pnpmTreeCharRe.ReplaceAllString(line, " ")
		cleaned = strings.TrimSpace(cleaned)
		if cleaned == "" {
			continue
		}
		// Extract package@version
		if m := pnpmPkgRe.FindString(cleaned); m != "" {
			if !seen[m] {
				seen[m] = true
				kept = append(kept, m)
			}
		}
	}

	return strings.Join(kept, "\n") + "\n"
}

// filterPnpmInstall shows errors and summary only.
func filterPnpmInstall(stdout, stderr string) string {
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
	if strings.Contains(lower, "error") || strings.Contains(lower, "err!") {
		return combined
	}
	lines := strings.Split(combined, "\n")
	var kept []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		// Keep lines with "added", "removed", "done in", "packages"
		if strings.Contains(strings.ToLower(l), "added") ||
			strings.Contains(strings.ToLower(l), "removed") ||
			strings.Contains(strings.ToLower(l), "done in") ||
			strings.Contains(strings.ToLower(l), "packages") {
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

// filterPnpmOutdated shows the outdated table compactly.
func filterPnpmOutdated(stdout, stderr string) string {
	if stderr != "" && stdout == "" {
		return stderr
	}
	combined := utils.StripANSI(stdout)
	lines := strings.Split(combined, "\n")
	var kept []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" {
			kept = append(kept, l)
		}
	}
	if len(kept) == 0 {
		return "all packages up to date ✓\n"
	}
	return strings.Join(kept, "\n") + "\n"
}

func init() {
	pnpmCmd.AddCommand(pnpmListCmd)
	pnpmCmd.AddCommand(pnpmInstallCmd)
	pnpmCmd.AddCommand(pnpmOutdatedCmd)
	pnpmCmd.AddCommand(pnpmAddCmd)
	rootCmd.AddCommand(pnpmCmd)
}
