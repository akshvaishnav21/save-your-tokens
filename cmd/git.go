package cmd

import (
	"fmt"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/saveyourtokens/syt/internal/config"
	"github.com/saveyourtokens/syt/internal/filter"
	"github.com/saveyourtokens/syt/internal/tracker"
	"github.com/saveyourtokens/syt/internal/tee"
	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/spf13/cobra"
)

var (
	gitDiffStatFileRe = regexp.MustCompile(`^\s+(\S.*?)\s+\|\s+(\d+)`)
	gitDiffSummaryRe  = regexp.MustCompile(`(\d+) files? changed`)
	aheadRe           = regexp.MustCompile(`ahead (\d+)`)
	behindRe          = regexp.MustCompile(`behind (\d+)`)
)

var gitCmd = &cobra.Command{
	Use:   "git",
	Short: "Token-optimized git commands",
}

var gitLogCmd = &cobra.Command{
	Use:   "log [args...]",
	Short: "Filtered git log (hash + message only)",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGitSubcmd("git log", "git", buildGitLogArgs(args), filterGitLog)
	},
}

var gitDiffCmd = &cobra.Command{
	Use:   "diff [args...]",
	Short: "Filtered git diff (stats only)",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGitSubcmd("git diff", "git", buildGitDiffArgs(args), filterGitDiff)
	},
}

var gitStatusCmd = &cobra.Command{
	Use:   "status [args...]",
	Short: "Filtered git status (compact summary)",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGitSubcmd("git status", "git", buildGitStatusArgs(args), filterGitStatus)
	},
}

var gitPushCmd = &cobra.Command{
	Use:   "push [args...]",
	Short: "git push with compact output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGitSubcmd("git push", "git", append([]string{"push"}, args...), filterGitSimple("push"))
	},
}

var gitPullCmd = &cobra.Command{
	Use:   "pull [args...]",
	Short: "git pull with compact output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGitSubcmd("git pull", "git", append([]string{"pull"}, args...), filterGitSimple("pull"))
	},
}

var gitCommitCmd = &cobra.Command{
	Use:   "commit [args...]",
	Short: "git commit with compact output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGitSubcmd("git commit", "git", append([]string{"commit"}, args...), filterGitSimple("commit"))
	},
}

var gitAddCmd = &cobra.Command{
	Use:   "add [args...]",
	Short: "git add with compact output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGitSubcmd("git add", "git", append([]string{"add"}, args...), filterGitSimple("add"))
	},
}

var gitFetchCmd = &cobra.Command{
	Use:   "fetch [args...]",
	Short: "git fetch with compact output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGitSubcmd("git fetch", "git", append([]string{"fetch"}, args...), filterGitSimple("fetch"))
	},
}

var gitBranchCmd = &cobra.Command{
	Use:   "branch [args...]",
	Short: "git branch with compact output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGitSubcmd("git branch", "git", append([]string{"branch"}, args...), filterGitBranch)
	},
}

var gitStashCmd = &cobra.Command{
	Use:   "stash [args...]",
	Short: "git stash with compact output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGitSubcmd("git stash", "git", append([]string{"stash"}, args...), filterGitSimple("stash"))
	},
}

var gitWorktreeCmd = &cobra.Command{
	Use:   "worktree [args...]",
	Short: "git worktree with compact output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGitSubcmd("git worktree", "git", append([]string{"worktree"}, args...), filterGitWorktree)
	},
}

func buildGitLogArgs(args []string) []string {
	result := []string{"log"}
	hasOneline := false
	for _, a := range args {
		if a == "--oneline" {
			hasOneline = true
		}
		result = append(result, a)
	}
	if !hasOneline {
		result = append(result, "--oneline")
	}
	return result
}

func buildGitDiffArgs(args []string) []string {
	// Check if user already passed --stat
	for _, a := range args {
		if a == "--stat" || strings.HasPrefix(a, "--stat=") {
			return append([]string{"diff"}, args...)
		}
	}
	// Only use --stat if not in verbose mode
	if verboseCount > 0 {
		return append([]string{"diff"}, args...)
	}
	return append([]string{"diff", "--stat"}, args...)
}

