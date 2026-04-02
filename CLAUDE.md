# GoTK - LLM Output Proxy

CLI tool (Go) that acts as a proxy to clean command output before sending to LLMs.
Goal: ~80% token reduction by stripping noise.

## Language

**All code, comments, documentation, backlog, and commit messages MUST be written in English.**

## Build & Run

```bash
go build -ldflags "-s -w -X main.Version=$(git describe --tags --always)" -o gotk ./cmd/gotk/
./gotk --stats grep -rn "pattern" .
echo "test" | ./gotk --stats
./gotk ctx BuildChain -t go          # context search
./gotk config show                   # show effective config
./gotk --debug echo hello            # diagnostic output
go test ./...
```

## Architecture

- `cmd/gotk/main.go` — CLI entrypoint, flag parsing, pipe detection
- `cmd/gotk/subcommands.go` — All subcommand handlers (bench, ctx, daemon, install, learn, measure, watch, hook, config)
- `cmd/gotk/usage.go` — Help text and subcommand help
- `internal/exec/` — Command execution and output capture
- `internal/filter/` — Filter chain (named filters with introspection) + individual filters (ANSI, whitespace, dedup, paths, trim, truncate, secrets, stack traces, summary)
- `internal/detect/` — Command registry (18 command types with binary aliases + filters), auto-detection from output
- `internal/ctx/` — Context search engine (walk, search, 5 output formatters)
- `internal/config/` — TOML config loader (3-level merge: global, project, local), `Show()` for introspection
- `internal/proxy/` — Proxy shell mode (`--shell`, `-c`), filter chain builder with named filters
- `internal/hook/` — Claude Code PreToolUse hook protocol handler
- `internal/daemon/` — Filtered shell session (zsh/bash interception)
- `internal/install/` — Auto-configure Claude Code hooks in settings.json
- `internal/cmdclass/` — Shared command classification (TrivialCommands, InteractiveCommands)
- `internal/mcp/` — MCP server (JSON-RPC over stdio, 4 tools)
- `internal/measure/` — Token consumption metrics and logging
- `internal/learn/` — Project-specific pattern learning (observe, analyze, suggest)
- `internal/bench/` — Benchmark suite with quality scoring
- `internal/classify/` — Semantic line classifier (error/warning/info/debug/noise)
- `internal/cache/` — Content-hash based output cache
- `internal/watch/` — File watcher for re-run on changes
- `internal/errors/` — Custom error types

## Design Principles

- Filters are composable functions `func(string) string` chained via `filter.Chain`
- Chain supports named filters (`AddNamed`) with `Names()`/`Len()` for introspection
- Command detection uses a registry pattern (single map: CmdType -> {name, binaries, filters})
- Generic filters always apply (ANSI strip, whitespace normalize, dedup, trim)
- Command-specific filters add on top (grep path compression, find prefix factoring, etc.)
- Stderr passes through unmodified
- `--quiet` suppresses info/warning messages, `--debug` shows config/detection/chain details

## Quality-First Filtering

**Output quality is the #1 priority — token reduction is secondary.**

GoTK must NEVER degrade the quality of LLM responses by removing semantically important information. Every filter must follow these rules:

1. **Never remove error messages, warnings, or diagnostic info** — these are the most valuable lines for an LLM
2. **Never remove context needed to understand a problem** — file paths tied to errors, line numbers in stack traces, variable values in assertions
3. **Preserve structural meaning** — indentation in code, hierarchy in tree output, grouping in test results
4. **Only remove pure noise** — ANSI codes, decorative separators, duplicate lines, redundant path prefixes, excessive blank lines
5. **When in doubt, keep it** — a false negative (keeping noise) is far less costly than a false positive (removing signal)
6. **Truncation must keep head AND tail** — errors and summaries often appear at the end of output
7. **Filters must be testable** — every filter needs golden-file tests with before/after validation to prevent quality regressions
