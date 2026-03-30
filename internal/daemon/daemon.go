// Package daemon implements GoTK's daemon mode — a filtered shell session
// that transparently intercepts all commands and pipes their output through
// GoTK for noise reduction.
package daemon

import (
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/hook"
)

// InteractiveCommands are programs that need direct terminal access
// and should not have their output piped through gotk.
var InteractiveCommands = map[string]bool{
	"vim":         true,
	"vi":          true,
	"nvim":        true,
	"nano":        true,
	"emacs":       true,
	"less":        true,
	"more":        true,
	"man":         true,
	"top":         true,
	"htop":        true,
	"btop":        true,
	"watch":       true,
	"ssh":         true,
	"mosh":        true,
	"tmux":        true,
	"screen":      true,
	"fzf":         true,
	"nnn":         true,
	"ranger":      true,
	"mc":          true,
	"lazygit":     true,
	"lazydocker":  true,
	"k9s":         true,
}

// ShouldSkip returns true if the command should not be wrapped by the daemon.
func ShouldSkip(cmd string) bool {
	cmd = strings.TrimSpace(cmd)
	if cmd == "" {
		return true
	}

	firstWord := extractFirstWord(cmd)
	if firstWord == "" {
		return true
	}

	// Skip gotk itself
	if firstWord == "gotk" {
		return true
	}

	// Skip trivial and interactive commands
	if hook.TrivialCommands[firstWord] {
		return true
	}
	if InteractiveCommands[firstWord] {
		return true
	}

	// Skip if already piped through gotk
	if strings.Contains(cmd, "| gotk") || strings.Contains(cmd, "|gotk") {
		return true
	}

	return false
}

// extractFirstWord returns the base name of the first non-assignment word.
func extractFirstWord(cmd string) string {
	for _, word := range strings.Fields(cmd) {
		// Skip env var assignments: FOO=bar
		if strings.Contains(word, "=") && !strings.Contains(word, "/") {
			continue
		}
		return filepath.Base(word)
	}
	return ""
}

// Start launches a daemon shell session. It spawns the user's shell with
// custom init scripts that intercept commands and pipe them through gotk.
func Start(cfg *config.Config) error {
	// Prevent nesting
	if os.Getenv("GOTK_DAEMON") == "1" {
		return fmt.Errorf("already inside a gotk daemon session (GOTK_DAEMON=1)")
	}

	shell := detectShell()
	shellName := filepath.Base(shell)

	gotkBin, err := findGotkBin()
	if err != nil {
		return fmt.Errorf("cannot locate gotk binary: %w", err)
	}

	sessionID := fmt.Sprintf("%d-%d", time.Now().UnixMilli(), os.Getpid())

	// Create temp init file
	initFile, err := writeInitFile(shellName, gotkBin, sessionID)
	if err != nil {
		return fmt.Errorf("cannot create init file: %w", err)
	}
	defer os.Remove(initFile)

	fmt.Fprintf(os.Stderr, "[gotk] starting daemon session (shell: %s)\n", shellName)
	fmt.Fprintf(os.Stderr, "[gotk] all command output will be filtered. type 'gotk exit' to stop.\n\n")

	// Enable measurement for the session
	cfg.Measure.Enabled = true

	// Build the shell command with custom init
	var cmd *exec.Cmd
	switch shellName {
	case "zsh":
		// ZDOTDIR trick: zsh reads $ZDOTDIR/.zshrc on startup
		tmpDir := filepath.Dir(initFile)
		cmd = exec.Command(shell)
		cmd.Env = append(os.Environ(),
			"ZDOTDIR="+tmpDir,
			"GOTK_DAEMON=1",
			"GOTK_SESSION_ID="+sessionID,
			"GOTK_BIN="+gotkBin,
		)
	case "bash":
		cmd = exec.Command(shell, "--rcfile", initFile, "-i")
		cmd.Env = append(os.Environ(),
			"GOTK_DAEMON=1",
			"GOTK_SESSION_ID="+sessionID,
			"GOTK_BIN="+gotkBin,
		)
	default:
		return fmt.Errorf("unsupported shell: %s (supported: bash, zsh)", shellName)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Forward signals to the child shell
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range sigCh {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()

	err = cmd.Run()
	signal.Stop(sigCh)
	close(sigCh)

	// Exit code from the shell
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
				os.Exit(status.ExitStatus())
			}
		}
	}

	return nil
}

