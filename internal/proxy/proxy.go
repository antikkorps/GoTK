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
	"github.com/antikkorps/GoTK/internal/shell"
)

// BuildChain creates a filter chain controlled by config, command type, and max lines.
func BuildChain(cfg *config.Config, cmdType detect.CmdType, maxLines int) *filter.Chain {
	return buildChain(cfg, cmdType, maxLines, filter.Summarize)
}

// BuildChainWithExit is like BuildChain but feeds the command's exit code and
// stderr into the summary step, so `result: PASS/FAIL` reflects authoritative
// signals (exit code, stderr-side runner totals) instead of only heuristic
// anchor matching on stdout.
func BuildChainWithExit(cfg *config.Config, cmdType detect.CmdType, maxLines int, exitCode int, stderr string) *filter.Chain {
	return buildChain(cfg, cmdType, maxLines, filter.SummarizeWithContext(exitCode, stderr))
}

func buildChain(cfg *config.Config, cmdType detect.CmdType, maxLines int, summarize filter.FilterFunc) *filter.Chain {
	chain := filter.NewChain()

	// Blacklist: remove matching lines early, before any other processing
	if len(cfg.Rules.AlwaysRemove) > 0 {
		chain.AddNamed("always_remove", filter.RemoveByRules(cfg.Rules.AlwaysRemove))
	}

	if cfg.Filters.StripANSI {
		chain.AddNamed("strip_ansi", filter.StripANSI)
	}
	// Timestamp prefixes are stripped before whitespace normalization and
	// dedup: runners pad after the clock (so the raw line still carries its
	// indentation at this point), and two otherwise-identical lines can only
	// collapse once the clock that distinguishes them is gone.
	if cfg.Filters.StripTimestamps {
		chain.AddNamed("strip_timestamps", filter.StripTimestamps)
	}
	if cfg.Filters.NormalizeWhitespace {
		chain.AddNamed("normalize_whitespace", filter.NormalizeWhitespace)
	}
	if cfg.Filters.Dedup {
		chain.AddNamed("dedup", filter.Dedup)
		// Collapse Node.js worker warnings that differ only by PID. Runs
		// after Dedup because the generic dedup cannot see through the PID;
		// this filter normalizes PIDs and collapses consecutive blocks. See
		// issue #37.
		chain.AddNamed("node_warnings", filter.CollapseNodeWarnings)
	}

	// Command-specific filters (CompressPaths is included via detect.FiltersFor
	// for certain types, so we only add it for generic if enabled)
	cmdFilters := detect.FiltersForWithOptions(cmdType, detect.Options{
		JestDropConsoleOnPass: cfg.Filters.JestDropConsoleOnPass,
	})
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
	chain.AddNamed("summarize", summarize)

	if cfg.Filters.Truncate {
		mode := filter.ParseAutoEscalate(cfg.General.AutoEscalate)
		chain.AddNamed("truncate", filter.TruncateWithEscalation(maxLines, mode, cfg.General.EscalateWindow))
	}

	return chain
}

// BuildStderrChain returns a conservative filter chain to apply to a
// command's stderr before emitting it. The stderr policy is deliberately
// narrow: only filters that strip pure noise or redact dangerous content
// run, never structural / summarizing / truncating filters. Stderr is the
// channel where diagnostics live, so we must not mangle its structure or
// drop any signal an LLM (or a human) needs to debug.
//
// Filters applied (in order):
//   - strip_ansi: remove ANSI escape codes (cosmetic; safe on any text).
//   - redact_secrets: same redaction policy as stdout, so a secret written
//     to stderr can't leak past gotk. Was the canonical missing piece
//     before this chain existed.
//   - node_warnings: PID-aware collapse of repeated worker warnings.
//     Node emits these via process.emitWarning to stderr, so multi-worker
//     test runs flood the channel. See issue #37.
//
// Streaming mode (cmd/gotk runStreaming) does not currently apply this
// chain because stderr is forwarded line-by-line as it arrives; batching
// it through the chain would defer output. Live stderr stays best-effort
// passthrough for now.
func BuildStderrChain(cfg *config.Config) *filter.Chain {
	chain := filter.NewChain()
	if cfg.Filters.StripANSI {
		chain.AddNamed("strip_ansi", filter.StripANSI)
	}
	if cfg.Security.RedactSecrets {
		chain.AddNamed("redact_secrets", filter.RedactSecrets)
	}
	chain.AddNamed("node_warnings", filter.CollapseNodeWarnings)
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
	sh, shFlag := shell.Default()

	// Use exec.RunWithTimeout with the configured timeout
	timeout := time.Duration(cfg.Security.CommandTimeout) * time.Second
	if timeout <= 0 {
		timeout = gotkexec.DefaultTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if passthrough() {
		cmd := exec.CommandContext(ctx, sh, shFlag, command)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		err := cmd.Run()
		return exitCode(err)
	}

	result, err := gotkexec.RunWithTimeout(ctx, sh, shFlag, command)
	if err != nil && result == nil {
		return exitCode(err)
	}

	raw := result.Stdout
	if raw == "" {
		return exitCodeFromResult(result, err)
	}

	// Apply the stderr policy chain (strip ANSI, redact secrets, collapse
	// node worker warnings). Without this, secrets in stderr would leak.
	if result.Stderr != "" {
		stderrChain := BuildStderrChain(cfg)
		fmt.Fprint(os.Stderr, stderrChain.Apply(result.Stderr))
	}

	// Detect command type from the first word; fall back to AutoDetect on
	// captured output so wrappers (jest, vitest, etc.) that aren't in the
	// binary registry still pick up the right filters.
	parts := strings.Fields(command)
	cmdType := detect.CmdGeneric
	if len(parts) > 0 {
		if mapped, ok := cfg.Commands[parts[0]]; ok {
			cmdType = detect.Identify(mapped)
		} else {
			cmdType, _ = detect.IdentifyOrDetect(parts[0], raw)
		}
	}

	exitCode := exitCodeFromResult(result, err)
	chain := BuildChainWithExit(cfg, cmdType, maxLines, exitCode, result.Stderr)
	if len(cfg.Rules.AlwaysKeep) > 0 {
		chain.AddNamed("always_keep", filter.KeepByRules(cfg.Rules.AlwaysKeep, raw))
	}
	cleaned := chain.Apply(raw)
	fmt.Fprint(os.Stdout, cleaned) //nolint:errcheck

	return exitCode
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
