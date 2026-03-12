# Architecture

This document covers GoTK's internal design. For usage and a high-level overview, see the [README](../README.md).

## Package Layout

```
cmd/gotk/main.go       Entry point. Parses flags, detects pipe vs direct mode,
                        orchestrates exec -> detect -> filter -> output.

internal/exec/          Runner. Executes a subprocess, captures stdout/stderr
                        separately, returns exit code.

internal/filter/        The filter chain and all generic filter functions.

internal/detect/        Command type identification (by name and by output
                        pattern) and command-specific filter functions.
```

## Filter Chain Pattern

The core abstraction is `filter.FilterFunc`:

```go
type FilterFunc func(string) string
```

Filters are pure functions: string in, string out. They are composed into a `filter.Chain`, which applies them sequentially — each filter receives the output of the previous one.

The chain is built in `main.go` (`buildFilterChainForType`) with a fixed structure:

1. **Generic filters** (always applied, in order):
   - `StripANSI`
   - `NormalizeWhitespace`
   - `Dedup`
2. **Command-specific filters** (zero or more, determined by `detect.FiltersFor`)
3. **Final cleanup**:
   - `TrimEmpty`
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
| `CmdGeneric` | `CompressPaths` |

For details on what each filter does, see [docs/filters.md](filters.md).

## Adding New Filters

To add a new generic filter:

1. Create a file in `internal/filter/` (e.g., `myfilter.go`)
2. Implement the function with signature `func(string) string`
3. Add it to the chain in `main.go`'s `buildFilterChainForType` at the appropriate position

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
