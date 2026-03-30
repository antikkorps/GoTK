package daemon

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/antikkorps/GoTK/internal/config"
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
	defer os.RemoveAll(filepath.Dir(path)) //nolint:errcheck

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
	defer os.Remove(path) //nolint:errcheck

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

func TestDetectShell_DefaultShell(t *testing.T) {
	// Save and restore env vars
	origShell := os.Getenv("SHELL")
	origGotkShell := os.Getenv("GOTK_SHELL")
	defer func() {
		os.Setenv("SHELL", origShell) //nolint:errcheck
		if origGotkShell != "" {
			os.Setenv("GOTK_SHELL", origGotkShell) //nolint:errcheck
		} else {
			os.Unsetenv("GOTK_SHELL") //nolint:errcheck
		}
	}()

	// Test GOTK_SHELL takes priority
	os.Setenv("GOTK_SHELL", "/bin/zsh") //nolint:errcheck
	if got := detectShell(); got != "/bin/zsh" {
		t.Errorf("detectShell() with GOTK_SHELL=/bin/zsh = %q, want /bin/zsh", got)
	}

	// Test SHELL env var
	os.Unsetenv("GOTK_SHELL")           //nolint:errcheck
	os.Setenv("SHELL", "/usr/bin/bash") //nolint:errcheck
	if got := detectShell(); got != "/usr/bin/bash" {
		t.Errorf("detectShell() with SHELL=/usr/bin/bash = %q, want /usr/bin/bash", got)
	}

	// Test SHELL=gotk is skipped (avoid recursion)
	os.Setenv("SHELL", "/usr/local/bin/gotk") //nolint:errcheck
	got := detectShell()
	if filepath.Base(got) == "gotk" {
		t.Errorf("detectShell() should not return gotk as shell, got %q", got)
	}
}

func TestFindGotkBin(t *testing.T) {
	// findGotkBin uses os.Executable(), which should return the test binary path
	path, err := findGotkBin()
	if err != nil {
		t.Fatalf("findGotkBin() error: %v", err)
	}
	if path == "" {
		t.Error("findGotkBin() returned empty path")
	}
	if !filepath.IsAbs(path) {
		t.Errorf("findGotkBin() should return absolute path, got %q", path)
	}
}

func TestWriteInitFile_Unsupported(t *testing.T) {
	_, err := writeInitFile("fish", "/usr/bin/gotk", "session-x")
	if err == nil {
		t.Error("writeInitFile with unsupported shell should return error")
	}
}

func TestWriteInitFile_Permissions(t *testing.T) {
	// Test zsh init file permissions
	path, err := writeInitFile("zsh", "/usr/local/bin/gotk", "session-perm")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Dir(path)) //nolint:errcheck

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("init file permissions = %o, want 0600", perm)
	}
}

func TestStatus_NotActive(t *testing.T) {
	origDaemon := os.Getenv("GOTK_DAEMON")
	defer os.Setenv("GOTK_DAEMON", origDaemon) //nolint:errcheck

	os.Unsetenv("GOTK_DAEMON") //nolint:errcheck
	// Should not panic
	Status()
}

func TestStatus_Active(t *testing.T) {
	origDaemon := os.Getenv("GOTK_DAEMON")
	origSession := os.Getenv("GOTK_SESSION_ID")
	defer func() {
		os.Setenv("GOTK_DAEMON", origDaemon)      //nolint:errcheck
		os.Setenv("GOTK_SESSION_ID", origSession) //nolint:errcheck
	}()

	os.Setenv("GOTK_DAEMON", "1")                //nolint:errcheck
	os.Setenv("GOTK_SESSION_ID", "test-session") //nolint:errcheck
	// Should not panic
	Status()
}

