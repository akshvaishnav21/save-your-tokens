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
	dockerLayerRe   = regexp.MustCompile(`(?i)^(sha256:|Step \d+|---> |Removing intermediate|Successfully built|Successfully tagged)`)
	dockerPsHeaderRe = regexp.MustCompile(`(?i)^CONTAINER ID`)
)

var dockerCmd = &cobra.Command{
	Use:   "docker",
	Short: "Token-optimized docker commands",
}

var dockerPsCmd = &cobra.Command{
	Use:                "ps [args...]",
	Short:              "Compact docker ps output",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDockerSubcmd("docker ps", append([]string{"ps"}, args...), filterDockerPs)
	},
}

var dockerBuildCmd = &cobra.Command{
	Use:                "build [args...]",
	Short:              "Filtered docker build (strip layer hashes)",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDockerSubcmd("docker build", append([]string{"build"}, args...), filterDockerBuild)
	},
}

var dockerLogsCmd = &cobra.Command{
	Use:                "logs [args...]",
	Short:              "docker logs (last N lines)",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runDockerSubcmd("docker logs", append([]string{"logs", "--tail=50"}, args...), filterDockerLogs)
	},
}

func runDockerSubcmd(cmdName string, args []string, fn func(string, string) string) error {
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

	exitCode, err := runner.RunWithFilter(cmdName, "docker", args, fn)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

// filterDockerPs produces a compact container table.
func filterDockerPs(stdout, stderr string) string {
	if stderr != "" && stdout == "" {
		return stderr
	}
	combined := utils.StripANSI(stdout)
	lines := strings.Split(combined, "\n")
	var kept []string
	for _, l := range lines {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		// Keep header and container lines, but compact them
		if dockerPsHeaderRe.MatchString(l) {
			// Just show: ID, IMAGE, STATUS, NAMES
			kept = append(kept, "CONTAINER ID  IMAGE  STATUS  NAMES")
			continue
		}
		// Each container line: take first field (id), image field, status, name
		fields := strings.Fields(l)
		if len(fields) >= 7 {
			id := fields[0][:12]
			if len(id) > 12 {
				id = id[:12]
			}
			image := fields[1]
			status := strings.Join(fields[4:6], " ")
			name := fields[len(fields)-1]
			kept = append(kept, fmt.Sprintf("%s  %-30s  %-20s  %s", id, image, status, name))
		}
	}
	if len(kept) == 0 {
		return "no containers running\n"
	}
	return strings.Join(kept, "\n") + "\n"
}

// filterDockerBuild strips layer hashes and verbose output.
func filterDockerBuild(stdout, stderr string) string {
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
		stripped := strings.TrimSpace(l)
		if stripped == "" {
			continue
		}
		if dockerLayerRe.MatchString(stripped) {
			// Keep final success/tag lines
			if strings.HasPrefix(strings.ToLower(stripped), "successfully") {
				kept = append(kept, stripped)
			}
			continue
		}
		// Skip lines starting with sha256:
		if strings.HasPrefix(stripped, "sha256:") || strings.HasPrefix(stripped, "#") {
			continue
		}
		kept = append(kept, stripped)
	}
	if len(kept) == 0 {
		return "docker build ok ✓\n"
	}
	return strings.Join(kept, "\n") + "\n"
}

// filterDockerLogs passes through log output (already limited by --tail).
func filterDockerLogs(stdout, stderr string) string {
	combined := stdout
	if stderr != "" {
		if combined != "" {
			combined += "\n" + stderr
		} else {
			combined = stderr
		}
	}
	return combined
}

func init() {
	dockerCmd.AddCommand(dockerPsCmd)
	dockerCmd.AddCommand(dockerBuildCmd)
	dockerCmd.AddCommand(dockerLogsCmd)
	rootCmd.AddCommand(dockerCmd)
}
