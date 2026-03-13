package filter

import (
	"bytes"
	"fmt"
	"os/exec"
	"time"

	"github.com/saveyourtokens/syt/internal/tee"
	"github.com/saveyourtokens/syt/internal/tracker"
	"github.com/saveyourtokens/syt/internal/utils"
)

// Runner executes commands and applies output filters.
type Runner struct {
	Verbose int
	Tracker *tracker.Tracker // nil = tracking disabled
	Tee     *tee.Tee         // nil = tee disabled
}

// RunWithFilter executes binary with args, applies filterFn, prints result.
// filterFn receives raw stdout and stderr, returns filtered string.
// If filterFn panics, falls back to printing raw stdout+stderr.
// If exitCode != 0 and raw output >= 500 bytes, triggers tee recovery.
// Tracking happens in a goroutine (non-blocking).
// Returns the original command exit code.
func (r *Runner) RunWithFilter(
	cmdName string,
	binary string,
	args []string,
	filterFn func(stdout, stderr string) string,
) (exitCode int, err error) {
	start := time.Now()

	cmd := exec.Command(binary, args...)
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	runErr := cmd.Run()
	elapsed := time.Since(start)

	exitCode = 0
	if runErr != nil {
		if exitErr, ok := runErr.(*exec.ExitError); ok {
			exitCode = exitErr.ExitCode()
			if exitCode < 0 {
				exitCode = 1
			}
		} else {
			return 1, fmt.Errorf("running %s: %w", binary, runErr)
		}
	}

	rawStdout := stdoutBuf.String()
	rawStderr := stderrBuf.String()
	rawCombined := rawStdout
	if rawStderr != "" {
		if rawCombined != "" {
			rawCombined += "\n" + rawStderr
		} else {
			rawCombined = rawStderr
		}
	}

	// Apply filter with panic recovery
	var filtered string
	var filterPanicked bool
	func() {
		defer func() {
			if rec := recover(); rec != nil {
				filterPanicked = true
			}
		}()
		filtered = filterFn(rawStdout, rawStderr)
	}()

	if filterPanicked || (filtered == "" && rawCombined != "") {
		filtered = rawCombined
	}

	fmt.Print(filtered)
	if len(filtered) > 0 && filtered[len(filtered)-1] != '\n' {
		fmt.Println()
	}

	// Tee on failure with large output
	if exitCode != 0 && len(rawCombined) >= 500 && r.Tee != nil {
		if filePath := r.Tee.Save(rawCombined, cmdName, exitCode); filePath != "" {
			hint := r.Tee.Hint(filePath)
			fmt.Println(hint)
		}
	}

	// Non-blocking tracker write
	if r.Tracker != nil {
		inputTokens := utils.CountTokens(rawCombined)
		outputTokens := utils.CountTokens(filtered)
		rec := tracker.Record{
			OriginalCmd:  cmdName,
			SytCmd:       "syt " + cmdName,
			InputTokens:  inputTokens,
			OutputTokens: outputTokens,
			ExecutionMs:  elapsed.Milliseconds(),
		}
		go func() {
			_ = r.Tracker.Track(rec)
		}()
	}

	return exitCode, nil
}
