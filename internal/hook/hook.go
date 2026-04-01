// Package hook handles the Claude Code hook JSON protocol.
//
// When configured as a PreToolUse hook, gotk receives a JSON payload on stdin
// describing the tool invocation. For Bash commands, it wraps the command with
// "| gotk" so output is filtered before Claude sees it.
package hook

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/antikkorps/GoTK/internal/cmdclass"
)

// Input represents the JSON payload received from Claude Code on stdin.
type Input struct {
	SessionID     string    `json:"session_id"`
	HookEventName string    `json:"hook_event_name"`
	ToolName      string    `json:"tool_name"`
	ToolInput     ToolInput `json:"tool_input"`
}

// ToolInput holds the tool-specific parameters.
type ToolInput struct {
	Command string `json:"command"`
}

// Output is what we return to Claude Code on stdout.
type Output struct {
	UpdatedInput *ToolInput `json:"updatedInput,omitempty"`
}

// TrivialCommands is an alias for backward compatibility.
// Use cmdclass.TrivialCommands for new code.
var TrivialCommands = cmdclass.TrivialCommands

// Run reads a Claude Code hook payload from r, processes it, and writes
// the response to w. It returns an error only for I/O or JSON parse failures.
func Run(r io.Reader, w io.Writer) error {
	data, err := io.ReadAll(r)
	if err != nil {
		return fmt.Errorf("reading hook input: %w", err)
	}

	var input Input
	if err := json.Unmarshal(data, &input); err != nil {
		return fmt.Errorf("parsing hook JSON: %w", err)
	}

	// Only handle PreToolUse events for Bash
	if input.HookEventName != "PreToolUse" || input.ToolName != "Bash" {
		return nil // exit 0, no output = no modification
	}

	cmd := strings.TrimSpace(input.ToolInput.Command)
	if cmd == "" {
		return nil
	}

	wrapped, ok := WrapCommand(cmd)
	if !ok {
		return nil // command should not be wrapped
	}

	output := Output{
		UpdatedInput: &ToolInput{Command: wrapped},
	}

	enc := json.NewEncoder(w)
	enc.SetEscapeHTML(false)
	return enc.Encode(output)
}

// WrapCommand wraps a shell command with "| gotk" for output filtering.
// It returns the wrapped command and true, or the original command and false
// if the command should not be wrapped.
func WrapCommand(cmd string) (string, bool) {
	// Don't double-wrap
	if strings.Contains(cmd, "| gotk") || strings.Contains(cmd, "|gotk") {
		return cmd, false
	}

	// Don't wrap if gotk is already being invoked
	trimmed := strings.TrimSpace(cmd)
	if strings.HasPrefix(trimmed, "gotk ") || trimmed == "gotk" {
		return cmd, false
	}

	// Extract the first word (command name) from possibly complex shell lines.
	// For compound commands (&&, ||, ;) we check just the first command.
	firstWord := extractFirstCommand(trimmed)
	if TrivialCommands[firstWord] {
		return cmd, false
	}

	// Skip variable assignments (VAR=value)
	if strings.Contains(firstWord, "=") && !strings.Contains(firstWord, "/") {
		return cmd, false
	}

	gotkPath := findGotkBinary()
	wrapped := fmt.Sprintf("set -o pipefail; (%s) | %s", cmd, gotkPath)
	return wrapped, true
}

// extractFirstCommand returns the first command word from a shell line,
// ignoring leading environment variable assignments (KEY=val cmd ...).
func extractFirstCommand(line string) string {
	fields := strings.Fields(line)
	for _, f := range fields {
		// Skip env var assignments: FOO=bar
		if strings.Contains(f, "=") && !strings.HasPrefix(f, "-") && !strings.Contains(f, "/") {
			continue
		}
		// Strip leading path: /usr/bin/grep -> grep
		return filepath.Base(f)
	}
	if len(fields) > 0 {
		return filepath.Base(fields[0])
	}
	return ""
}

// findGotkBinary returns the absolute path to the gotk binary.
func findGotkBinary() string {
	exe, err := os.Executable()
	if err == nil {
		resolved, err := filepath.EvalSymlinks(exe)
		if err == nil {
			return resolved
		}
		return exe
	}
	return "gotk"
}
