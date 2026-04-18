package proxy

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"

	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/detect"
	gotkexec "github.com/antikkorps/GoTK/internal/exec"
	"github.com/antikkorps/GoTK/internal/filter"
)

// BuildChain creates a filter chain controlled by config, command type, and max lines.
func BuildChain(cfg *config.Config, cmdType detect.CmdType, maxLines int) *filter.Chain {
	chain := filter.NewChain()

	// Blacklist: remove matching lines early, before any other processing
	if len(cfg.Rules.AlwaysRemove) > 0 {
		chain.AddNamed("always_remove", filter.RemoveByRules(cfg.Rules.AlwaysRemove))
	}

	if cfg.Filters.StripANSI {
		chain.AddNamed("strip_ansi", filter.StripANSI)
	}
	if cfg.Filters.NormalizeWhitespace {
		chain.AddNamed("normalize_whitespace", filter.NormalizeWhitespace)
	}
	if cfg.Filters.Dedup {
		chain.AddNamed("dedup", filter.Dedup)
	}

	// Command-specific filters (CompressPaths is included via detect.FiltersFor
	// for certain types, so we only add it for generic if enabled)
	cmdFilters := detect.FiltersFor(cmdType)
	for _, f := range cmdFilters {
		chain.AddNamed(cmdType.String(), f)
	}

	if cfg.Filters.TrimDecorative {
		chain.AddNamed("trim_decorative", filter.TrimEmpty)
	}

	// Stack trace compression runs on all output (generic filter) since
	// panics/tracebacks can appear in any command's stderr captured as stdout.
	chain.AddNamed("stack_traces", filter.CompressStackTraces)

	// Secret redaction -- always runs before truncation to ensure no secrets
	// leak even in truncated output.
	if cfg.Security.RedactSecrets {
		chain.AddNamed("redact_secrets", filter.RedactSecrets)
	}

	// Summary goes before truncation so LLM always sees the overview
	chain.AddNamed("summarize", filter.Summarize)

	if cfg.Filters.Truncate {
		mode := filter.ParseAutoEscalate(cfg.General.AutoEscalate)
		chain.AddNamed("truncate", filter.TruncateWithEscalation(maxLines, mode, cfg.General.EscalateWindow))
	}

	return chain
}

// BuildChainWithKeep creates a filter chain and wraps it to enforce always_keep rules.
// Use this instead of BuildChain when the original input is available.
func BuildChainWithKeep(cfg *config.Config, cmdType detect.CmdType, maxLines int, originalInput string) *filter.Chain {
	chain := BuildChain(cfg, cmdType, maxLines)

	// Whitelist: restore any matching lines that were removed by the chain
	if len(cfg.Rules.AlwaysKeep) > 0 {
		chain.AddNamed("always_keep", filter.KeepByRules(cfg.Rules.AlwaysKeep, originalInput))
	}

	return chain
}

// passthrough returns true if filtering should be skipped.
func passthrough() bool {
	return os.Getenv("GOTK_PASSTHROUGH") == "1"
}

// RunCommand executes a single command string through the shell, filters
// stdout, and passes stderr through unmodified. Returns the exit code.
func RunCommand(cfg *config.Config, command string, maxLines int) int {
	shell := findShell()

	// Use exec.RunWithTimeout with the configured timeout
	timeout := time.Duration(cfg.Security.CommandTimeout) * time.Second
	if timeout <= 0 {
		timeout = gotkexec.DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if passthrough() {
		cmd := exec.CommandContext(ctx, shell, "-c", command)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		return exitCode(err)
	}

	result, err := gotkexec.RunWithTimeout(ctx, shell, "-c", command)
	if err != nil && result == nil {
		return exitCode(err)
	}

	raw := result.Stdout
	if raw == "" {
		return exitCodeFromResult(result, err)
	}

	// Pass stderr through unmodified
	if result.Stderr != "" {
		fmt.Fprint(os.Stderr, result.Stderr)
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

	chain := BuildChainWithKeep(cfg, cmdType, maxLines, raw)
	cleaned := chain.Apply(raw)
	fmt.Fprint(os.Stdout, cleaned) //nolint:errcheck

	return exitCodeFromResult(result, err)
}

// RunShell starts an interactive-ish proxy shell that reads commands from
// stdin, executes them, and writes filtered output to stdout. This mode
// is designed for LLM tool integration where the caller sets
// SHELL=/path/to/gotk and sends commands one at a time.
func RunShell(cfg *config.Config, maxLines int) int {
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

		code := RunCommand(cfg, line, maxLines)
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

func exitCodeFromResult(result *gotkexec.Result, err error) int {
	if result != nil {
		return result.ExitCode
	}
	return exitCode(err)
}
