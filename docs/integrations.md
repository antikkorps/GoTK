# GoTK Integrations

How to use GoTK with various LLM coding tools to reduce token usage.

GoTK works with any tool that executes shell commands, through three mechanisms:

1. **Shell proxy** — `SHELL=gotk` (100% automatic for tools that spawn a shell)
2. **CLAUDE.md instructions** — Tell Claude to prefix commands with `gotk` (for Claude Code)
3. **Direct execution / pipe** — `gotk cmd` or `cmd | gotk` (manual or scripted)

> **How to choose?** Most AI tools use `$SHELL -c "command"` to run commands.
> Setting `SHELL=gotk` intercepts every command automatically — the LLM never
> knows GoTK is running. Claude Code is the exception: its Bash tool does not
> use `$SHELL`, so it requires a different approach (CLAUDE.md or MCP).
> See [cli-vs-mcp.md](cli-vs-mcp.md) for a detailed comparison.

---

## Claude Code

### Method 1: CLAUDE.md instructions (recommended CLI approach)

Claude Code reads `CLAUDE.md` at the project root on every conversation. Add
instructions telling Claude to use GoTK for all shell commands:

Add to your project's `CLAUDE.md`:

```markdown
## Token Optimization

Always use gotk to execute shell commands that produce verbose output:
  gotk grep -rn "pattern" .
  gotk go test ./...
  gotk git log --oneline -20
  gotk find . -name "*.go"
  gotk make build 2>&1

Never run grep, find, git, go test, make, or other verbose commands without
the gotk prefix. This reduces token consumption by ~80%.
```

A ready-to-copy template is available at `examples/CLAUDE.md.gotk-template`.

> **Note:** This approach is ~95% reliable — Claude follows CLAUDE.md instructions
> consistently but not 100% of the time. For guaranteed filtering, use MCP mode.

### Method 2: MCP Server (100% automatic)

Register GoTK as an MCP tool server. This exposes four tools that Claude can use with filtered output:

| Tool | Description |
|------|-------------|
| `gotk_exec` | Execute any command and return cleaned output |
| `gotk_filter` | Filter pre-existing text through the cleaning pipeline |
| `gotk_read` | Read a file with smart truncation and noise removal |
| `gotk_grep` | Search file contents with grouped, compressed results |

`gotk_read` and `gotk_grep` are more token-efficient than using `gotk_exec` with `cat`/`grep` because they apply specialized defaults (higher truncation limits for reads, grep-specific output grouping).

**Via CLI (recommended):**

```bash
claude mcp add --transport stdio --scope project gotk -- /usr/local/bin/gotk --mcp
```

Use `--scope local` (default) to enable for all projects, or `--scope project` to enable only for the current project (creates `.mcp.json` at the project root).

**Via `.mcp.json` at the project root:**

```json
{
  "mcpServers": {
    "gotk": {
      "command": "/usr/local/bin/gotk",
      "args": ["--mcp"]
    }
  }
}
```

> **Note:** The `.mcp.json` file must be at the **project root**, not inside `.claude/`.

---

## Aider

Aider uses `$SHELL -c "command"` to execute shell commands. Set `SHELL` to GoTK:

```bash
export SHELL=/usr/local/bin/gotk
export GOTK_SHELL=/bin/bash   # so gotk knows which real shell to use
aider
```

Or use the wrapper script:

```bash
./examples/aider-gotk.sh --model claude-3-opus
```

The wrapper handles locating `gotk`, setting environment variables, and forwarding all arguments to `aider`.

---

## Cursor

### Option A: Terminal profile

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

For Linux, replace `osx` with `linux`.

### Option B: Environment wrapper

```bash
export SHELL=/usr/local/bin/gotk
export GOTK_SHELL=/bin/bash
cursor .
```

Or use `examples/cursor-gotk.sh`.

---

## Continue.dev

In `~/.continue/config.json`, configure the terminal context provider:

