# GoTK Integrations

How to use GoTK with various LLM coding tools to reduce token usage.

GoTK works with any tool that executes shell commands, through three mechanisms:

1. **Pipe mode** — `command | gotk` (for hooks and manual use)
2. **Shell proxy** — `SHELL=gotk` (for tools that spawn a shell)
3. **Direct execution** — `gotk exec command` (for explicit wrapping)

---

## Claude Code

### Method 1: Hook (recommended)

Claude Code supports output hooks that process command output before sending it to the model. GoTK's pipe mode is a perfect fit.

Add to `~/.claude/settings.json`:

```json
{
  "hooks": {
    "shell_command_output": [
      {
        "matcher": "",
        "command": "gotk"
      }
    ]
  }
}
```

This sends all command output through `gotk` in pipe mode. GoTK auto-detects the command type from the output format and applies appropriate filters.

For a wrapper script with fallback handling, see `examples/claude-code-hook.sh`.

### Method 2: MCP Server

If Claude Code supports MCP tool servers, you can register GoTK as a tool that wraps command execution:

```json
{
  "mcpServers": {
    "gotk": {
      "command": "gotk",
      "args": ["--shell"]
    }
  }
}
```

This starts GoTK in proxy shell mode, where it reads commands from stdin, executes them, and returns filtered output.

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

GoTK reads configuration from:

1. `./gotk.toml` (project-local, highest priority)
2. `~/.config/gotk/config.toml` (global)

Key settings for integrations:

```toml
[general]
stats = false        # keep false for hook/pipe use — stats go to stderr
max_lines = 50       # truncation limit (0 = unlimited)
shell_mode = false   # set true to default to shell mode

[filters]
strip_ansi = true
normalize_whitespace = true
dedup = true
trim_decorative = true
truncate = true
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
