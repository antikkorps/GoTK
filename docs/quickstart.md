# GoTK Quick Start

Get started in 30 seconds. GoTK cleans command output before sending to LLMs — fewer tokens, same quality.

## Install

```bash
go build -o /usr/local/bin/gotk ./cmd/gotk/
```

Or keep it local to your project:

```bash
go build -o gotk ./cmd/gotk/
export PATH="$PWD:$PATH"
```

## Verify

```bash
gotk --version
echo "test output with noise" | gotk --stats
```

---

## Claude Code

Add to your project's `CLAUDE.md`:

```markdown
## Token Optimization

Always use gotk to execute shell commands that produce verbose output:
  gotk grep -rn "pattern" .
  gotk go test ./...
  gotk git log --oneline -20
  gotk find . -name "*.go"

Use gotk ctx for codebase search (optimized for LLM consumption):
  gotk ctx pattern -t go             # scan mode
  gotk ctx pattern --def -t go       # definition search

Never run grep, find, git, go test, make, or other verbose commands without
the gotk prefix. This reduces token consumption by ~80%.
```

Claude reads this file on every conversation and follows the instructions.

A ready-to-copy template is available at `examples/CLAUDE.md.gotk-template`.

> **Note:** For 100% automatic filtering, use MCP mode instead:
> `claude mcp add --transport stdio gotk -- gotk --mcp`

## Aider

```bash
SHELL=/usr/local/bin/gotk GOTK_SHELL=/bin/bash aider
```

## Cursor

In Cursor's `settings.json`:

```json
{
  "terminal.integrated.profiles.osx": {
    "gotk": {
      "path": "/usr/local/bin/gotk",
      "args": ["--shell"]
    }
  },
  "terminal.integrated.defaultProfile.osx": "gotk"
}
```

Or:

```bash
SHELL=/usr/local/bin/gotk GOTK_SHELL=/bin/bash cursor .
```

## Continue.dev

In `~/.continue/config.json`:

```json
{
  "contextProviders": [
    {
      "name": "terminal",
      "params": { "shell": "/usr/local/bin/gotk -c" }
    }
  ]
}
```

## Any Other AI Tool

**SHELL replacement** (if the tool respects `$SHELL`):

```bash
SHELL=/usr/local/bin/gotk GOTK_SHELL=/bin/bash your-ai-tool
```

**Pipe filter**:

```bash
your-command | gotk
```

**Direct execution**:

```bash
gotk your-command args...
```

---

## Context Search

Search your codebase with LLM-optimized output. Built-in exclusions skip node_modules, .git, vendor, lock files, and binaries.

```bash
gotk ctx BuildChain                    # Scan: file list + match counts
gotk ctx BuildChain -d 5              # Detail: context windows around matches
gotk ctx Handler --def -t go          # Def: function/class declarations
gotk ctx main --tree                  # Tree: structural skeleton
gotk ctx "func.*Error" --summary      # Summary: directory breakdown
gotk ctx Config -t py -m 10           # Filter by type, limit results
```

Token savings vs raw grep: **-48% to -98%** depending on mode.

---

## Tuning

```bash
# Less filtering (keep more output)
gotk --conservative grep -rn "pattern" .

# More filtering (save more tokens)
gotk --aggressive find / -name "*.log"

# See what was removed
gotk --stats go test ./...

# No truncation
gotk --no-truncate make build

# Stream real-time
gotk --stream make build

# Disable temporarily
GOTK_PASSTHROUGH=1 your-command
```

## Measure Token Savings

```bash
gotk --measure grep -rn "func" .
gotk measure last
gotk measure report --period 7d
```

---

## Next Steps

- `gotk --help` or `gotk help <command>` for detailed usage
- `man gotk` for the full manual
- [CLI vs MCP comparison](cli-vs-mcp.md)
- [Advanced integration patterns](integrations.md)