func TestPrintSummary_WithEntries(t *testing.T) {
	// Create a temp measure log with entries
	tmpFile, err := os.CreateTemp("", "gotk_summary_*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name()) //nolint:errcheck

	sessionID := "test-summary-session"
	tmpFile.WriteString(`{"session":"test-summary-session","ts":"2026-03-30T10:00:00Z","cmd":"grep -rn test .","raw_tokens":1000,"tokens_saved":800,"duration_us":5000}` + "\n") //nolint:errcheck
	tmpFile.WriteString(`{"session":"test-summary-session","ts":"2026-03-30T10:01:00Z","cmd":"go test ./...","raw_tokens":2000,"tokens_saved":1500,"duration_us":3000}` + "\n")  //nolint:errcheck
	tmpFile.WriteString(`{"session":"other-session","ts":"2026-03-30T10:02:00Z","cmd":"ls","raw_tokens":100,"tokens_saved":50,"duration_us":1000}` + "\n")                       //nolint:errcheck
	tmpFile.Close()                                                                                                                                                              //nolint:errcheck

	var buf bytes.Buffer
	PrintSummary(&buf, tmpFile.Name(), sessionID)

	output := buf.String()
	if !strings.Contains(output, "commands filtered: 2") {
		t.Errorf("summary should show 2 commands filtered, got: %s", output)
	}
	if !strings.Contains(output, "tokens saved") {
		t.Errorf("summary should show tokens saved, got: %s", output)
	}
}

func TestShouldSkip_PathWithSlash(t *testing.T) {
	// Commands with paths containing / should not be treated as env vars
	if ShouldSkip("/usr/bin/grep pattern .") {
		t.Error("/usr/bin/grep should not be skipped")
	}
}

func TestInit_Zsh(t *testing.T) {
	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Init("zsh")

	w.Close() //nolint:errcheck
	os.Stdout = old

	if err != nil {
		t.Fatalf("Init(zsh) error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r) //nolint:errcheck
	output := buf.String()

	if !strings.Contains(output, "GOTK_DAEMON=1") {
		t.Error("Init should output script with GOTK_DAEMON")
	}
	if !strings.Contains(output, "accept-line") {
		t.Error("zsh init should contain accept-line widget")
	}
}

func TestInit_Bash(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	err := Init("bash")

	w.Close() //nolint:errcheck
	os.Stdout = old

	if err != nil {
		t.Fatalf("Init(bash) error: %v", err)
	}

	var buf bytes.Buffer
	buf.ReadFrom(r) //nolint:errcheck
	output := buf.String()

	if !strings.Contains(output, "extdebug") {
		t.Error("bash init should contain extdebug")
	}
}

func TestDetectShell_Fallback(t *testing.T) {
	origShell := os.Getenv("SHELL")
	origGotkShell := os.Getenv("GOTK_SHELL")
	defer func() {
		os.Setenv("SHELL", origShell) //nolint:errcheck
		if origGotkShell != "" {
			os.Setenv("GOTK_SHELL", origGotkShell) //nolint:errcheck
		} else {
			os.Unsetenv("GOTK_SHELL") //nolint:errcheck
		}
	}()

	os.Unsetenv("GOTK_SHELL") //nolint:errcheck
	os.Unsetenv("SHELL")      //nolint:errcheck

	got := detectShell()
	if got == "" {
		t.Error("detectShell() should return a fallback shell")
	}
}

func TestShouldSkip_OnlyEnvVars(t *testing.T) {
	// Command with only env var assignments, no actual command
	if !ShouldSkip("FOO=bar") {
		t.Error("command with only env vars should be skipped (no command word)")
	}
}

func TestStart_NestedPrevention(t *testing.T) {
	origDaemon := os.Getenv("GOTK_DAEMON")
	defer os.Setenv("GOTK_DAEMON", origDaemon) //nolint:errcheck

	os.Setenv("GOTK_DAEMON", "1") //nolint:errcheck

	cfg := &config.Config{}
	err := Start(cfg)
	if err == nil {
		t.Fatal("Start() inside daemon session should return error")
	}
	if !strings.Contains(err.Error(), "already inside") {
		t.Errorf("error should mention 'already inside', got: %v", err)
	}
}

func TestPrepareSession_Success(t *testing.T) {
	origDaemon := os.Getenv("GOTK_DAEMON")
	defer os.Setenv("GOTK_DAEMON", origDaemon) //nolint:errcheck
	os.Unsetenv("GOTK_DAEMON")                 //nolint:errcheck

	setup, err := prepareSession()
	if err != nil {
		t.Fatalf("prepareSession() error: %v", err)
	}
	defer func() {
		if setup.initFile != "" {
			os.Remove(setup.initFile) //nolint:errcheck
			// Also try to remove parent dir (zsh case)
			os.Remove(filepath.Dir(setup.initFile)) //nolint:errcheck
		}
	}()

	if setup.shell == "" {
		t.Error("shell should not be empty")
	}
	if setup.shellName == "" {
		t.Error("shellName should not be empty")
	}
	if setup.gotkBin == "" {
		t.Error("gotkBin should not be empty")
	}
	if setup.sessionID == "" {
		t.Error("sessionID should not be empty")
	}
	if setup.initFile == "" {
		t.Error("initFile should not be empty")
	}

	// Verify init file exists and has correct permissions
	info, err := os.Stat(setup.initFile)
	if err != nil {
		t.Fatalf("init file not found: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("init file permissions = %o, want 0600", info.Mode().Perm())
	}
}

func TestPrepareSession_Nested(t *testing.T) {
	origDaemon := os.Getenv("GOTK_DAEMON")
	defer os.Setenv("GOTK_DAEMON", origDaemon) //nolint:errcheck

	os.Setenv("GOTK_DAEMON", "1") //nolint:errcheck

	_, err := prepareSession()
	if err == nil {
		t.Fatal("prepareSession() inside daemon should fail")
	}
}

func TestPrintSummary_ZeroTokens(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "gotk_summary_zero_*.jsonl")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name()) //nolint:errcheck

	// Entry with zero raw tokens
	tmpFile.WriteString(`{"session":"zero-session","ts":"2026-03-30T10:00:00Z","cmd":"echo","raw_tokens":0,"tokens_saved":0,"duration_us":100}` + "\n") //nolint:errcheck
	tmpFile.Close()                                                                                                                                     //nolint:errcheck

	var buf bytes.Buffer
	PrintSummary(&buf, tmpFile.Name(), "zero-session")

	output := buf.String()
	if !strings.Contains(output, "commands filtered: 1") {
		t.Errorf("should show 1 command, got: %s", output)
	}
}

func TestWriteInitFile_BashPermissions(t *testing.T) {
	path, err := writeInitFile("bash", "/usr/local/bin/gotk", "session-bash-perm")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(path) //nolint:errcheck

	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("bash init file permissions = %o, want 0600", perm)
	}
}

func TestWriteInitFile_ZshOrigZdotdir(t *testing.T) {
	origZdotdir := os.Getenv("ZDOTDIR")
	defer func() {
		if origZdotdir != "" {
			os.Setenv("ZDOTDIR", origZdotdir) //nolint:errcheck
		} else {
			os.Unsetenv("ZDOTDIR") //nolint:errcheck
		}
	}()

	os.Setenv("ZDOTDIR", "/custom/zdotdir") //nolint:errcheck
	path, err := writeInitFile("zsh", "/usr/local/bin/gotk", "session-zdot")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(filepath.Dir(path)) //nolint:errcheck

	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "/custom/zdotdir") {
		t.Error("zsh init should contain the original ZDOTDIR value")
	}
}
