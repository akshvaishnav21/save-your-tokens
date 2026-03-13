package cmd

import (
	"fmt"
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
	cargoTestFailRe    = regexp.MustCompile(`^test\s+\S+\s+\.\.\.\s+FAILED`)
	cargoTestPassRe    = regexp.MustCompile(`^test\s+\S+\s+\.\.\.\s+ok`)
	cargoTestSummaryRe = regexp.MustCompile(`^test result:`)
	cargoCompileRe     = regexp.MustCompile(`^\s*(?:Compiling|Downloaded|Downloading|Updating|Fetching)\s`)
	cargoFinishedRe    = regexp.MustCompile(`^\s*Finished\s`)
	cargoErrorRe       = regexp.MustCompile(`^error`)
	cargoWarningRe     = regexp.MustCompile(`^warning`)
	cargoClippyErrRe   = regexp.MustCompile(`^error(\[E\d+\])?:`)
	cargoClippyWarnRe  = regexp.MustCompile(`^warning(\[.*\])?:`)
)

var cargoCmd = &cobra.Command{
	Use:   "cargo",
	Short: "Token-optimized cargo commands",
}

var cargoTestCmd = &cobra.Command{
	Use:                "test [args...]",
	Short:              "Filtered cargo test (failures + summary only)",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCargoSubcmd("cargo test", append([]string{"test"}, args...), filterCargoTest)
	},
}

var cargoBuildCmd = &cobra.Command{
	Use:                "build [args...]",
	Short:              "Filtered cargo build (errors only)",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCargoSubcmd("cargo build", append([]string{"build"}, args...), filterCargoBuild)
	},
}

var cargoClippyCmd = &cobra.Command{
	Use:                "clippy [args...]",
	Short:              "Filtered cargo clippy (errors + warnings only)",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCargoSubcmd("cargo clippy", append([]string{"clippy"}, args...), filterCargoClippy)
	},
}

var cargoCheckCmd = &cobra.Command{
	Use:                "check [args...]",
	Short:              "Filtered cargo check (errors only)",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCargoSubcmd("cargo check", append([]string{"check"}, args...), filterCargoClippy)
	},
}

var cargoRunCmd = &cobra.Command{
	Use:                "run [args...]",
	Short:              "cargo run with compact build output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runCargoSubcmd("cargo run", append([]string{"run"}, args...), filterCargoRun)
	},
}

