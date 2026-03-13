package cmd

import (
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

var prismaAsciiArtRe = regexp.MustCompile(`(?i)^[\s*╔╗╚╝║═\-─│+|\\\/]+$`)

var prismaCmd = &cobra.Command{
	Use:   "prisma",
	Short: "Token-optimized prisma commands",
}

var prismaMigrateCmd = &cobra.Command{
	Use:                "migrate [args...]",
	Short:              "Filtered prisma migrate",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPrismaSubcmd("prisma migrate", append([]string{"migrate"}, args...), filterPrisma)
	},
}

var prismaGenerateCmd = &cobra.Command{
	Use:                "generate [args...]",
	Short:              "Filtered prisma generate",
	DisableFlagParsing: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runPrismaSubcmd("prisma generate", append([]string{"generate"}, args...), filterPrisma)
	},
}

func runPrismaSubcmd(cmdName string, args []string, fn func(string, string) string) error {
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

	exitCode, err := runner.RunWithFilter(cmdName, "npx", append([]string{"prisma"}, args...), fn)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

// filterPrisma strips ASCII art and keeps meaningful output.
func filterPrisma(stdout, stderr string) string {
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

	for _, line := range lines {
		stripped := strings.TrimSpace(line)
		if stripped == "" {
			continue
		}
		// Skip ASCII art lines (lines with only border chars)
		if prismaAsciiArtRe.MatchString(stripped) {
			continue
		}
		// Skip Prisma logo lines
		if strings.Contains(stripped, "◞") || strings.Contains(stripped, "◟") ||
			strings.Contains(stripped, "Prisma is") || strings.HasPrefix(stripped, "Prisma schema") {
			continue
		}
		kept = append(kept, stripped)
	}

	if len(kept) == 0 {
		return "prisma ok ✓\n"
	}
	return strings.Join(kept, "\n") + "\n"
}

func init() {
	prismaCmd.AddCommand(prismaMigrateCmd)
	prismaCmd.AddCommand(prismaGenerateCmd)
	rootCmd.AddCommand(prismaCmd)
}
