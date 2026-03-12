package proxy

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"

	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/filter"
)

// BuildChain creates a filter chain controlled by config and command type.
func BuildChain(cfg *config.Config, cmdType detect.CmdType) *filter.Chain {
	chain := filter.NewChain()

	if cfg.Filters.StripANSI {
		chain.Add(filter.StripANSI)
	}
	if cfg.Filters.NormalizeWhitespace {
		chain.Add(filter.NormalizeWhitespace)
	}
	if cfg.Filters.Dedup {
		chain.Add(filter.Dedup)
	}

	// Command-specific filters (CompressPaths is included via detect.FiltersFor
	// for certain types, so we only add it for generic if enabled)
	cmdFilters := detect.FiltersFor(cmdType)
	for _, f := range cmdFilters {
		chain.Add(f)
	}

	if cfg.Filters.TrimDecorative {
		chain.Add(filter.TrimEmpty)
	}

	// Stack trace compression runs on all output (generic filter) since
	// panics/tracebacks can appear in any command's stderr captured as stdout.
	chain.Add(filter.CompressStackTraces)

	if cfg.Filters.Truncate {
		chain.Add(filter.Truncate)
	}

	return chain
}

// passthrough returns true if filtering should be skipped.
func passthrough() bool {
	return os.Getenv("GOTK_PASSTHROUGH") == "1"
}

// RunCommand executes a single command string through the shell, filters
// stdout, and passes stderr through unmodified. Returns the exit code.
func RunCommand(cfg *config.Config, command string) int {
	shell := findShell()
	cmd := exec.Command(shell, "-c", command)
	cmd.Stdin = os.Stdin
	cmd.Stderr = os.Stderr

	if passthrough() {
		cmd.Stdout = os.Stdout
		err := cmd.Run()
		return exitCode(err)
	}

	// Capture stdout for filtering
	stdout, err := cmd.Output()
	if err != nil {
		// Still process any output we got
	}

	raw := string(stdout)
	if raw == "" {
		return exitCode(err)
	}

	// Detect command type from the first word
	parts := strings.Fields(command)
	cmdType := detect.CmdGeneric
	if len(parts) > 0 {
		cmdType = detect.Identify(parts[0])
		// Check custom command mappings
		if mapped, ok := cfg.Commands[parts[0]]; ok {
			cmdType = detect.Identify(mapped)
		}
	}

	chain := BuildChain(cfg, cmdType)
	cleaned := chain.Apply(raw)
	fmt.Fprint(os.Stdout, cleaned)

	return exitCode(err)
}

// RunShell starts an interactive-ish proxy shell that reads commands from
// stdin, executes them, and writes filtered output to stdout. This mode
// is designed for LLM tool integration where the caller sets
// SHELL=/path/to/gotk and sends commands one at a time.
func RunShell(cfg *config.Config) int {
	scanner := bufio.NewScanner(os.Stdin)

	// Increase scanner buffer for large commands
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Handle exit/quit
		trimmed := strings.TrimSpace(line)
		if trimmed == "exit" || trimmed == "quit" {
			break
		}

		code := RunCommand(cfg, line)
		_ = code // We don't exit the shell loop on non-zero
	}

	return 0
}

// findShell returns a real shell to use for executing commands.
// Avoids recursion by not returning gotk itself.
func findShell() string {
	// Check GOTK_SHELL first (explicit override)
	if s := os.Getenv("GOTK_SHELL"); s != "" {
		return s
	}

	// Don't use SHELL if it points to gotk (would recurse)
	if s := os.Getenv("SHELL"); s != "" {
		base := s
		if idx := strings.LastIndexByte(s, '/'); idx >= 0 {
			base = s[idx+1:]
		}
		if base != "gotk" {
			return s
		}
	}

	// Fallback chain
	for _, sh := range []string{"/bin/bash", "/bin/sh"} {
		if _, err := os.Stat(sh); err == nil {
			return sh
		}
	}

	return "sh"
}

func exitCode(err error) int {
	if err == nil {
		return 0
	}
	if exitErr, ok := err.(*exec.ExitError); ok {
		if status, ok := exitErr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus()
		}
		return 1
	}
	return 1
}
