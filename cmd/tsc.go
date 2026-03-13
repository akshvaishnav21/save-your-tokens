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

var tscErrorRe = regexp.MustCompile(`^(.+)\((\d+),(\d+)\):\s+error\s+(TS\d+):\s+(.+)$`)

var tscCmd = &cobra.Command{
	Use:                "tsc [args...]",
	Short:              "Filtered TypeScript compiler (errors grouped by file)",
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

		exitCode, err := runner.RunWithFilter("tsc", "tsc", args, filterTsc)
		if err != nil {
			return err
		}
		if exitCode != 0 {
			os.Exit(exitCode)
		}
		return nil
	},
}

type tscError struct {
	File    string
	Line    int
	Col     int
	Code    string
	Message string
}

// filterTsc parses TypeScript compiler output and groups errors by file.
func filterTsc(stdout, stderr string) string {
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
	var errors []tscError

	for _, line := range lines {
		if m := tscErrorRe.FindStringSubmatch(line); m != nil {
			errors = append(errors, tscError{
				File:    m[1],
				Code:    m[4],
				Message: m[5],
			})
		}
	}

	if len(errors) == 0 {
		return "no TypeScript errors ✓\n"
	}

	// Group by file
	fileErrors := make(map[string][]tscError)
	fileOrder := []string{}
	seen := make(map[string]bool)
	for _, e := range errors {
		if !seen[e.File] {
			fileOrder = append(fileOrder, e.File)
			seen[e.File] = true
		}
		fileErrors[e.File] = append(fileErrors[e.File], e)
	}
	sort.Strings(fileOrder)

	var sb strings.Builder
	for _, file := range fileOrder {
		errs := fileErrors[file]
		codes := make(map[string]bool)
		for _, e := range errs {
			codes[e.Code] = true
		}
		codeList := make([]string, 0, len(codes))
		for c := range codes {
			codeList = append(codeList, c)
		}
		sort.Strings(codeList)
		fmt.Fprintf(&sb, "%s: %d errors (%s)\n", file, len(errs), strings.Join(codeList, ", "))
	}

	totalFiles := len(fileOrder)
	fmt.Fprintf(&sb, "%d errors in %d files\n", len(errors), totalFiles)
	return sb.String()
}

func init() {
	rootCmd.AddCommand(tscCmd)
}
