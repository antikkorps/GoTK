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

One command to set up 100% automatic filtering:

```bash
gotk install claude
```

This registers GoTK as a Claude Code hook. Every Bash command Claude runs
is automatically filtered — no CLAUDE.md instructions needed, no MCP overhead.

```bash
gotk install claude --global     # All projects (~/. claude/settings.json)
gotk install claude --status     # Check if installed
gotk install claude --uninstall  # Remove
```

> **Alternatives:** CLAUDE.md instructions (~95% reliable) or MCP mode (higher
> token overhead). See [integrations.md](integrations.md) for all options.

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

## Daemon Mode (Filtered Shell)

Start a shell session where all output is automatically filtered:

```bash
gotk daemon
```

Your prompt changes to `[gotk] $` so you know filtering is active. Interactive
programs (vim, ssh, less) and trivial commands (cd, pwd) pass through normally.

```bash
gotk daemon status          # Check if inside a session
eval "$(gotk daemon init)"  # Inject into current shell (advanced)
```

Type `gotk exit` to leave — a session summary shows total commands filtered and tokens saved.

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

## Watch Mode

Re-run commands on file changes with filtered output — great for TDD loops:

```bash
gotk watch -- go test ./...                     # Watch all files
gotk watch --ext .go --ext .mod -- go test ./...  # Only .go and .mod files
gotk watch --interval 5s -p src -- make build   # Custom interval and path
```

## Pattern Learning

Teach GoTK which lines are noise in your project. After a few sessions, it suggests `always_remove` patterns for your `.gotk.toml`:

```bash
gotk learn run go test ./...        # Observe test output
gotk learn run make build           # Observe build output
gotk learn run cargo build          # Repeat a few times...
gotk learn suggest                  # Get pattern suggestions
gotk learn status                   # View observation stats
gotk learn clear                    # Reset observations
```

Passive mode — observe while working normally:

```bash
gotk --learn go test ./...
```

## Benchmarks

Measure GoTK's performance on realistic fixtures:

```bash
gotk bench                     # Full benchmark suite
gotk bench --per-filter        # Per-filter contribution breakdown
gotk bench --quality           # Quality score (% important lines preserved)
gotk bench --abtest            # Compare conservative/balanced/aggressive
gotk bench --json              # JSON output for CI
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
