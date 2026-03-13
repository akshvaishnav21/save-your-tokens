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
	pytestCollectRe = regexp.MustCompile(`(?i)^(collecting|collected)`)
	pytestFailRe    = regexp.MustCompile(`^FAILED\s+`)
	pytestPassRe    = regexp.MustCompile(`^PASSED\s+`)
	pytestSepRe     = regexp.MustCompile(`^={5,}|^-{5,}`)
	pytestSummaryRe = regexp.MustCompile(`(?i)(\d+\s+passed|\d+\s+failed|\d+\s+error)`)
	pytestFailHeadRe = regexp.MustCompile(`^_+\s+(.+)\s+_+$`)
)

var pytestCmd = &cobra.Command{
	Use:                "pytest [args...]",
	Short:              "Filtered pytest output (failures + summary only)",
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

		exitCode, err := runner.RunWithFilter("pytest", "pytest", args, filterPytest)
		if err != nil {
			return err
		}
		if exitCode != 0 {
			os.Exit(exitCode)
		}
		return nil
	},
}

type pytestState int

const (
	pytestStateCollecting pytestState = iota
	pytestStateRunning
	pytestStateFailureDetail
	pytestStateSummary
)

// filterPytest uses a state machine to extract failures + summary.
func filterPytest(stdout, stderr string) string {
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
	state := pytestStateCollecting
	var failureBlocks [][]string
	var summaryLines []string
	var currentBlock []string
	var passCount, failCount int

	for _, line := range lines {
		switch state {
		case pytestStateCollecting:
			if pytestCollectRe.MatchString(line) {
				continue
			}
			// Switch to running once we see test markers
			if pytestSepRe.MatchString(line) && strings.Contains(strings.ToLower(line), "test session") {
				state = pytestStateRunning
				continue
			}
			if pytestSepRe.MatchString(line) {
				state = pytestStateRunning
			}

		case pytestStateRunning:
			if pytestCollectRe.MatchString(line) {
				continue
			}
			if pytestFailRe.MatchString(line) {
				failCount++
				continue
			}
			if pytestPassRe.MatchString(line) {
				passCount++
				continue
			}
			// Detect FAILURES section
			if strings.Contains(strings.ToUpper(line), "FAILURES") && pytestSepRe.MatchString(line) {
				state = pytestStateFailureDetail
				continue
			}
			// Summary section
			if pytestSummaryRe.MatchString(line) {
				summaryLines = append(summaryLines, strings.TrimSpace(line))
				state = pytestStateSummary
				continue
			}

		case pytestStateFailureDetail:
			// Detect start of new failure block
			if pytestFailHeadRe.MatchString(line) {
				if len(currentBlock) > 0 {
					failureBlocks = append(failureBlocks, currentBlock)
				}
				currentBlock = []string{strings.TrimSpace(line)}
				continue
			}
			// Detect short test summary
			if strings.Contains(strings.ToUpper(line), "SHORT TEST SUMMARY") ||
				(pytestSummaryRe.MatchString(line)) {
				if len(currentBlock) > 0 {
					failureBlocks = append(failureBlocks, currentBlock)
					currentBlock = nil
				}
				summaryLines = append(summaryLines, strings.TrimSpace(line))
				state = pytestStateSummary
				continue
			}
			if pytestSepRe.MatchString(line) {
				if strings.Contains(strings.ToUpper(line), "PASS") ||
					strings.Contains(strings.ToUpper(line), "FAIL") {
					if len(currentBlock) > 0 {
						failureBlocks = append(failureBlocks, currentBlock)
						currentBlock = nil
					}
					summaryLines = append(summaryLines, strings.TrimSpace(line))
					state = pytestStateSummary
				}
				continue
			}
			currentBlock = append(currentBlock, line)

		case pytestStateSummary:
			if strings.TrimSpace(line) != "" {
				summaryLines = append(summaryLines, strings.TrimSpace(line))
			}
		}
	}
	if len(currentBlock) > 0 {
		failureBlocks = append(failureBlocks, currentBlock)
	}

	if len(failureBlocks) == 0 && len(summaryLines) == 0 {
		if failCount == 0 {
			return fmt.Sprintf("%d passed ✓\n", passCount)
		}
	}

	var sb strings.Builder
	for _, block := range failureBlocks {
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
	if sb.Len() == 0 {
		return fmt.Sprintf("%d passed ✓\n", passCount)
	}
	return sb.String()
}

func init() {
	rootCmd.AddCommand(pytestCmd)
}
