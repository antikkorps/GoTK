package main

import (
	"fmt"
	"os"
	"strings"
)

func printUsage() {
	usage := `GoTK - LLM Output Proxy

Clean command output before sending to LLMs. Reduces token usage by ~80%
while preserving errors, warnings, and semantically important content.

Best used as a CLI tool integrated with your AI coding assistant.
See "gotk help mcp" for MCP server mode (higher token overhead).

Usage:
  gotk [flags] <command> [args...]    Run a command with filtered output
  <command> | gotk [flags]            Filter piped input
  gotk -c "command"                   Shell-compatible execution
  gotk --shell                        Proxy shell mode (for SHELL= integration)

Subcommands:
  ctx       Context search with 5 output modes (gotk ctx pattern)
  learn     Project-specific pattern learning (run, suggest, status, clear)
  config    Show loaded config files and effective settings
  daemon    Start a filtered shell session (gotk daemon)
  install   Configure GoTK integration (gotk install claude)
  exec      Execute a command explicitly (gotk exec -- cmd args...)
  watch     Re-run command on file changes (gotk watch -- make test)
  bench     Run benchmark suite
  measure   Token consumption metrics (report, last, status, clear)
  help      Show help for a subcommand (gotk help watch)

Flags:
  -s, --stats          Show reduction statistics on stderr
  -m, --max-lines N    Max output lines (default: 50, keeps head+tail)
  --no-truncate        Disable line truncation
  --conservative       Minimal reduction, zero info loss
  --balanced           Default mode, good reduction
  --aggressive         Maximum reduction
  --mode MODE          Set filter mode explicitly
  --stream             Stream output line-by-line (real-time)
  --measure            Enable token consumption logging
  --learn              Passively observe output for pattern learning
  --profile PROFILE    LLM profile: claude, gpt, gemini
  --auto-escalate MODE Preserve failure context when truncating: off|hint|window|conservative (default: window)
  -q, --quiet          Suppress informational messages on stderr
  --debug              Show diagnostic output (config, detection, timing)
  --shell              Start proxy shell mode
  -c "command"         Execute single command
  --mcp                Start MCP server (JSON-RPC over stdio)
  -h, --help           Show this help
  -v, --version        Show version

Examples:
  gotk ctx BuildChain -t go                 Context search
  gotk grep -rn "func main" .              Direct mode
  gotk --stats go test ./...               With stats
  find . -name "*.go" | gotk               Pipe mode
  gotk --aggressive find / -name "*.log"   Maximum reduction
  gotk --stream make build                 Real-time filtering

AI tool integration (CLI mode, recommended for token efficiency):
  Claude Code  gotk install claude                          (auto hook)
  Aider        SHELL=gotk GOTK_SHELL=/bin/bash aider       (100% auto)
  Cursor       SHELL=gotk GOTK_SHELL=/bin/bash cursor .    (100% auto)

Config: ~/.config/gotk/config.toml | .gotk.toml | ./gotk.toml
Environment: GOTK_PASSTHROUGH=1 (bypass) | GOTK_SHELL (real shell)

Run "gotk help <subcommand>" for details. See also: man gotk`

	fmt.Println(strings.TrimSpace(usage))
}