```json
{
  "contextProviders": [
    {
      "name": "terminal",
      "params": {
        "shell": "/usr/local/bin/gotk -c"
      }
    }
  ]
}
```

Or set `SHELL` in your environment before launching the editor.

---

## Generic LLM Tool Integration

Any tool that executes shell commands can use GoTK. There are three patterns:

### Pattern 1: SHELL replacement

If the tool respects `$SHELL`:

```bash
export SHELL=/usr/local/bin/gotk
export GOTK_SHELL=/bin/bash
your-llm-tool
```

GoTK handles `-c "command"` like a regular shell, but filters the output.

### Pattern 2: Pipe filter

If the tool lets you post-process output:

```bash
some-command | gotk
```

GoTK reads from stdin, auto-detects the command type from the output format, applies filters, and writes cleaned output to stdout.

### Pattern 3: Direct execution

Wrap the command explicitly:

```bash
gotk exec -- grep -rn "pattern" .
gotk find . -name "*.go"
```

GoTK executes the command, identifies its type, filters the output, and prints the result.

---

## Environment Variables

| Variable | Description |
|---|---|
| `GOTK_PASSTHROUGH=1` | Disable all filtering. Output passes through unchanged. Use as an escape hatch when debugging or when you need raw output. |
| `GOTK_SHELL=/bin/bash` | The real shell GoTK uses internally to execute commands. Defaults to `$SHELL` (if not gotk itself), then `/bin/bash`, then `/bin/sh`. |
| `GOTK_BIN=gotk` | Used by wrapper scripts to locate the gotk binary. |

---

## Configuration

GoTK reads configuration from three levels (in order of precedence):

1. `~/.config/gotk/config.toml` (global defaults)
2. `.gotk.toml` (project config — found by walking up parent directories)
3. `./gotk.toml` (local override, highest priority)

Key settings for integrations:

```toml
[general]
mode = "balanced"    # conservative | balanced | aggressive
stats = false        # keep false for hook/pipe use — stats go to stderr
max_lines = 50       # truncation limit (0 = unlimited)
shell_mode = false   # set true to default to shell mode

[filters]
strip_ansi = true
normalize_whitespace = true
dedup = true
trim_decorative = true
truncate = true

[rules]
always_keep = ["^ERROR:", "^FATAL:"]     # regex: never remove matching lines
always_remove = ["^DEBUG:", "^TRACE:"]   # regex: always remove matching lines

[truncation]
grep = 30       # per-command max_lines overrides
test = 200      # more lines for test output
git = 100

[security]
command_timeout = 300   # 5 min — important for MCP with test suites
rate_limit = 0          # max MCP tool calls/min (0 = disabled)
rate_burst = 10         # burst size for rate limiter
sandbox_mode = false    # restrict MCP to read-only commands
```

---

## Troubleshooting

### gotk not found

Ensure `gotk` is in your `$PATH`:

```bash
go build -o /usr/local/bin/gotk ./cmd/gotk/
```

Or set `GOTK_BIN` to the full path in wrapper scripts.

### Recursive shell (gotk calling itself)

If you set `SHELL=gotk`, GoTK detects this and falls back to `/bin/bash` or `/bin/sh` for internal command execution. You can also set `GOTK_SHELL` explicitly:

```bash
export GOTK_SHELL=/bin/bash
```

### Output is empty or truncated too aggressively

1. Check `max_lines` in your config (default: 50 lines, keeps head + tail)
2. Temporarily disable filtering: `GOTK_PASSTHROUGH=1`
3. Use `--no-truncate` to disable line truncation
4. Use `--stats` to see how much was removed

### Stats appearing in LLM output

Stats are written to stderr, not stdout. If your tool captures stderr, set `stats = false` in config (this is the default).

### Filtering removes important output

GoTK prioritizes output quality over token reduction. If you find a case where important information is removed, it is a bug. Set `GOTK_PASSTHROUGH=1` as a workaround and report the issue.
