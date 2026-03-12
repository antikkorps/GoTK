package exec

import (
	"bufio"
	"context"
	"os/exec"
	"sync"
	"syscall"
	"time"
)

// StreamResult is sent for each line of output.
type StreamResult struct {
	Line     string
	IsStderr bool
}

// RunStream executes a command and streams output line-by-line via a channel.
// The channel is closed when the command completes.
// Returns the channel and a function that waits for completion and returns the exit code.
func RunStream(name string, args ...string) (<-chan StreamResult, func() int) {
	return RunStreamWithTimeout(context.Background(), name, args...)
}

// RunStreamWithTimeout executes a command with timeout support and streams output
// line-by-line via a channel. The channel is closed when the command completes
// or the context is cancelled/timed out.
func RunStreamWithTimeout(ctx context.Context, name string, args ...string) (<-chan StreamResult, func() int) {
	ch := make(chan StreamResult, 64)

	cmd := exec.CommandContext(ctx, name, args...)

	stdoutPipe, err := cmd.StdoutPipe()
	if err != nil {
		close(ch)
		return ch, func() int { return 1 }
	}

	stderrPipe, err := cmd.StderrPipe()
	if err != nil {
		close(ch)
		return ch, func() int { return 1 }
	}

	if err := cmd.Start(); err != nil {
		close(ch)
		return ch, func() int { return 1 }
	}

	var wg sync.WaitGroup
	wg.Add(2)

	// Read stdout lines.
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stdoutPipe)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			ch <- StreamResult{Line: scanner.Text(), IsStderr: false}
		}
	}()

	// Read stderr lines.
	go func() {
		defer wg.Done()
		scanner := bufio.NewScanner(stderrPipe)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for scanner.Scan() {
			ch <- StreamResult{Line: scanner.Text(), IsStderr: true}
		}
	}()

	// Close the channel after both readers finish.
	var cmdErr error
	var exitOnce sync.Once
	var exitCode int
	done := make(chan struct{})

	go func() {
		wg.Wait()
		cmdErr = cmd.Wait()
		close(ch)
		close(done)
	}()

	waitFn := func() int {
		exitOnce.Do(func() {
			<-done
			if cmdErr == nil {
				exitCode = 0
				return
			}
			// Check for context cancellation/timeout
			if ctx.Err() != nil {
				exitCode = 124 // conventional timeout exit code
				return
			}
			if exitErr, ok := cmdErr.(*exec.ExitError); ok {
				if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
					exitCode = status.ExitStatus()
					return
				}
				exitCode = 1
				return
			}
			exitCode = 1
		})
		return exitCode
	}

	return ch, waitFn
}

// StreamTimeout is the default timeout for streaming commands.
var StreamTimeout = 5 * time.Minute