func printSubcommandHelp(sub string) {
	helps := map[string]string{
		"ctx": `gotk ctx — Context Search

Usage:
  gotk ctx [flags] <pattern> [directory]

Search codebase and format results for LLM consumption. Five output modes
with built-in exclusions (node_modules, .git, vendor, etc.) and binary
file detection. Output is filtered through the GoTK pipeline.

Modes:
  (default)     Scan: file paths with match counts, indented matches
  -d [N]        Detail: N-line context windows with overlap merging (default: 3)
  --def         Def: language-aware declarations (func, class, struct, etc.)
  --tree        Tree: structural skeleton (imports, types, functions)
  --summary     Summary: directory breakdown table

Flags:
  -t, --type EXT     File extension filter (e.g., -t go, -t py)
  -g, --glob GLOB    Glob filter on filename (e.g., -g "*.test.js")
  -m, --max N        Max file results (0 = unlimited)
  -p, --path DIR     Root directory (alternative to positional arg)

Examples:
  gotk ctx BuildChain                       Scan mode (default)
  gotk ctx BuildChain -d 5                  Detail mode with 5 lines context
  gotk ctx BuildChain --def                 Definition mode
  gotk ctx BuildChain --tree                Tree/skeleton mode
  gotk ctx BuildChain --summary             Summary mode
  gotk --stats ctx BuildChain -t go -m 5     Filtered with stats
  gotk ctx "func.*Error" -t go              Regex search in Go files`,

		"config": `gotk config — Show configuration

Usage:
  gotk config [show]

Display the effective configuration after merging all config files.
Shows which files were loaded and in what order (global, project, local).

Config file locations (in precedence order):
  1. ~/.config/gotk/config.toml    Global config
  2. .gotk.toml                    Project config (found by walking up from cwd)
  3. ./gotk.toml                   Local config (current directory)

Examples:
  gotk config                      Show effective config
  gotk config show                 Same as above`,

		"exec": `gotk exec — Explicit command execution

Usage:
  gotk [flags] exec -- <command> [args...]

Execute a command with GoTK flags clearly separated from command arguments
using the -- delimiter.

Examples:
  gotk exec -- grep -rn "pattern" .
  gotk --aggressive exec -- find / -name "*.log"
  gotk --stats exec -- go test -v ./...`,

		"watch": `gotk watch — Watch files and re-run command on changes

Usage:
  gotk watch [watch-flags] -- <command> [args...]

Watch for file changes and automatically re-run the command with filtered
output. Useful for test-driven development with AI assistants.

Watch flags (before --):
  -i, --interval D    Polling interval (default: 2s, e.g., 5s, 500ms, 1m)
  -e, --ext EXT       File extension to watch (repeatable)
  -p, --path PATH     Path to watch (repeatable, default: ".")

Examples:
  gotk watch -- go test ./...
  gotk watch --ext .go -- go test ./...
  gotk watch --interval 5s -- make build
  gotk watch -e .py -p src -- python -m pytest
  gotk watch -e .go -e .mod -p ./internal -- go test ./internal/...`,

		"bench": `gotk bench — Run benchmark suite

Usage:
  gotk bench [flags]

Run benchmarks on realistic command output fixtures to measure reduction
rates, quality scores, and latency.

Flags:
  --json          Output results as JSON
  --per-filter    Show per-filter contribution breakdown
  --quality       Measure quality score (% of important lines preserved)
  --abtest        Compare conservative/balanced/aggressive modes side by side

Examples:
  gotk bench
  gotk bench --per-filter
  gotk bench --quality
  gotk bench --abtest
  gotk bench --json`,

		"measure": `gotk measure — Token consumption metrics

Usage:
  gotk measure <subcommand>

Track and analyze token savings from GoTK filtering.

Subcommands:
  report [--json] [--period D]    Show savings report (e.g., --period 7d)
  last [N]                        Show last N invocations (default: 10)
  status                          Show measurement status and log info
  clear                           Clear the measurement log

To enable measurement, use --measure flag or set measure.enabled = true
in your config file. Measurement is auto-enabled in MCP mode.

Examples:
  gotk --measure grep -rn "func" .
  gotk measure last
  gotk measure last 20
  gotk measure report
  gotk measure report --period 7d --json
  gotk measure status`,

		"learn": `gotk learn — Project-Specific Pattern Learning

Usage:
  gotk learn run <command> [args...]   Observe a command's output
  gotk learn suggest                   Analyze and propose always_remove patterns
  gotk learn status                    Show observation statistics
  gotk learn clear                     Delete all observations

Passive mode (observe while working normally):
  gotk --learn <command> [args...]     Same as normal execution + silent observation

How it works:
  1. Run your usual commands with "gotk learn run" a few times
  2. GoTK classifies each line (noise/debug/info/warning/error)
  3. Normalizes variable parts (hashes, timestamps, numbers, versions)
  4. Groups by normalized form and tracks frequency across sessions
  5. "gotk learn suggest" proposes regex patterns for .gotk.toml [rules]

Safety:
  - Patterns that match any warning/error line are never suggested
  - Minimum 80% noise confidence required (configurable)
  - Human review required: suggestions are not auto-applied

Config (.gotk.toml):
  [learn]
  min_sessions = 3       # Sessions needed before suggesting
  min_frequency = 0.05   # Minimum line frequency (5%)
  min_noise = 0.80       # Minimum noise confidence (80%)

Examples:
  gotk learn run go test ./...         Observe test output
  gotk learn run make build            Observe build output
  gotk learn suggest                   See pattern suggestions
  gotk --learn go test ./...           Passive observation`,

		"daemon": `gotk daemon — Filtered shell session

Usage:
  gotk daemon [start]     Start a daemon session (spawns filtered shell)
  gotk daemon status      Check if inside a daemon session
  gotk daemon init        Print shell init code (for eval "$(gotk daemon init)")
  gotk daemon stop        Instructions to exit the session

How it works:
  GoTK spawns your shell (bash or zsh) with hooks that intercept every
  command. Non-trivial command output is automatically piped through gotk,
  reducing noise while preserving errors, warnings, and important content.

  Interactive programs (vim, less, top, ssh, etc.) and trivial commands
  (cd, pwd, echo, etc.) are not intercepted.

  A [gotk] prefix is added to your prompt so you know filtering is active.
  Type "gotk exit" to leave and see a session summary with tokens saved.

  Measurement is automatically enabled during daemon sessions.

Manual init (advanced):
  eval "$(gotk daemon init)"          # Inject into current shell
  eval "$(gotk daemon init --zsh)"    # Force zsh init
  eval "$(gotk daemon init --bash)"   # Force bash init

Examples:
  gotk daemon                         # Start filtered shell
  gotk daemon status                  # Check if active
  # Inside session: all commands auto-filtered
  grep -rn "pattern" .                # Output is filtered
  vim file.go                         # Opens normally (interactive)
  gotk exit                           # Leave session, see summary`,

		"install": `gotk install — Configure GoTK integrations

Usage:
  gotk install claude [flags]

Targets:
  claude    Install GoTK as a Claude Code PreToolUse hook

Flags:
  --global      Install in ~/.claude/settings.json (all projects)
  --project     Install in .claude/settings.json (default, current project)
  --uninstall   Remove GoTK hook configuration
  --status      Check if GoTK hook is installed

How it works:
  GoTK registers as a PreToolUse hook for the Bash tool. When Claude Code
  runs a shell command, the hook wraps it with "| gotk" so the output is
  filtered before Claude sees it. This reduces token usage by ~80%.

  Trivial commands (cd, pwd, echo, etc.) are not wrapped.
  Commands already piped through gotk are not double-wrapped.

Examples:
  gotk install claude                  Install for current project
  gotk install claude --global         Install for all projects
  gotk install claude --status         Check installation status
  gotk install claude --uninstall      Remove hook`,

		"hook": `gotk hook — Claude Code hook handler (internal)

Usage:
  gotk hook

This subcommand is called automatically by Claude Code when configured as
a PreToolUse hook. It reads a JSON payload from stdin, wraps Bash commands
with "| gotk" for output filtering, and writes the response to stdout.

You normally don't call this directly. Use "gotk install claude" to set up
the hook configuration automatically.

JSON input format (from Claude Code):
  {
    "hook_event_name": "PreToolUse",
    "tool_name": "Bash",
    "tool_input": {"command": "grep -rn pattern ."}
  }

JSON output format (to Claude Code):
  {
    "updatedInput": {"command": "set -o pipefail; (grep -rn pattern .) | /path/to/gotk"}
  }`,

		"mcp": `gotk --mcp — MCP Server Mode (Model Context Protocol)

Usage:
  gotk --mcp

Start a JSON-RPC server over stdio that exposes GoTK tools to MCP-compatible
AI coding assistants (Claude Code, etc.).

Exposed tools:
  gotk_exec     Execute any command and return cleaned output
  gotk_filter   Filter pre-existing text through the cleaning pipeline
  gotk_read     Read a file with smart truncation and noise removal
  gotk_grep     Search file contents with grouped, compressed results

Setup:
  claude mcp add --transport stdio gotk -- gotk --mcp

Note: CLI mode (PostToolUse hook) is more token-efficient than MCP mode.
MCP adds JSON-RPC overhead and tool schema tokens on every LLM turn.
See docs/cli-vs-mcp.md for a detailed comparison.

Use MCP mode when:
  - Your AI tool doesn't support hooks or SHELL replacement
  - You want the AI to explicitly choose when to filter
  - You need gotk_read or gotk_grep specialized tools`,
	}

	if h, ok := helps[sub]; ok {
		fmt.Println(strings.TrimSpace(h))
	} else {
		fmt.Fprintf(os.Stderr, "gotk help: unknown subcommand %q\n\n", sub)
		printUsage()
	}
}
