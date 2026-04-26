package exec

import (
	"context"
	"runtime"
	"strings"
	"testing"
)

func TestRunStream_Echo(t *testing.T) {
	ch, wait := RunStream(context.Background(), "echo", "hello world")

	var lines []StreamResult
	for r := range ch {
		lines = append(lines, r)
	}

	code := wait()
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0].Line != "hello world" {
		t.Errorf("expected 'hello world', got %q", lines[0].Line)
	}
	if lines[0].IsStderr {
		t.Error("expected stdout, got stderr")
	}
}

func TestRunStream_MultipleLines(t *testing.T) {
	if runtime.GOOS == "windows" {
		// Windows printf doesn't interpret backslash escapes — "\n" stays
		// literal, so the command emits a single line. Skipping here keeps
		// the test POSIX-only; line splitting itself is exercised by
		// per-package unit tests on bufio.Scanner.
		t.Skip("printf escape semantics differ on Windows")
	}
	// Use printf to emit multiple lines.
	ch, wait := RunStream(context.Background(), "printf", "line1\nline2\nline3\n")

	var lines []string
	for r := range ch {
		if !r.IsStderr {
			lines = append(lines, r.Line)
		}
	}

	code := wait()
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}

	expected := []string{"line1", "line2", "line3"}
	for i, want := range expected {
		if lines[i] != want {
			t.Errorf("line %d: expected %q, got %q", i, want, lines[i])
		}
	}
}

func TestRunStream_Stderr(t *testing.T) {
	// Write to stderr using sh -c.
	ch, wait := RunStream(context.Background(), "sh", "-c", "echo err >&2")

	var stderrLines []string
	for r := range ch {
		if r.IsStderr {
			stderrLines = append(stderrLines, r.Line)
		}
	}

	wait()

	if len(stderrLines) == 0 {
		t.Fatal("expected stderr output, got none")
	}
	if stderrLines[0] != "err" {
		t.Errorf("expected 'err', got %q", stderrLines[0])
	}
}

func TestRunStream_NonZeroExit(t *testing.T) {
	ch, wait := RunStream(context.Background(), "sh", "-c", "exit 42")

	// Drain channel.
	for range ch {
	}

	code := wait()
	if code != 42 {
		t.Errorf("expected exit code 42, got %d", code)
	}
}

func TestRunStream_MixedOutput(t *testing.T) {
	// Produce both stdout and stderr.
	ch, wait := RunStream(context.Background(), "sh", "-c", "echo out; echo err >&2; echo out2")

	var stdout, stderr []string
	for r := range ch {
		if r.IsStderr {
			stderr = append(stderr, r.Line)
		} else {
			stdout = append(stdout, r.Line)
		}
	}

	code := wait()
	if code != 0 {
		t.Fatalf("expected exit code 0, got %d", code)
	}

	stdoutJoined := strings.Join(stdout, "\n")
	if !strings.Contains(stdoutJoined, "out") {
		t.Errorf("expected stdout to contain 'out', got %q", stdoutJoined)
	}

	if len(stderr) == 0 {
		t.Error("expected stderr output")
	}
}

func TestRunStream_PartialLine(t *testing.T) {
	// printf without trailing newline -- scanner should still emit it.
	ch, wait := RunStream(context.Background(), "printf", "no-newline")

	var lines []string
	for r := range ch {
		if !r.IsStderr {
			lines = append(lines, r.Line)
		}
	}

	wait()

	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0] != "no-newline" {
		t.Errorf("expected 'no-newline', got %q", lines[0])
	}
}

func TestRunStream_CommandNotFound(t *testing.T) {
	ch, wait := RunStream(context.Background(), "__nonexistent_command_xyz__")

	for range ch {
	}

	code := wait()
	if runtime.GOOS != "windows" && code == 0 {
		t.Error("expected non-zero exit code for missing command")
	}
}

func TestRunStream_WaitIdempotent(t *testing.T) {
	ch, wait := RunStream(context.Background(), "echo", "test")
	for range ch {
	}

	code1 := wait()
	code2 := wait()
	if code1 != code2 {
		t.Errorf("wait() should be idempotent: got %d then %d", code1, code2)
	}
}
