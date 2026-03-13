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
	nextRouteRe  = regexp.MustCompile(`(?i)(GET|POST|PUT|DELETE|HEAD)\s+/\S*\s+[\d.]+\s*(kB|ms|B)`)
	nextBundleRe = regexp.MustCompile(`(?i)(First Load JS|chunks|Route \(app\)|○|●|λ)`)
	nextErrorRe  = regexp.MustCompile(`(?i)(error|failed|Error)`)
)

var nextCmd = &cobra.Command{
	Use:   "next",
	Short: "Token-optimized Next.js commands",
}

var nextBuildCmd = &cobra.Command{
	Use:                "build [args...]",
	Short:              "Filtered next build output (routes + bundle stats)",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNextSubcmd("next build", append([]string{"build"}, args...), filterNextBuild)
	},
}

var nextDevCmd = &cobra.Command{
	Use:                "dev [args...]",
	Short:              "Filtered next dev (startup confirmation)",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runNextSubcmd("next dev", append([]string{"dev"}, args...), filterNextDev)
	},
}

func runNextSubcmd(cmdName string, args []string, fn func(string, string) string) error {
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

	exitCode, err := runner.RunWithFilter(cmdName, "npx", append([]string{"next"}, args...), fn)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

// filterNextBuild shows route metrics + bundle stats.
func filterNextBuild(stdout, stderr string) string {
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
	var hasError bool

	for _, line := range lines {
		stripped := strings.TrimSpace(line)
		if stripped == "" {
			continue
		}
		if nextErrorRe.MatchString(line) {
			hasError = true
			kept = append(kept, stripped)
			continue
		}
		if nextRouteRe.MatchString(line) || nextBundleRe.MatchString(line) {
			kept = append(kept, stripped)
			continue
		}
		// Keep summary lines
		if strings.Contains(strings.ToLower(stripped), "compiled") ||
			strings.Contains(strings.ToLower(stripped), "build") ||
			strings.Contains(strings.ToLower(stripped), "export") {
			kept = append(kept, stripped)
		}
	}

	if !hasError && len(kept) == 0 {
		return "next build ok ✓\n"
	}
	return strings.Join(kept, "\n") + "\n"
}

// filterNextDev shows startup confirmation.
func filterNextDev(stdout, stderr string) string {
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
	for _, line := range lines {
		stripped := strings.TrimSpace(line)
		if stripped == "" {
			continue
		}
		lower := strings.ToLower(stripped)
		if strings.Contains(lower, "ready") || strings.Contains(lower, "started") ||
			strings.Contains(lower, "localhost") || strings.Contains(lower, "error") {
			kept = append(kept, stripped)
		}
	}
	if len(kept) == 0 {
		return "next dev started ✓\n"
	}
	return strings.Join(kept, "\n") + "\n"
}

func init() {
	nextCmd.AddCommand(nextBuildCmd)
	nextCmd.AddCommand(nextDevCmd)
	rootCmd.AddCommand(nextCmd)
}
