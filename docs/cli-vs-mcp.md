# CLI vs MCP: Choosing the Right Mode

## TL;DR

**CLI mode (hook/pipe/shell) is recommended for most users.** It's simpler, more token-efficient, and works with any AI coding tool. MCP mode is useful when you need the AI to discover GoTK's capabilities automatically.

---

## Why CLI Mode Uses Fewer Tokens

### 1. No JSON-RPC overhead

MCP wraps every request and response in JSON-RPC envelopes with metadata, tool schemas, and protocol framing. CLI mode is just stdin/stdout — zero overhead.

### 2. No tool discovery tokens

MCP exposes tool schemas (`gotk_exec`, `gotk_filter`, `gotk_read`, `gotk_grep`, `gotk_ctx`) that the LLM must process in its context window on every turn. CLI mode is invisible to the LLM — it never knows GoTK exists.

### 3. No extra tool calls

With MCP, the LLM must reason about which GoTK tool to call instead of a regular shell command. That reasoning costs tokens. With CLI hooks, filtering happens automatically — the LLM just runs commands normally.

### 4. Compartmentalized context

CLI mode processes output *before* it enters the LLM context. MCP mode processes output *inside* the LLM's tool-call flow, consuming context window space for the request envelope itself.

---

## Token Impact

| Mode | Extra tokens per command | Over 100 commands |
|------|--------------------------|-------------------|
| CLI (hook/pipe) | 0 | 0 |
| MCP | ~200-500 (schema + JSON-RPC + reasoning) | 20,000-50,000 |

CLI mode gives you GoTK's full filtering benefits with zero token cost for the integration itself.

---

## When to Use CLI Mode

- **Aider** — `SHELL=gotk` (100% automatic, true proxy)
- **Cursor** — `SHELL=gotk` or terminal profile (100% automatic)
- **Continue.dev** — Terminal context provider (100% automatic)
- **Any tool** that respects `$SHELL` (100% automatic)
- **Claude Code** — CLAUDE.md instructions (~95% reliable, CLI-only option)
- When you want **maximum token savings**

### How CLI automation works per tool

| Tool | Mechanism | Automation |
|------|-----------|------------|
| Aider | `SHELL=gotk` — Aider calls `$SHELL -c "cmd"` | 100% automatic |
| Cursor | `SHELL=gotk` or terminal profile | 100% automatic |
| Continue.dev | `shell: gotk -c` in config | 100% automatic |
| Claude Code | CLAUDE.md instructions | ~95% (Claude follows instructions) |

> Claude Code's Bash tool does not use `$SHELL` — it executes commands directly.
> This is why `SHELL=gotk` does not work for Claude Code. The CLAUDE.md approach
> is the best CLI option; for 100% guaranteed filtering, use MCP mode.

## When to Use MCP Mode

- **Claude Code** when you need 100% guaranteed filtering (not ~95%)
- When the AI tool doesn't support `$SHELL` replacement
- When you want the AI to **explicitly choose** when to use filtered output
- When you need `gotk_read` or `gotk_grep` specialized tools
- For **experimentation and debugging** (explicit tool calls are more visible)

---

## Quick Setup Comparison

### Aider, Cursor, Continue.dev (SHELL proxy — 100% automatic)

```bash
# Works for any tool that uses $SHELL
SHELL=/usr/local/bin/gotk GOTK_SHELL=/bin/bash aider
SHELL=/usr/local/bin/gotk GOTK_SHELL=/bin/bash cursor .
```

Result: every command the tool runs goes through GoTK. The LLM receives
only cleaned output. Zero overhead, zero awareness.

### Claude Code

**CLI (CLAUDE.md instructions):**

Add to your project's `CLAUDE.md`:

```markdown
## Token Optimization
Always use gotk to execute shell commands that produce verbose output:
  gotk grep -rn "pattern" .
  gotk go test ./...
Never run verbose commands without the gotk prefix.
```

Result: Claude prefixes its commands with `gotk`. ~95% compliance.

**MCP (100% automatic):**

```bash
claude mcp add --transport stdio gotk -- gotk --mcp
```

Result: the LLM sees 5 new tools. 100% filtering but with JSON-RPC token overhead.

---

## Measuring the Difference

Use `gotk measure` to compare token consumption between modes:

```bash
# Enable measurement
gotk --measure grep -rn "func" .

# Compare results
gotk measure report --period 7d
```

---

## Recommendation

Default to CLI mode. It provides the same filtering quality with zero token overhead. Reserve MCP mode for tools that don't support any other integration method.
