package utils

import (
	"context"
	"bytes"
	"os/exec"
	"time"
)

// ExecResult contains outcome of command execution
type ExecResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Error    error
}

// RunCommand executes a subprocess command with context support
func RunCommand(ctx context.Context, name string, args ...string) ExecResult {
	cmd := exec.CommandContext(ctx, name, args...)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	} else if err != nil {
		exitCode = 1
	}

	return ExecResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Error:    err,
	}
}

// CommandExists checks if a binary executable exists in PATH
func CommandExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// MeasureExec executes command and records duration
func MeasureExec(fn func() (string, string, error)) (output string, errStr string, durationMs int64, err error) {
	start := time.Now()
	out, stdErr, runErr := fn()
	elapsed := time.Since(start).Milliseconds()
	return out, stdErr, elapsed, runErr
}