func buildGitStatusArgs(args []string) []string {
	return append([]string{"status", "--short", "--branch"}, args...)
}

func runGitSubcmd(cmdName, binary string, args []string, fn func(string, string) string) error {
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

// filterGitLog compresses git log output.
// Handles both --oneline format and verbose format.
func filterGitLog(stdout, stderr string) string {
	if stdout == "" && stderr != "" {
		return stderr
	}
	if stdout == "" {
		return "(no commits)"
	}

	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")

	// Detect if this is verbose format (has "Author:" lines)
	isVerbose := false
	for _, l := range lines {
		if strings.HasPrefix(l, "Author:") || strings.HasPrefix(l, "Date:") {
			isVerbose = true
			break
		}
	}

	var kept []string

	if isVerbose {
		// Verbose format: extract commit hash + first line of message
		var currentHash string
		var collectingMsg bool
		var msgCollected bool
		for _, line := range lines {
			if strings.HasPrefix(line, "commit ") {
				// Save previous commit if we have one
				currentHash = ""
				collectingMsg = false
				msgCollected = false
				// Extract short hash
				parts := strings.Fields(line)
				if len(parts) >= 2 && len(parts[1]) >= 7 {
					currentHash = parts[1][:7]
				}
				continue
			}
			if strings.HasPrefix(line, "Author:") || strings.HasPrefix(line, "Date:") ||
				strings.HasPrefix(line, "Merge:") || strings.HasPrefix(line, "gpg:") {
				continue
			}
			if currentHash != "" && !msgCollected {
				msg := strings.TrimSpace(line)
				if msg != "" && !collectingMsg {
					collectingMsg = true
					kept = append(kept, currentHash+" "+msg)
					msgCollected = true
				}
			}
		}
	} else {
		// --oneline format: keep as-is
		for _, line := range lines {
			line = strings.TrimSpace(line)
			if line == "" {
				continue
			}
			kept = append(kept, line)
		}
	}

	if len(kept) == 0 {
		if stderr != "" {
			return stderr
		}
		return "(no commits)"
	}
	result := strings.Join(kept, "\n")
	result += fmt.Sprintf("\n%d commits", len(kept))
	return result
}

// filterGitDiff compresses git diff --stat output.
func filterGitDiff(stdout, stderr string) string {
	if stdout == "" {
		if stderr != "" {
			return stderr
		}
		return "no changes"
	}
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	var files []string
	var summary string
	for _, line := range lines {
		if gitDiffStatFileRe.MatchString(line) {
			files = append(files, strings.TrimSpace(line))
		} else if gitDiffSummaryRe.MatchString(line) {
			summary = strings.TrimSpace(line)
		}
	}
	if len(files) == 0 {
		return stdout // passthrough if format unexpected
	}
	var sb strings.Builder
	for _, f := range files {
		sb.WriteString(f)
		sb.WriteString("\n")
	}
	if summary != "" {
		sb.WriteString(summary)
		sb.WriteString("\n")
	}
	return sb.String()
}

// filterGitStatus compresses git status --short --branch output.
func filterGitStatus(stdout, stderr string) string {
	if stdout == "" {
		if stderr != "" {
			return stderr
		}
		return "clean"
	}
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")

	var branchLine string
	var staged, modified, untracked int

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			branchLine = line[3:]
			continue
		}
		if len(line) < 2 {
			continue
		}
		x := line[0]
		y := line[1]
		if x != ' ' && x != '?' && x != '!' {
			staged++
		}
		if y == 'M' || y == 'D' {
			modified++
		}
		if x == '?' && y == '?' {
			untracked++
		}
	}

	// Parse branch info: "main...origin/main [ahead 2, behind 3]"
	var branch string
	var ahead, behind int

	// Extract branch name (before ...)
	if idx := strings.Index(branchLine, "..."); idx >= 0 {
		branch = branchLine[:idx]
	} else if idx := strings.Index(branchLine, " ["); idx >= 0 {
		branch = branchLine[:idx]
	} else {
		branch = branchLine
	}

	// Extract ahead/behind from [...] section
	if m := aheadRe.FindStringSubmatch(branchLine); m != nil {
		ahead, _ = strconv.Atoi(m[1])
	}
	if m := behindRe.FindStringSubmatch(branchLine); m != nil {
		behind, _ = strconv.Atoi(m[1])
	}

	var sb strings.Builder
	sb.WriteString("On ")
	sb.WriteString(strings.TrimSpace(branch))
	if ahead > 0 {
		fmt.Fprintf(&sb, " (↑%d)", ahead)
	}
	if behind > 0 {
		fmt.Fprintf(&sb, " (↓%d)", behind)
	}
	sb.WriteString(". ")

	statusParts := []string{}
	if staged > 0 {
		statusParts = append(statusParts, fmt.Sprintf("Staged: %d", staged))
	}
	if modified > 0 {
		statusParts = append(statusParts, fmt.Sprintf("Modified: %d", modified))
	}
	if untracked > 0 {
		statusParts = append(statusParts, fmt.Sprintf("Untracked: %d", untracked))
	}
	if len(statusParts) == 0 {
		sb.WriteString("Clean")
	} else {
		sb.WriteString(strings.Join(statusParts, " · "))
	}
	return sb.String()
}

