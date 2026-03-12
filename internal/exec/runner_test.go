package exec

import (
	"runtime"
	"strings"
	"testing"
)

func TestRun(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tests use Unix commands")
	}

	t.Run("successful command echo", func(t *testing.T) {
		result, err := Run("echo", "hello", "world")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", result.ExitCode)
		}
		got := strings.TrimSpace(result.Stdout)
		if got != "hello world" {
			t.Errorf("expected stdout %q, got %q", "hello world", got)
		}
	})

	t.Run("command with non-zero exit code", func(t *testing.T) {
		// grep returns exit code 1 when no match found
		result, err := Run("grep", "impossiblepatternXYZ123", "/dev/null")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result.ExitCode == 0 {
			t.Error("expected non-zero exit code for grep with no match")
		}
	})

	t.Run("command not found error", func(t *testing.T) {
		_, err := Run("nonexistent_command_xyz_12345")
		if err == nil {
			t.Error("expected error for command not found")
		}
	})

	t.Run("command captures stderr", func(t *testing.T) {
		// Use sh -c to write to stderr
		result, err := Run("sh", "-c", "echo error_msg >&2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result.Stderr, "error_msg") {
			t.Errorf("expected stderr to contain 'error_msg', got %q", result.Stderr)
		}
	})

	t.Run("command with both stdout and stderr", func(t *testing.T) {
		result, err := Run("sh", "-c", "echo out; echo err >&2")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(result.Stdout, "out") {
			t.Errorf("expected stdout to contain 'out', got %q", result.Stdout)
		}
		if !strings.Contains(result.Stderr, "err") {
			t.Errorf("expected stderr to contain 'err', got %q", result.Stderr)
		}
	})

	t.Run("result struct fields", func(t *testing.T) {
		result, err := Run("echo", "test")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if result == nil {
			t.Fatal("expected non-nil result")
		}
		// Verify all fields are accessible
		_ = result.Stdout
		_ = result.Stderr
		_ = result.ExitCode
	})
}
