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
	playwrightFailRe    = regexp.MustCompile(`(?i)^\s+\d+\)\s+`)
	playwrightPassRe    = regexp.MustCompile(`(?i)^\s+✓|passed`)
	playwrightSummaryRe = regexp.MustCompile(`(?i)(\d+\s+passed|\d+\s+failed)`)
)

var playwrightCmd = &cobra.Command{
	Use:                "playwright [args...]",
	Short:              "Filtered playwright test output (failures + summary only)",
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

		exitCode, err := runner.RunWithFilter("playwright", "npx", append([]string{"playwright", "test"}, args...), filterPlaywright)
		if err != nil {
			return err
		}
		if exitCode != 0 {
			os.Exit(exitCode)
		}
		return nil
	},
}

// filterPlaywright shows failures grouped by suite + summary.
func filterPlaywright(stdout, stderr string) string {
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
	var currentFail []string
	var inFail bool
	var passCount int

	for _, line := range lines {
		stripped := strings.TrimSpace(line)

		if playwrightSummaryRe.MatchString(line) &&
			(strings.Contains(line, "passed") || strings.Contains(line, "failed")) {
			summaryLines = append(summaryLines, stripped)
			if inFail && currentFail != nil {
				failBlocks = append(failBlocks, currentFail)
				currentFail = nil
				inFail = false
			}
			continue
		}

		if playwrightFailRe.MatchString(line) {
			if currentFail != nil {
				failBlocks = append(failBlocks, currentFail)
			}
			inFail = true
			currentFail = []string{stripped}
			continue
		}

		if strings.Contains(line, "✓") || strings.Contains(line, "passed") {
			passCount++
			continue
		}

		if inFail {
			if stripped == "" && len(currentFail) > 3 {
				failBlocks = append(failBlocks, currentFail)
				currentFail = nil
				inFail = false
			} else {
				currentFail = append(currentFail, stripped)
			}
		}
	}
	if currentFail != nil {
		failBlocks = append(failBlocks, currentFail)
	}

	if len(failBlocks) == 0 {
		return fmt.Sprintf("%d tests passed ✓\n", passCount)
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
	return sb.String()
}

func init() {
	rootCmd.AddCommand(playwrightCmd)
}