// filterGitSimple handles push/pull/commit/add/fetch: success=one line, error=full.
func filterGitSimple(op string) func(string, string) string {
	return func(stdout, stderr string) string {
		combined := stdout
		if stderr != "" {
			if combined != "" {
				combined += "\n" + stderr
			} else {
				combined = stderr
			}
		}
		// Look for error indicators
		lower := strings.ToLower(combined)
		if strings.Contains(lower, "error") || strings.Contains(lower, "fatal") ||
			strings.Contains(lower, "rejected") || strings.Contains(lower, "failed") {
			return combined
		}
		// Extract meaningful one-liner
		lines := strings.Split(strings.TrimSpace(combined), "\n")
		for _, l := range lines {
			l = strings.TrimSpace(l)
			if l != "" && !strings.HasPrefix(l, "remote: ") && !strings.HasPrefix(l, "To ") &&
				!strings.HasPrefix(l, "From ") && !strings.HasPrefix(l, "Counting") &&
				!strings.HasPrefix(l, "Writing") && !strings.HasPrefix(l, "Compressing") {
				return fmt.Sprintf("ok ✓ %s: %s", op, l)
			}
		}
		if combined != "" {
			return fmt.Sprintf("ok ✓ %s", op)
		}
		return fmt.Sprintf("ok ✓ %s", op)
	}
}

// filterGitBranch compresses git branch output.
func filterGitBranch(stdout, stderr string) string {
	if stderr != "" && stdout == "" {
		return stderr
	}
	lines := strings.Split(strings.TrimRight(stdout, "\n"), "\n")
	var kept []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			kept = append(kept, l)
		}
	}
	return strings.Join(kept, "\n")
}

// filterGitWorktree compresses git worktree output.
func filterGitWorktree(stdout, stderr string) string {
	if stderr != "" && stdout == "" {
		return stderr
	}
	return strings.TrimSpace(stdout)
}

func init() {
	// Wire up subcommands
	gitCmd.AddCommand(gitLogCmd)
	gitCmd.AddCommand(gitDiffCmd)
	gitCmd.AddCommand(gitStatusCmd)
	gitCmd.AddCommand(gitPushCmd)
	gitCmd.AddCommand(gitPullCmd)
	gitCmd.AddCommand(gitCommitCmd)
	gitCmd.AddCommand(gitAddCmd)
	gitCmd.AddCommand(gitFetchCmd)
	gitCmd.AddCommand(gitBranchCmd)
	gitCmd.AddCommand(gitStashCmd)
	gitCmd.AddCommand(gitWorktreeCmd)
	rootCmd.AddCommand(gitCmd)
}
