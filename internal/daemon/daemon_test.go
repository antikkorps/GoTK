package daemon

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShouldSkip_TrivialCommands(t *testing.T) {
	trivial := []string{
		"cd /tmp", "pwd", "echo hello", "export FOO=bar",
		"true", "false", "exit", "source ~/.bashrc",
	}
	for _, cmd := range trivial {
		if !ShouldSkip(cmd) {
			t.Errorf("ShouldSkip(%q) = false, want true (trivial)", cmd)
		}
	}
}

func TestShouldSkip_InteractiveCommands(t *testing.T) {
	interactive := []string{
		"vim file.go", "nvim .", "less output.log",
		"top", "htop", "ssh user@host", "tmux new -s work",
		"fzf", "man ls",
	}
	for _, cmd := range interactive {
		if !ShouldSkip(cmd) {
			t.Errorf("ShouldSkip(%q) = false, want true (interactive)", cmd)
		}
	}
}

func TestShouldSkip_FilterableCommands(t *testing.T) {
	filterable := []string{
		"grep -rn pattern .", "find . -name '*.go'",
		"go test ./...", "git log --oneline -20",
		"ls -la", "make build", "cargo build",
		"docker ps", "kubectl get pods",
	}
	for _, cmd := range filterable {
		if ShouldSkip(cmd) {
			t.Errorf("ShouldSkip(%q) = true, want false (should filter)", cmd)
		}
	}
}

func TestShouldSkip_GotkItself(t *testing.T) {
	if !ShouldSkip("gotk bench --json") {
		t.Error("gotk commands should be skipped")
	}
	if !ShouldSkip("gotk ctx pattern -t go") {
		t.Error("gotk commands should be skipped")
	}
}

func TestShouldSkip_AlreadyPiped(t *testing.T) {
	if !ShouldSkip("grep -rn pattern . | gotk") {
		t.Error("already piped commands should be skipped")
	}
	if !ShouldSkip("make build |gotk --stats") {
		t.Error("already piped commands should be skipped")
	}
}

func TestShouldSkip_EmptyCommand(t *testing.T) {
	if !ShouldSkip("") {
		t.Error("empty command should be skipped")
	}
	if !ShouldSkip("   ") {
		t.Error("whitespace-only command should be skipped")
	}
}

func TestShouldSkip_EnvPrefix(t *testing.T) {
	if ShouldSkip("LANG=C sort file.txt") {
		t.Error("env-prefixed command should not be skipped")
	}
	if ShouldSkip("CC=gcc make") {
		t.Error("env-prefixed command should not be skipped")
	}
}

func TestExtractFirstWord(t *testing.T) {
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
		got := extractFirstWord(tt.input)
		if got != tt.want {
			t.Errorf("extractFirstWord(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestGenerateScript_Zsh(t *testing.T) {
	script := generateScript("zsh", "/usr/local/bin/gotk", "test-session-123")
	if script == "" {
		t.Fatal("expected non-empty zsh script")
	}
	if !strings.Contains(script, "/usr/local/bin/gotk") {
		t.Error("script should contain gotk binary path")
	}
	if !strings.Contains(script, "test-session-123") {
		t.Error("script should contain session ID")
	}
	if !strings.Contains(script, "accept-line") {
		t.Error("zsh script should override accept-line")
	}
	if !strings.Contains(script, "GOTK_DAEMON=1") {
		t.Error("script should set GOTK_DAEMON")
	}
	if !strings.Contains(script, `"gotk exit"`) && !strings.Contains(script, `"exit"`) {
		t.Error("zsh script should define gotk wrapper with exit support")
	}
}

func TestGenerateScript_Bash(t *testing.T) {
	script := generateScript("bash", "/usr/local/bin/gotk", "test-session-456")
	if script == "" {
		t.Fatal("expected non-empty bash script")
	}
	if !strings.Contains(script, "/usr/local/bin/gotk") {
		t.Error("script should contain gotk binary path")
	}
	if !strings.Contains(script, "test-session-456") {
		t.Error("script should contain session ID")
	}
	if !strings.Contains(script, "extdebug") {
		t.Error("bash script should use extdebug")
	}
	if !strings.Contains(script, "DEBUG") {
		t.Error("bash script should set DEBUG trap")
	}
}

func TestGenerateScript_Unknown(t *testing.T) {
	script := generateScript("fish", "/usr/local/bin/gotk", "id")
	if script != "" {
		t.Error("unsupported shell should return empty string")
	}
}

func TestWriteInitFile_Zsh(t *testing.T) {
	path, err := writeInitFile("zsh", "/usr/local/bin/gotk", "session-1")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Dir(path))

	if filepath.Base(path) != ".zshrc" {
		t.Errorf("zsh init file should be named .zshrc, got %s", filepath.Base(path))
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "GOTK_DAEMON=1") {
		t.Error("init file should contain GOTK_DAEMON")
	}
}

func TestWriteInitFile_Bash(t *testing.T) {
	path, err := writeInitFile("bash", "/usr/local/bin/gotk", "session-2")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path)

	if !strings.Contains(path, "gotk-daemon") {
		t.Errorf("bash init file should have gotk-daemon prefix, got %s", path)
	}

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "GOTK_DAEMON=1") {
		t.Error("init file should contain GOTK_DAEMON")
	}
}

func TestPrintSummary_NoEntries(t *testing.T) {
	var buf bytes.Buffer
	PrintSummary(&buf, "/nonexistent/path", "no-session")
	// Should not panic, may produce no output or "no commands" message
}
