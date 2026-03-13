package exec

import (
	"bytes"
	"context"
	"io"
	"os/exec"
	"syscall"
	"time"

	gotkerrors "github.com/antikkorps/GoTK/internal/errors"
)

// DefaultTimeout is the default command execution timeout.
const DefaultTimeout = 30 * time.Second

// MaxOutputBytes is the maximum number of bytes captured from stdout or stderr.
const MaxOutputBytes = 10 * 1024 * 1024 // 10MB

const truncationMessage = "\n[output truncated: exceeded 10MB limit]"

// Result holds the output of a command execution.
type Result struct {
	Stdout   string
	Stderr   string
	ExitCode int
}

// LimitedBuffer wraps an io.Writer and stops writing after max bytes.
// Once the limit is reached, all subsequent writes are silently discarded.
type LimitedBuffer struct {
	buf     bytes.Buffer
	max     int
	written int
	capped  bool
}

// NewLimitedBuffer creates a LimitedBuffer with the given maximum size.
func NewLimitedBuffer(max int) *LimitedBuffer {
	return &LimitedBuffer{max: max}
}

// Write implements io.Writer. It writes up to the remaining capacity
// and silently discards the rest.
func (lb *LimitedBuffer) Write(p []byte) (int, error) {
	if lb.capped {
		return len(p), nil // pretend we wrote it all
	}
	remaining := lb.max - lb.written
	if remaining <= 0 {
		lb.capped = true
		return len(p), nil
	}
	toWrite := p
	if len(toWrite) > remaining {
		toWrite = toWrite[:remaining]
		lb.capped = true
	}
	n, err := lb.buf.Write(toWrite)
	lb.written += n
	if err != nil {
		return n, err
	}
	return len(p), nil // report full write to avoid cmd errors
}

// String returns the buffered content, appending a truncation message if capped.
func (lb *LimitedBuffer) String() string {
	s := lb.buf.String()
	if lb.capped {
		s += truncationMessage
	}
	return s
}

// Truncated returns true if output was truncated.
func (lb *LimitedBuffer) Truncated() bool {
	return lb.capped
}

// Run executes a command and captures its output.
// This is the backwards-compatible version that uses a background context.
func Run(name string, args ...string) (*Result, error) {
	return RunWithTimeout(context.Background(), name, args...)
}

// RunWithTimeout executes a command with the given context for timeout/cancellation
// and captures its output. stdout and stderr are each capped at MaxOutputBytes.
func RunWithTimeout(ctx context.Context, name string, args ...string) (*Result, error) {
	cmd := exec.CommandContext(ctx, name, args...)

	stdout := NewLimitedBuffer(MaxOutputBytes)
	stderr := NewLimitedBuffer(MaxOutputBytes)
	cmd.Stdout = io.Writer(stdout)
	cmd.Stderr = io.Writer(stderr)

	err := cmd.Run()

	result := &Result{
		Stdout: stdout.String(),
		Stderr: stderr.String(),
	}

	if err != nil {
		// Check if the context caused the error (timeout or cancellation)
		if ctx.Err() != nil {
			result.ExitCode = 124 // conventional timeout exit code
			return result, &gotkerrors.TimeoutError{Cause: ctx.Err()}
		}
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