func runCargoSubcmd(cmdName string, args []string, fn func(string, string) string) error {
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

	exitCode, err := runner.RunWithFilter(cmdName, "cargo", args, fn)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

// filterCargoTest shows only failed tests + summary.
func filterCargoTest(stdout, stderr string) string {
	combined := stdout
	if stderr != "" {
		if combined != "" {
			combined += "\n" + stderr
		} else {
			combined = stderr
		}
	}
	lines := strings.Split(combined, "\n")

	var failures []string
	var summaryLine string
	var inFailBlock bool
	var failBlock []string
	var passCount int

	for _, line := range lines {
		if cargoTestSummaryRe.MatchString(line) {
			summaryLine = strings.TrimSpace(line)
			inFailBlock = false
			if failBlock != nil {
				failures = append(failures, strings.Join(failBlock, "\n"))
				failBlock = nil
			}
			continue
		}
		if cargoTestFailRe.MatchString(line) {
			inFailBlock = true
			failBlock = []string{strings.TrimSpace(line)}
			continue
		}
		if cargoTestPassRe.MatchString(line) {
			passCount++
			inFailBlock = false
			if failBlock != nil {
				failures = append(failures, strings.Join(failBlock, "\n"))
				failBlock = nil
			}
			continue
		}
		// Compile errors
		if cargoErrorRe.MatchString(line) {
			failures = append(failures, strings.TrimSpace(line))
			inFailBlock = false
			continue
		}
		if inFailBlock {
			failBlock = append(failBlock, line)
		}
	}
	if failBlock != nil {
		failures = append(failures, strings.Join(failBlock, "\n"))
	}

	var sb strings.Builder
	if len(failures) > 0 {
		for _, f := range failures {
			sb.WriteString(f)
			sb.WriteString("\n")
		}
	}
	if summaryLine != "" {
		sb.WriteString(summaryLine)
		sb.WriteString("\n")
	} else if len(failures) == 0 {
		sb.WriteString(fmt.Sprintf("%d passed ✓\n", passCount))
	}
	return sb.String()
}

// filterCargoBuild strips Compiling/Downloaded lines.
func filterCargoBuild(stdout, stderr string) string {
	combined := stdout
	if stderr != "" {
		if combined != "" {
			combined += "\n" + stderr
		} else {
			combined = stderr
		}
	}
	lines := strings.Split(combined, "\n")
	var kept []string
	var finishedLine string

	for _, line := range lines {
		if cargoCompileRe.MatchString(line) {
			continue
		}
		if cargoFinishedRe.MatchString(line) {
			finishedLine = strings.TrimSpace(line)
			continue
		}
		if strings.TrimSpace(line) == "" {
			continue
		}
		kept = append(kept, line)
	}

	if len(kept) == 0 && finishedLine != "" {
		// Extract timing
		if idx := strings.Index(finishedLine, "in "); idx >= 0 {
			timing := finishedLine[idx+3:]
			return fmt.Sprintf("built in %s ✓\n", timing)
		}
		return "build ok ✓\n"
	}

	var sb strings.Builder
	for _, l := range kept {
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	if finishedLine != "" {
		sb.WriteString(finishedLine)
		sb.WriteString("\n")
	}
	return sb.String()
}

// filterCargoClippy shows errors and warnings only.
func filterCargoClippy(stdout, stderr string) string {
	combined := stdout
	if stderr != "" {
		if combined != "" {
			combined += "\n" + stderr
		} else {
			combined = stderr
		}
	}
	lines := strings.Split(combined, "\n")
	var kept []string
	var inBlock bool
	var blockLines []string

	for _, line := range lines {
		if cargoCompileRe.MatchString(line) {
			continue
		}
		if cargoFinishedRe.MatchString(line) {
			if inBlock && len(blockLines) > 0 {
				kept = append(kept, blockLines...)
				blockLines = nil
				inBlock = false
			}
			kept = append(kept, strings.TrimSpace(line))
			continue
		}
		if cargoClippyErrRe.MatchString(line) || cargoClippyWarnRe.MatchString(line) {
			if inBlock && len(blockLines) > 0 {
				kept = append(kept, blockLines...)
			}
			inBlock = true
			blockLines = []string{line}
			continue
		}
		if inBlock {
			if strings.TrimSpace(line) == "" {
				kept = append(kept, blockLines...)
				blockLines = nil
				inBlock = false
			} else {
				blockLines = append(blockLines, line)
			}
		}
	}
	if inBlock && len(blockLines) > 0 {
		kept = append(kept, blockLines...)
	}

	if len(kept) == 0 {
		return "no issues ✓\n"
	}
	return strings.Join(kept, "\n") + "\n"
}

// filterCargoRun strips build noise, keeps program output.
func filterCargoRun(stdout, stderr string) string {
	// Strip stderr build lines, keep stdout program output
	stderrLines := strings.Split(stderr, "\n")
	var filteredStderr []string
	for _, l := range stderrLines {
		if cargoCompileRe.MatchString(l) || cargoFinishedRe.MatchString(l) {
			continue
		}
		if strings.TrimSpace(l) != "" {
			filteredStderr = append(filteredStderr, l)
		}
	}
	result := stdout
	if len(filteredStderr) > 0 {
		if result != "" {
			result += "\n"
		}
		result += strings.Join(filteredStderr, "\n")
	}
	return result
}

func init() {
	cargoCmd.AddCommand(cargoTestCmd)
	cargoCmd.AddCommand(cargoBuildCmd)
	cargoCmd.AddCommand(cargoClippyCmd)
	cargoCmd.AddCommand(cargoCheckCmd)
	cargoCmd.AddCommand(cargoRunCmd)
	rootCmd.AddCommand(cargoCmd)
}
