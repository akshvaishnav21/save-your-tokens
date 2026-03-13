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
	vitestFailRe    = regexp.MustCompile(`(?i)^\s*(FAIL|✗|×)\s`)
	vitestPassRe    = regexp.MustCompile(`(?i)^\s*(PASS|✓|√)\s`)
	vitestSummaryRe = regexp.MustCompile(`(?i)(Tests|Test Files)\s+\d+`)
	vitestDurRe     = regexp.MustCompile(`(?i)Duration\s+[\d.]+\s*s`)
)

var vitestCmd = &cobra.Command{
	Use:                "vitest [args...]",
	Short:              "Filtered vitest output (failures + summary only)",
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

		// Build args: ensure "run" is present for non-watch mode
		runArgs := args
		if len(runArgs) == 0 {
			runArgs = []string{"run"}
		}

		exitCode, err := runner.RunWithFilter("vitest", "vitest", runArgs, filterVitest)
		if err != nil {
			return err
		}
		if exitCode != 0 {
			os.Exit(exitCode)
		}
		return nil
	},
}

// filterVitest strips ANSI, shows FAIL blocks + summary.
func filterVitest(stdout, stderr string) string {
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

	var failBlocks [][]string
	var summaryLines []string
	var durationLine string
	var currentFail []string
	var inFail bool
	var passCount int
	var failCount int

	for _, line := range lines {
		stripped := strings.TrimSpace(line)

		if vitestSummaryRe.MatchString(line) {
			summaryLines = append(summaryLines, stripped)
			inFail = false
			if currentFail != nil {
				failBlocks = append(failBlocks, currentFail)
				currentFail = nil
			}
			continue
		}
		if vitestDurRe.MatchString(line) {
			durationLine = stripped
			continue
		}

		if vitestFailRe.MatchString(line) {
			if currentFail != nil {
				failBlocks = append(failBlocks, currentFail)
			}
			inFail = true
			currentFail = []string{stripped}
			failCount++
			continue
		}
		if vitestPassRe.MatchString(line) {
			passCount++
			if inFail {
				inFail = false
				if currentFail != nil {
					failBlocks = append(failBlocks, currentFail)
					currentFail = nil
				}
			}
			continue
		}

		if inFail {
			if stripped == "" && len(currentFail) > 2 {
				failBlocks = append(failBlocks, currentFail)
				currentFail = nil
				inFail = false
			} else {
				currentFail = append(currentFail, line)
			}
		}
	}
	if currentFail != nil {
		failBlocks = append(failBlocks, currentFail)
	}

	if failCount == 0 && len(failBlocks) == 0 {
		dur := ""
		if durationLine != "" {
			dur = " (" + durationLine + ")"
		}
		return fmt.Sprintf("%d tests passed ✓%s\n", passCount, dur)
	}

	var sb strings.Builder
	for _, block := range failBlocks {
		for _, l := range block {
			sb.WriteString(l)
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}
	for _, l := range summaryLines {
		sb.WriteString(l)
		sb.WriteString("\n")
	}
	if durationLine != "" {
		sb.WriteString(durationLine)
		sb.WriteString("\n")
	}
	return sb.String()
}

func init() {
	rootCmd.AddCommand(vitestCmd)
}
