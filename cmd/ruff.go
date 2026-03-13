package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/saveyourtokens/syt/internal/config"
	"github.com/saveyourtokens/syt/internal/filter"
	"github.com/saveyourtokens/syt/internal/tee"
	"github.com/saveyourtokens/syt/internal/tracker"
	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/spf13/cobra"
)

var ruffCmd = &cobra.Command{
	Use:   "ruff",
	Short: "Token-optimized ruff commands",
}

var ruffCheckCmd = &cobra.Command{
	Use:                "check [args...]",
	Short:              "Filtered ruff check (grouped violations)",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRuffSubcmd("ruff check", append([]string{"check"}, args...), filterRuffCheck)
	},
}

var ruffFormatCmd = &cobra.Command{
	Use:                "format [args...]",
	Short:              "Filtered ruff format output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRuffSubcmd("ruff format", append([]string{"format"}, args...), filterRuffFormat)
	},
}

func runRuffSubcmd(cmdName string, args []string, fn func(string, string) string) error {
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

	exitCode, err := runner.RunWithFilter(cmdName, "ruff", args, fn)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

// ruffViolation represents a parsed ruff violation.
type ruffViolation struct {
	File    string
	Code    string
	Message string
}

// filterRuffCheck groups ruff violations by rule code.
func filterRuffCheck(stdout, stderr string) string {
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
	codeCounts := make(map[string]int)
	codeMessages := make(map[string]string)
	var summaryLine string

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: file.py:line:col: CODE message
		parts := strings.SplitN(line, " ", 3)
		if len(parts) >= 2 {
			// Extract code (uppercase letters + digits)
			for _, p := range parts {
				if len(p) >= 2 && p[0] >= 'A' && p[0] <= 'Z' {
					code := p
					if idx := strings.Index(code, ":"); idx > 0 {
						code = code[:idx]
					}
					if len(code) >= 2 && code[0] >= 'A' && code[0] <= 'Z' {
						codeCounts[code]++
						if codeMessages[code] == "" && len(parts) >= 3 {
							codeMessages[code] = utils.Truncate(parts[2], 60)
						}
					}
				}
			}
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "found") || strings.Contains(lower, "error") {
			summaryLine = line
		}
	}

	if len(codeCounts) == 0 {
		return "no ruff issues ✓\n"
	}

	codes := make([]string, 0, len(codeCounts))
	for c := range codeCounts {
		codes = append(codes, c)
	}
	sort.Slice(codes, func(i, j int) bool {
		return codeCounts[codes[i]] > codeCounts[codes[j]]
	})

	var sb strings.Builder
	for _, c := range codes {
		fmt.Fprintf(&sb, "  %-10s  %d×  %s\n", c, codeCounts[c], codeMessages[c])
	}
	if summaryLine != "" {
		sb.WriteString(summaryLine)
		sb.WriteString("\n")
	}
	return sb.String()
}

// filterRuffFormat shows files that need formatting.
func filterRuffFormat(stdout, stderr string) string {
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
		l = strings.TrimSpace(l)
		if l != "" {
			kept = append(kept, l)
		}
	}
	if len(kept) == 0 {
		return "all files formatted ✓\n"
	}
	return strings.Join(kept, "\n") + "\n"
}

func init() {
	ruffCmd.AddCommand(ruffCheckCmd)
	ruffCmd.AddCommand(ruffFormatCmd)
	rootCmd.AddCommand(ruffCmd)
}
