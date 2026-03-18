# GoTK - LLM Output Proxy

GoTK is a CLI tool that sits between shell commands and LLMs, stripping noise from command output to reduce token usage. It removes ANSI codes, collapses duplicate lines, compresses file paths, and applies command-specific optimizations — all without losing semantically important information.

## Benchmarks

| Command | Typical Reduction |
|---------|-------------------|
| `grep -rn` | **-95%** |
| `git log` | **-90%** |
| `find` | **-70%** |
| `ls -la` | **-51%** |
| `pnpm test` (real project, 5601 lines) | **-98%** |

Results vary by output size and content. Use `--stats` to see exact savings per invocation.

## Installation

```bash
go install github.com/antikkorps/GoTK/cmd/gotk@latest
```

Or build from source:

```bash
git clone https://github.com/antikkorps/GoTK.git
cd GoTK
go build -o gotk ./cmd/gotk/
```

To make `gotk` available system-wide:

```bash
sudo ln -s $(pwd)/gotk /usr/local/bin/gotk
```

For LLM tool integrations (Claude Code, Aider, Cursor, Continue.dev), see [docs/integrations.md](docs/integrations.md).

## Usage

GoTK operates in several modes:

### Direct mode

Prefix any command with `gotk`:

```bash
gotk grep -rn "func main" .
gotk git log --oneline -20
gotk find . -name "*.go"
gotk go test ./...
```

### Context search

Search your codebase with output optimized for LLMs. Five output modes, built-in exclusions (node_modules, .git, vendor, lock files), and binary file detection:

```bash
gotk ctx BuildChain                    # Scan: file list + match counts
gotk ctx BuildChain -d 5              # Detail: 5-line context windows
gotk ctx BuildChain --def             # Def: function/class/type declarations
gotk ctx BuildChain --tree            # Tree: structural skeleton
gotk ctx BuildChain --summary         # Summary: directory breakdown table
gotk ctx BuildChain -t go -m 10      # Filter by type, max 10 files
```

Token savings vs raw `grep -rn`: **-48% to -98%** depending on mode and result volume.

### Explicit exec mode

Use `exec --` to separate gotk flags from command flags:

```bash
gotk exec -- grep -rn "TODO" .
gotk --stats exec -- git diff HEAD~3
```

### Pipe mode

Pipe any command's output through gotk. Output format is auto-detected:

```bash
grep -rn "TODO" . | gotk
git log -50 | gotk --stats
cat build.log | gotk --max-lines 100
```

## Flags

| Flag | Short | Description |
|------|-------|-------------|
| `--stats` | `-s` | Print reduction statistics to stderr |
| `--max-lines N` | `-m N` | Maximum output lines (default: 50, keeps head + tail) |
| `--no-truncate` | | Disable line limit entirely |
| `--conservative` | | Minimal reduction, zero info loss |
| `--balanced` | | Default mode — good reduction, preserves important lines |
| `--aggressive` | | Maximum reduction, acceptable info loss |
| `--stream` | | Stream output line-by-line with real-time filtering |
| `--help` | `-h` | Show help |
| `--version` | `-v` | Show version |

## How It Works

```
command output
      |
      v
  [Capture] ---- stdout captured, stderr passed through
      |
      v
  [Detect] ----- identify command type (by name or output patterns)
      |
      v
  [Filter Chain]
      |-- StripANSI ............. remove escape codes
      |-- NormalizeWhitespace ... collapse blanks, trim trailing spaces
      |-- Dedup ................. collapse consecutive duplicate lines
      |-- <command-specific> .... grep grouping, path compression, etc.
      |-- TrimEmpty ............. remove decorative separator lines
      |-- Truncate .............. cap at --max-lines (head + tail)
      |
      v
  clean output (stdout)
  stats (stderr, if --stats)
```

Command detection works two ways:
- **Direct/exec mode**: the binary name is matched against known commands (grep, rg, find, fd, git, go, ls, exa, etc.)
- **Pipe mode**: the first 20 non-empty lines are pattern-matched against known output formats (grep-style `file:line:content`, git commit hashes, permission strings, etc.)

## Architecture

For detailed internals, see [docs/architecture.md](docs/architecture.md).

```
cmd/gotk/          CLI entrypoint, flag parsing, pipe detection
internal/exec/     Command execution and output capture
internal/filter/   Filter chain + individual filter functions
internal/detect/   Command identification + command-specific filter selection
internal/ctx/      Context search engine (walk, search, 5 formatters)
internal/mcp/      MCP server (gotk_exec, gotk_filter, gotk_read, gotk_grep, gotk_ctx)
internal/cache/    LRU content-hash cache for filter results
```

Key design decisions:
- Filters are composable `func(string) string` functions chained via `filter.Chain`
- Generic filters always run; command-specific filters are added based on detection
- Stderr passes through unmodified — only stdout is cleaned
- Conservative by default: never discard semantically important content

## Configuration

GoTK reads configuration from three levels (in order of precedence):

1. `~/.config/gotk/config.toml` — Global defaults
2. `.gotk.toml` — Project config (found by walking up to repo root)
3. `./gotk.toml` — Local override

```toml
[general]
mode = "balanced"    # conservative | balanced | aggressive
max_lines = 50

[filters]
strip_ansi = true
dedup = true
truncate = true

[security]
redact_secrets = true
command_timeout = 30
rate_limit = 0          # MCP rate limit (requests/min, 0 = disabled)
sandbox_mode = false    # MCP sandbox: restrict to read-only commands

[rules]
always_keep = ["^ERROR:", "^FATAL:"]     # regex: these lines are never removed
always_remove = ["^DEBUG:", "^TRACE:"]   # regex: these lines are always removed

[truncation]
grep = 30       # per-command max_lines overrides
test = 200
git = 100
```

## Filter Catalog

See [docs/filters.md](docs/filters.md) for a complete catalog with before/after examples.

## Contributing

1. Fork and clone the repo
2. Create a feature branch
3. Add tests for new functionality
4. Run `go test ./...`
5. Submit a pull request

To add a new filter, see [Adding new filters](docs/architecture.md#adding-new-filters).
To add support for a new command, see [Adding new command types](docs/architecture.md#adding-new-command-types).

## License

MIT
