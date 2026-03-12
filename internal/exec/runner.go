package exec

import (
	"bytes"
	"os/exec"
	"syscall"
)

// Result holds the output of a command execution.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// Run executes a command and captures its output.
func Run(name string, args ...string) (*Result, error) {
	cmd := exec.Command(name, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	result := &Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				result.ExitCode = status.ExitStatus()
			} else {
				result.ExitCode = 1
			}
			// Non-zero exit is not necessarily an error (e.g., grep no match)
			return result, nil
		}
		// Actual execution error (command not found, etc.)
		return result, err
	}

	return result, nil
}
