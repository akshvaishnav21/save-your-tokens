package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/saveyourtokens/syt/internal/config"
	"github.com/saveyourtokens/syt/internal/filter"
	"github.com/saveyourtokens/syt/internal/tee"
	"github.com/saveyourtokens/syt/internal/tracker"
	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/spf13/cobra"
)

var goCmd = &cobra.Command{
	Use:   "go",
	Short: "Token-optimized go commands",
}

var goTestCmd = &cobra.Command{
	Use:                "test [args...]",
	Short:              "Filtered go test (failures only)",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGoSubcmd("go test", buildGoTestArgs(args), filterGoTest)
	},
}

var goBuildCmd = &cobra.Command{
	Use:                "build [args...]",
	Short:              "Filtered go build (errors only)",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGoSubcmd("go build", append([]string{"build"}, args...), filterGoBuild)
	},
}

var goVetCmd = &cobra.Command{
	Use:                "vet [args...]",
	Short:              "Filtered go vet (issues only)",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGoSubcmd("go vet", append([]string{"vet"}, args...), filterGoVet)
	},
}

var goRunCmd = &cobra.Command{
	Use:                "run [args...]",
	Short:              "go run passthrough",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runGoSubcmd("go run", append([]string{"run"}, args...), filterGoRun)
	},
}

func buildGoTestArgs(args []string) []string {
	// Insert -json flag if not present
	hasJSON := false
	for _, a := range args {
		if a == "-json" {
			hasJSON = true
			break
		}
	}
	if hasJSON {
		return append([]string{"test"}, args...)
	}
	return append([]string{"test", "-json"}, args...)
}

func runGoSubcmd(cmdName string, args []string, fn func(string, string) string) error {
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

	exitCode, err := runner.RunWithFilter(cmdName, "go", args, fn)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

// goTestEvent is a single JSON line from go test -json
type goTestEvent struct {
	Action  string  `json:"Action"`
	Package string  `json:"Package"`
	Test    string  `json:"Test"`
	Output  string  `json:"Output"`
	Elapsed float64 `json:"Elapsed"`
}

// filterGoTest parses go test -json output and shows failures only.
func filterGoTest(stdout, stderr string) string {
	if stderr != "" && stdout == "" {
		return stderr
	}

	lines := strings.Split(stdout, "\n")
	// Track output per test
	testOutput := make(map[string][]string)
	var failedTests []string
	var passCount int
	var failCount int

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event goTestEvent
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}

		key := event.Package + "/" + event.Test
		switch event.Action {
		case "output":
			if event.Test != "" {
				testOutput[key] = append(testOutput[key], event.Output)
			}
		case "pass":
			if event.Test != "" {
				passCount++
			}
		case "fail":
			if event.Test != "" {
				failCount++
				failedTests = append(failedTests, key)
			}
		}
	}

	if failCount == 0 && passCount == 0 {
		// Fallback: return stderr if any
		if stderr != "" {
			return stderr
		}
		return "no tests run\n"
	}

	if failCount == 0 {
		return fmt.Sprintf("%d tests passed ✓\n", passCount)
	}

	var sb strings.Builder
	for _, key := range failedTests {
		if out, ok := testOutput[key]; ok {
			for _, l := range out {
				sb.WriteString(l)
			}
		}
	}
	fmt.Fprintf(&sb, "\n%d passed, %d failed\n", passCount, failCount)
	return sb.String()
}

// filterGoBuild shows only errors.
func filterGoBuild(stdout, stderr string) string {
	combined := stderr
	if stdout != "" {
		if combined != "" {
			combined = stdout + "\n" + combined
		} else {
			combined = stdout
		}
	}
	if strings.TrimSpace(combined) == "" {
		return "build ok ✓\n"
	}
	return combined
}

// filterGoVet shows only vet issues.
func filterGoVet(stdout, stderr string) string {
	combined := stderr
	if stdout != "" {
		if combined != "" {
			combined = stdout + "\n" + combined
		} else {
			combined = stdout
		}
	}
	if strings.TrimSpace(combined) == "" {
		return "vet ok ✓\n"
	}
	return combined
}

// filterGoRun passes through output unchanged.
func filterGoRun(stdout, stderr string) string {
	if stderr != "" && stdout == "" {
		return stderr
	}
	if stdout != "" && stderr != "" {
		return stdout + "\n" + stderr
	}
	return stdout
}

func init() {
	goCmd.AddCommand(goTestCmd)
	goCmd.AddCommand(goBuildCmd)
	goCmd.AddCommand(goVetCmd)
	goCmd.AddCommand(goRunCmd)
	rootCmd.AddCommand(goCmd)
}
