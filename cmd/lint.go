package cmd

import (
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/saveyourtokens/syt/internal/config"
	"github.com/saveyourtokens/syt/internal/filter"
	"github.com/saveyourtokens/syt/internal/tee"
	"github.com/saveyourtokens/syt/internal/tracker"
	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/spf13/cobra"
)

var (
	eslintViolationRe = regexp.MustCompile(`^\s+\d+:\d+\s+(error|warning)\s+(.+?)\s+(\S+)$`)
	eslintFileRe      = regexp.MustCompile(`^(/[^\s]+|\./[^\s]+|[A-Za-z]:[^\s]+)\s*$`)
	biomeViolationRe  = regexp.MustCompile(`(?i)(error|warn):\s+(.+?)\s+(\w+/\w+)`)
)

var lintCmd = &cobra.Command{
	Use:   "lint",
	Short: "Token-optimized lint commands (eslint, biome, mypy, prettier)",
}

var lintEslintCmd = &cobra.Command{
	Use:                "eslint [args...]",
	Short:              "Filtered eslint output (grouped by rule)",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLintSubcmd("eslint", "eslint", args, filterEslint)
	},
}

var lintBiomeCmd = &cobra.Command{
	Use:                "biome [args...]",
	Short:              "Filtered biome check output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLintSubcmd("biome check", "biome", append([]string{"check"}, args...), filterBiome)
	},
}

var lintMypyCmd = &cobra.Command{
	Use:                "mypy [args...]",
	Short:              "Filtered mypy output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLintSubcmd("mypy", "mypy", args, filterMypy)
	},
}

var lintPrettierCmd = &cobra.Command{
	Use:                "prettier [args...]",
	Short:              "Filtered prettier --check output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runLintSubcmd("prettier --check", "prettier", append([]string{"--check"}, args...), filterPrettier)
	},
}

func runLintSubcmd(cmdName, binary string, args []string, fn func(string, string) string) error {
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

	exitCode, err := runner.RunWithFilter(cmdName, binary, args, fn)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

// filterEslint groups violations by rule.
func filterEslint(stdout, stderr string) string {
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
	ruleCounts := make(map[string]int)
	ruleExamples := make(map[string]string)
	var summaryLines []string

	for _, line := range lines {
		if m := eslintViolationRe.FindStringSubmatch(line); m != nil {
			rule := m[3]
			ruleCounts[rule]++
			if ruleExamples[rule] == "" {
				ruleExamples[rule] = strings.TrimSpace(m[2])
			}
			continue
		}
		lower := strings.ToLower(line)
		if strings.Contains(lower, "problem") || strings.Contains(lower, "error") ||
			strings.Contains(lower, "warning") {
			l := strings.TrimSpace(line)
			if l != "" {
				summaryLines = append(summaryLines, l)
			}
		}
	}

	if len(ruleCounts) == 0 {
		return "no lint issues ✓\n"
	}

	// Sort rules by count
	rules := make([]string, 0, len(ruleCounts))
	for r := range ruleCounts {
		rules = append(rules, r)
	}
	sort.Slice(rules, func(i, j int) bool {
		return ruleCounts[rules[i]] > ruleCounts[rules[j]]
	})

	var sb strings.Builder
	for _, r := range rules {
		fmt.Fprintf(&sb, "  %-40s  %d×  %s\n", r, ruleCounts[r], utils.Truncate(ruleExamples[r], 50))
	}
	// Deduplicate summary
	seen := make(map[string]bool)
	for _, l := range summaryLines {
		if !seen[l] {
			seen[l] = true
			sb.WriteString(l)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}

// filterBiome shows errors and warnings grouped.
func filterBiome(stdout, stderr string) string {
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
		lower := strings.ToLower(l)
		if strings.Contains(lower, "error") || strings.Contains(lower, "warn") ||
			strings.Contains(lower, "fixed") || strings.Contains(lower, "found") {
			if l != "" {
				kept = append(kept, l)
			}
		}
	}
	if len(kept) == 0 {
		return "no issues ✓\n"
	}
	return strings.Join(kept, "\n") + "\n"
}

// filterMypy shows mypy errors compactly.
func filterMypy(stdout, stderr string) string {
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
	var errors []string
	var summary string
	for _, l := range lines {
		if strings.Contains(l, ": error:") || strings.Contains(l, ": note:") {
			errors = append(errors, strings.TrimSpace(l))
		} else if strings.HasPrefix(l, "Found") || strings.HasPrefix(l, "Success") {
			summary = strings.TrimSpace(l)
		}
	}
	var sb strings.Builder
	for _, e := range errors {
		sb.WriteString(e)
		sb.WriteString("\n")
	}
	if summary != "" {
		sb.WriteString(summary)
		sb.WriteString("\n")
	}
	if sb.Len() == 0 {
		return "no type errors ✓\n"
	}
	return sb.String()
}

// filterPrettier shows files that need formatting.
func filterPrettier(stdout, stderr string) string {
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
	var needsFormat []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l != "" && !strings.HasPrefix(l, "Checking") {
			needsFormat = append(needsFormat, l)
		}
	}
	if len(needsFormat) == 0 {
		return "all files formatted ✓\n"
	}
	return strings.Join(needsFormat, "\n") + "\n"
}

func init() {
	lintCmd.AddCommand(lintEslintCmd)
	lintCmd.AddCommand(lintBiomeCmd)
	lintCmd.AddCommand(lintMypyCmd)
	lintCmd.AddCommand(lintPrettierCmd)
	rootCmd.AddCommand(lintCmd)
}
