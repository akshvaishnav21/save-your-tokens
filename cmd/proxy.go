package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/saveyourtokens/syt/internal/config"
	"github.com/saveyourtokens/syt/internal/tracker"
	"github.com/saveyourtokens/syt/internal/utils"
	"github.com/spf13/cobra"
)

var proxyCmd = &cobra.Command{
	Use:                "proxy <cmd> [args...]",
	Short:              "Passthrough with token tracking",
	DisableFlagParsing: true,
	Args:               cobra.MinimumNArgs(1),
	RunE:               runProxy,
}

func runProxy(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("proxy requires at least one argument")
	}

	binary := args[0]
	cmdArgs := args[1:]

	// Resolve binary
	binaryPath, err := exec.LookPath(binary)
	if err != nil {
		binaryPath = binary
	}

	c := exec.Command(binaryPath, cmdArgs...)
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr

	cmdErr := c.Run()
	exitCode := 0
	if cmdErr != nil {
		if exitErr, ok := cmdErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
		} else {
			return cmdErr
		}
	}

	// Track (non-blocking)
	cfg := config.Load()
	dbPath := cfg.Tracking.DatabasePath
	if dbPath == "" {
		dbPath = utils.DataDir() + "/syt.db"
	}
	go func() {
		t, err := tracker.NewTracker(dbPath)
		if err != nil {
			return
		}
		defer t.Close()
		_ = t.Track(tracker.Record{
			OriginalCmd: binary,
			SytCmd:      "syt proxy " + binary,
		})
	}()

	if exitCode != 0 {
		os.Exit(exitCode)
	}
	return nil
}

func init() {
	rootCmd.AddCommand(proxyCmd)
}
