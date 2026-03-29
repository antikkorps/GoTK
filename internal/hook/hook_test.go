package hook

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
)

func TestRun_PreToolUseBash(t *testing.T) {
	input := `{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":"grep -rn pattern ."}}`
	var out bytes.Buffer
	if err := Run(strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}

	var result Output
	if err := json.Unmarshal(out.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if result.UpdatedInput == nil {
		t.Fatal("expected updatedInput, got nil")
	}
	if !strings.Contains(result.UpdatedInput.Command, "| ") {
		t.Errorf("expected pipe to gotk, got: %s", result.UpdatedInput.Command)
	}
	if !strings.HasPrefix(result.UpdatedInput.Command, "set -o pipefail; (grep -rn pattern .) | ") {
		t.Errorf("unexpected wrapping: %s", result.UpdatedInput.Command)
	}
}

func TestRun_NonBashTool(t *testing.T) {
	input := `{"hook_event_name":"PreToolUse","tool_name":"Read","tool_input":{"file_path":"/foo"}}`
	var out bytes.Buffer
	if err := Run(strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Errorf("expected no output for non-Bash tool, got: %s", out.String())
	}
}

func TestRun_PostToolUse(t *testing.T) {
	input := `{"hook_event_name":"PostToolUse","tool_name":"Bash","tool_input":{"command":"ls"}}`
	var out bytes.Buffer
	if err := Run(strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Errorf("expected no output for PostToolUse, got: %s", out.String())
	}
}

func TestWrapCommand_Normal(t *testing.T) {
	wrapped, ok := WrapCommand("find . -name '*.go'")
	if !ok {
		t.Fatal("expected command to be wrapped")
	}
	if !strings.Contains(wrapped, "set -o pipefail") {
		t.Error("missing pipefail")
	}
	if !strings.Contains(wrapped, "(find . -name '*.go')") {
		t.Errorf("original command not preserved: %s", wrapped)
	}
}

func TestWrapCommand_TrivialCommands(t *testing.T) {
	for _, cmd := range []string{"cd /tmp", "pwd", "echo hello", "export FOO=bar", "true", "exit"} {
		_, ok := WrapCommand(cmd)
		if ok {
			t.Errorf("trivial command %q should not be wrapped", cmd)
		}
	}
}

func TestWrapCommand_AlreadyWrapped(t *testing.T) {
	_, ok := WrapCommand("grep -rn pattern . | gotk")
	if ok {
		t.Error("already-wrapped command should not be double-wrapped")
	}

	_, ok = WrapCommand("grep -rn pattern . |gotk --stats")
	if ok {
		t.Error("already-wrapped command (no space) should not be double-wrapped")
	}
}

func TestWrapCommand_GotkInvocation(t *testing.T) {
	_, ok := WrapCommand("gotk bench --json")
	if ok {
		t.Error("gotk invocation should not be wrapped")
	}

	_, ok = WrapCommand("gotk ctx BuildChain")
	if ok {
		t.Error("gotk invocation should not be wrapped")
	}
}

func TestWrapCommand_EnvVarAssignment(t *testing.T) {
	_, ok := WrapCommand("FOO=bar")
	if ok {
		t.Error("variable assignment should not be wrapped")
	}
}

func TestWrapCommand_EnvVarPrefix(t *testing.T) {
	wrapped, ok := WrapCommand("LANG=C sort file.txt")
	if !ok {
		t.Fatal("env-prefixed command should be wrapped")
	}
	if !strings.Contains(wrapped, "LANG=C sort file.txt") {
		t.Errorf("command not preserved: %s", wrapped)
	}
}

func TestExtractFirstCommand(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"grep -rn pattern .", "grep"},
		{"/usr/bin/grep -rn pattern .", "grep"},
		{"LANG=C sort file.txt", "sort"},
		{"FOO=bar BAZ=qux cmd", "cmd"},
		{"cd /tmp", "cd"},
		{"", ""},
	}
	for _, tt := range tests {
		got := extractFirstCommand(tt.input)
		if got != tt.want {
			t.Errorf("extractFirstCommand(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestRun_EmptyCommand(t *testing.T) {
	input := `{"hook_event_name":"PreToolUse","tool_name":"Bash","tool_input":{"command":""}}`
	var out bytes.Buffer
	if err := Run(strings.NewReader(input), &out); err != nil {
		t.Fatal(err)
	}
	if out.Len() != 0 {
		t.Errorf("expected no output for empty command, got: %s", out.String())
	}
}

func TestRun_InvalidJSON(t *testing.T) {
	err := Run(strings.NewReader("not json"), &bytes.Buffer{})
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
