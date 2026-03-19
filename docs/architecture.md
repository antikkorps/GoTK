# Architecture

This document covers GoTK's internal design. For usage and a high-level overview, see the [README](../README.md).

## Package Layout

```
cmd/gotk/main.go       Entry point. Parses flags, detects pipe vs direct mode,
                        orchestrates exec -> detect -> filter -> output.

internal/exec/          Runner. Executes a subprocess, captures stdout/stderr
                        separately, returns exit code. Supports timeout and
                        streaming (RunStream).

internal/filter/        The filter chain, generic filter functions, stream
                        filter, stack trace compression, secret redaction,
                        and output summarization.

internal/detect/        Command type identification (by name and by output
                        pattern) and command-specific filter functions for
                        9 command types (grep, find, git, go, ls, docker,
                        npm/yarn, cargo, make).

internal/classify/      Semantic line classifier. Categorizes each line as
                        Noise, Debug, Info, Warning, Error, or Critical.
                        Used by quality scoring, learn, and summarization.

internal/ctx/           Context search engine. Walks files with built-in
                        exclusions, regex search, and 5 output formatters
                        (scan, detail, def, tree, summary). Output goes
                        through proxy.BuildChain for standard GoTK filtering.

internal/proxy/         Proxy shell mode (--shell, -c). Builds the filter
                        chain (BuildChain) combining config, command type,
                        and rules.

internal/config/        TOML config loader with 3-level precedence (global,
                        project, local). No external dependencies.

internal/mcp/           MCP server (Model Context Protocol). Exposes
                        gotk_exec, gotk_filter, gotk_read, gotk_grep,
                        gotk_ctx as JSON-RPC tools over stdio.

internal/cache/         LRU content-hash cache for filter results. Skips
                        re-filtering identical output across invocations.

internal/learn/         Project-specific pattern learning. Observes command
                        output, classifies lines, normalizes variable parts,
                        and suggests always_remove patterns for .gotk.toml.

internal/watch/         Watch mode. Polls for file changes and re-runs the
                        command with filtered output. Debounce and extension
                        filtering support.

internal/measure/       Token consumption measurement. JSONL logger, report
                        generation, quality scoring.

internal/bench/         Benchmark suite with realistic fixtures. Per-filter
                        contribution analysis, A/B testing, quality scoring.

internal/errors/        Custom error types (typed errors instead of raw strings).
```

## Filter Chain Pattern

The core abstraction is `filter.FilterFunc`:

```go
type FilterFunc func(string) string
```

Filters are pure functions: string in, string out. They are composed into a `filter.Chain`, which applies them sequentially — each filter receives the output of the previous one.

The chain is built in `proxy.BuildChain()` with a fixed structure:

1. **Blacklist rules** (`RemoveByRules` — from `[rules] always_remove` config)
2. **Generic filters** (always applied, in order):
   - `StripANSI`
   - `NormalizeWhitespace`
   - `Dedup`
3. **Command-specific filters** (zero or more, determined by `detect.FiltersFor`)
4. **Final cleanup**:
   - `TrimEmpty`
   - `CompressStackTraces` (Go, Python, Node.js)
   - `RedactSecrets` (API keys, tokens, passwords, JWTs)
   - `Summarize` (error/warning counts for large output)
   - `Truncate` (always last)

Order matters. ANSI codes must be stripped before whitespace normalization. Dedup runs before command-specific filters so they operate on already-deduplicated input. Truncation is always last so the line budget applies to the final cleaned output.

## Command Detection

GoTK identifies the command type to select command-specific filters. There are two detection paths:

### Explicit detection (direct/exec mode)

`detect.Identify(command)` examines the binary name (after `filepath.Base` and stripping `.exe`). It maps names to `CmdType` constants:

| CmdType | Matched binaries |
|---------|------------------|
| `CmdGrep` | grep, rg, ag, ack |
| `CmdFind` | find, fd |
| `CmdGit` | git, gh |
| `CmdGoTool` | go |
| `CmdLs` | ls, exa, eza, lsd |
| `CmdDocker` | docker, docker-compose, podman |
| `CmdNpm` | npm, yarn, pnpm, npx, bun |
| `CmdCargo` | cargo, rustc |
| `CmdMake` | make, cmake, ninja |
| `CmdGeneric` | everything else |

### Auto-detection (pipe mode)

`detect.AutoDetect(output)` samples the first 20 non-empty lines and matches them against regex patterns:

| Pattern | Detected type |
|---------|--------------|
| `file:linenum:content` | grep |
| Lines starting with `./` or `/` paths | find |
| `commit [hex hash]` | git log |
| `diff --git a/...` | git diff |
| `ok`/`FAIL`/`--- PASS` prefixes | go test |
| 10-char permission strings (`drwxr-xr-x`) | ls |

A command type is selected only if it matches more than 40% of sampled lines (minimum 2 matches). This prevents false positives on mixed output.

## Command-Specific Filters

`detect.FiltersFor(cmdType)` returns the specialized filters for each command type. These live in `internal/detect/detect.go` alongside the detection logic, since they are tightly coupled to it.

| CmdType | Filters applied |
|---------|----------------|
| `CmdGrep` | `CompressPaths`, `compressGrepOutput` |
| `CmdFind` | `CompressPaths`, `compressFindOutput` |
| `CmdGit` | `compressGitOutput` |
| `CmdGoTool` | `CompressPaths`, `compressGoOutput` |
| `CmdLs` | `compressLsOutput` |
| `CmdDocker` | `compressDockerOutput` |
| `CmdNpm` | `compressNpmOutput` |
| `CmdCargo` | `compressCargoOutput` |
| `CmdMake` | `compressMakeOutput` |
| `CmdGeneric` | `CompressPaths` |

For details on what each filter does, see [docs/filters.md](filters.md).

## Adding New Filters

To add a new generic filter:

1. Create a file in `internal/filter/` (e.g., `myfilter.go`)
2. Implement the function with signature `func(string) string`
3. Add it to the chain in `proxy.BuildChain()` at the appropriate position

Guidelines:
- Filters must be idempotent — applying twice should produce the same result as once
- Never discard content that could be semantically meaningful
- Prefer conservative transformations; it is better to leave noise than to lose signal

## Adding New Command Types

1. Add a new `CmdType` constant in `internal/detect/detect.go`
2. Add the binary name mapping in `Identify()`
3. Add a pattern in `internal/detect/autodetect.go` for pipe-mode detection
4. Write the command-specific filter function(s) in `detect.go`
5. Register the filters in `FiltersFor()`

The filter function receives fully deduplicated, whitespace-normalized, ANSI-stripped text. It should focus on structural compression (grouping, prefix factoring, metadata stripping) rather than basic cleanup.
