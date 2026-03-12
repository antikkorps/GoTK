# GoTK - LLM Output Proxy

GoTK is a CLI tool that sits between shell commands and LLMs, stripping noise from command output to reduce token usage. It removes ANSI codes, collapses duplicate lines, compresses file paths, and applies command-specific optimizations — all without losing semantically important information.

## Benchmarks

| Command | Typical Reduction |
|---------|-------------------|
| `grep -rn` | **-95%** |
| `git log` | **-90%** |
| `find` | **-70%** |
| `ls -la` | **-51%** |

Results vary by output size and content. Use `--stats` to see exact savings per invocation.

## Installation

```bash
go install github.com/fMusic/GoTK/cmd/gotk@latest
```

Or build from source:

```bash
git clone https://github.com/fMusic/GoTK.git
cd GoTK
go build -o gotk ./cmd/gotk/
```

## Usage

GoTK operates in three modes:

### Direct mode

Prefix any command with `gotk`:

```bash
gotk grep -rn "func main" .
gotk git log --oneline -20
gotk find . -name "*.go"
gotk go test ./...
```

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
```

Key design decisions:
- Filters are composable `func(string) string` functions chained via `filter.Chain`
- Generic filters always run; command-specific filters are added based on detection
- Stderr passes through unmodified — only stdout is cleaned
- Conservative by default: never discard semantically important content

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