// Status prints whether we're inside a daemon session.
func Status() {
	if os.Getenv("GOTK_DAEMON") == "1" {
		sessionID := os.Getenv("GOTK_SESSION_ID")
		fmt.Fprintf(os.Stderr, "GoTK daemon: active (session %s)\n", sessionID)
	} else {
		fmt.Fprintf(os.Stderr, "GoTK daemon: not active\n")
	}
}

// Init prints the shell init code for manual eval.
func Init(shell string) error {
	gotkBin, err := findGotkBin()
	if err != nil {
		return err
	}
	sessionID := fmt.Sprintf("%d-%d", time.Now().UnixMilli(), os.Getpid())
	script := generateScript(filepath.Base(shell), gotkBin, sessionID)
	fmt.Print(script)
	return nil
}

func detectShell() string {
	// Check GOTK_SHELL first (explicit override for daemon's child shell)
	if s := os.Getenv("GOTK_SHELL"); s != "" {
		return s
	}

	// Use SHELL env var
	if s := os.Getenv("SHELL"); s != "" {
		base := filepath.Base(s)
		if base != "gotk" {
			return s
		}
	}

	// Fallback
	for _, sh := range []string{"/bin/zsh", "/bin/bash"} {
		if _, err := os.Stat(sh); err == nil {
			return sh
		}
	}
	return "/bin/sh"
}

func findGotkBin() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return "", err
	}
	resolved, err := filepath.EvalSymlinks(exe)
	if err != nil {
		return "", fmt.Errorf("cannot resolve symlinks for %q: %w", exe, err)
	}

	// Ensure the path is absolute
	if !filepath.IsAbs(resolved) {
		return "", fmt.Errorf("gotk binary path %q is not absolute", resolved)
	}

	// Verify the file exists and is a regular file (not a directory or device)
	info, err := os.Stat(resolved)
	if err != nil {
		return "", fmt.Errorf("gotk binary not found at %q: %w", resolved, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("gotk binary path %q is not a regular file", resolved)
	}

	return resolved, nil
}

func generateScript(shellName, gotkBin, sessionID string) string {
	var tmpl string
	switch shellName {
	case "zsh":
		tmpl = zshInit
	case "bash":
		tmpl = bashInit
	default:
		return ""
	}

	s := strings.ReplaceAll(tmpl, "GOTK_BIN_PLACEHOLDER", gotkBin)
	s = strings.ReplaceAll(s, "SESSION_ID_PLACEHOLDER", sessionID)
	return s
}

func writeInitFile(shellName, gotkBin, sessionID string) (string, error) {
	script := generateScript(shellName, gotkBin, sessionID)
	if script == "" {
		return "", fmt.Errorf("unsupported shell: %s", shellName)
	}

	// For zsh, the file must be named .zshrc in a temp ZDOTDIR
	var filename string
	switch shellName {
	case "zsh":
		tmpDir, err := os.MkdirTemp("", "gotk-daemon-*")
		if err != nil {
			return "", err
		}
		filename = filepath.Join(tmpDir, ".zshrc")
		// Replace ZDOTDIR placeholder with original value
		origZdotdir := os.Getenv("ZDOTDIR")
		if origZdotdir == "" {
			origZdotdir = os.Getenv("HOME")
		}
		script = strings.ReplaceAll(script, "ORIG_ZDOTDIR_PLACEHOLDER", origZdotdir)
	case "bash":
		f, err := os.CreateTemp("", "gotk-daemon-*.bashrc")
		if err != nil {
			return "", err
		}
		filename = f.Name()
		f.Close()
	}

	if err := os.WriteFile(filename, []byte(script), 0600); err != nil {
		return "", err
	}
	return filename, nil
}
