package gates

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

// runCmdFunc is the function used to run commands. Tests can replace this
// to inject mock behavior without executing real binaries.
var runCmdFunc = runCmdImpl

// runCmd delegates to runCmdFunc so that tests can swap in a mock.
func runCmd(ctx context.Context, dir string, timeoutSec int, name string, args ...string) (bool, string, error) {
	return runCmdFunc(ctx, dir, timeoutSec, name, args...)
}

// runCmdImpl is the real implementation that executes external commands.
func runCmdImpl(ctx context.Context, dir string, timeoutSec int, name string, args ...string) (bool, string, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Duration(timeoutSec)*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = dir

	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	output := buf.String()

	if ctx.Err() == context.DeadlineExceeded {
		return false, output, fmt.Errorf("timeout after %ds", timeoutSec)
	}

	if err != nil {
		// Non-zero exit code — command ran but failed
		if _, ok := err.(*exec.ExitError); ok {
			return false, output, nil
		}
		// Command not found or other system error
		return false, output, fmt.Errorf("exec %s: %w", name, err)
	}

	return true, output, nil
}
