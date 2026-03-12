# GoTK - LLM Output Proxy

CLI tool (Go) that acts as a proxy to clean command output before sending to LLMs.
Goal: ~80% token reduction by stripping noise.

## Language

**All code, comments, documentation, backlog, and commit messages MUST be written in English.**

## Build & Run

```bash
go build -o gotk ./cmd/gotk/
./gotk --stats grep -rn "pattern" .
echo "test" | ./gotk --stats
go test ./...
```

## Architecture

- `cmd/gotk/` — CLI entrypoint, flag parsing, pipe detection
- `internal/exec/` — Command execution and output capture
- `internal/filter/` — Filter chain + individual filters (ANSI, whitespace, dedup, paths, trim, truncate)
- `internal/detect/` — Command detection (explicit + auto-detect) + command-specific filter selection
- `internal/config/` — TOML config loader (no external deps)
- `internal/proxy/` — Proxy shell mode (`--shell`, `-c`)

## Design Principles

- Filters are composable functions `func(string) string` chained via `filter.Chain`
- Generic filters always apply (ANSI strip, whitespace normalize, dedup, trim)
- Command-specific filters add on top (grep path compression, find prefix factoring, etc.)
- Stderr passes through unmodified

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
