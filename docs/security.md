# GoTK Security

This document describes the security measures built into GoTK to protect users when using it as an LLM output proxy.

## Secret Redaction

GoTK automatically detects and redacts potential secrets from command output before sending it to an LLM. This prevents accidental leakage of credentials, API keys, and other sensitive data.

**What is redacted:**

- **API keys and tokens** with known prefixes: `sk-...` (OpenAI/Stripe), `ghp_...` / `ghu_...` / `ghs_...` (GitHub), `glpat-...` (GitLab), `xoxb-...` / `xoxp-...` (Slack)
- **AWS access key IDs**: patterns matching `AKIA` followed by 16 uppercase alphanumeric characters
- **JWT tokens**: base64-encoded tokens in the `eyJ...` format (header.payload.signature)
- **PEM private keys**: `-----BEGIN ... PRIVATE KEY-----` blocks
- **Connection string passwords**: passwords in URLs like `://user:password@host` (only the password is redacted)
- **Environment variable values** where the key name contains `KEY`, `SECRET`, `TOKEN`, `PASSWORD`, or `APIKEY` (case-insensitive). The key name is preserved; only the value is replaced with `[REDACTED]`.

**Example:**

```
# Input
AWS_SECRET_KEY=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY
DATABASE_URL=postgres://admin:s3cretP4ss@db.example.com/mydb

# Output (redacted)
AWS_SECRET_KEY=[REDACTED]
DATABASE_URL=postgres://admin:[REDACTED]@db.example.com/mydb
```

Secret redaction is enabled by default. To disable it:

```toml
[security]
redact_secrets = false
```

## Command Timeout

All commands executed via the MCP server have a configurable timeout. This prevents runaway commands from consuming resources indefinitely.

- **Default timeout**: 30 seconds
- Commands that exceed the timeout are killed and return exit code 124
- The timeout applies to the entire command execution, including any subprocesses

Configure the timeout in your config file:

```toml
[security]
command_timeout = 60  # seconds, 0 to use default
```

## Output Size Limits

stdout and stderr output from commands is capped at 10MB each. If a command produces more output than the limit, the output is truncated and a marker is appended:

```
[output truncated: exceeded 10MB limit]
```

This prevents memory exhaustion from commands that produce extremely large output.

Configure the limit:

```toml
[security]
max_output_bytes = 10485760  # 10MB in bytes
```

## MCP Command Validation

When GoTK runs as an MCP server, it validates commands against a denylist of clearly destructive operations before execution. Blocked commands include:

- `rm -rf /` and variants
- `mkfs` (filesystem formatting)
- `dd` (raw disk writes)
- `format` (Windows disk formatting)
- Fork bombs
- `chmod -R 777 /`
- `shutdown`, `reboot`, `halt`
- Disk wiping commands

If a command matches the denylist, it is rejected with an error message and logged to stderr.

**Note:** GoTK uses a denylist (not an allowlist) to avoid being too restrictive for general use. The denylist targets commands that are almost never intentional in an LLM-assisted workflow.

## MCP Input Size Validation

MCP requests with input exceeding 10MB are rejected before processing. This prevents denial-of-service via extremely large payloads.

## Audit Logging

All commands executed through the MCP server are logged to stderr with the prefix `[gotk-mcp] EXEC:`. This provides an audit trail of what commands were run:

```
[gotk-mcp] EXEC: grep -rn "func main" ./src/
[gotk-mcp] BLOCKED command: "rm -rf /" - command blocked: contains dangerous pattern "rm -rf /"
```

## Graceful Shutdown

GoTK handles SIGINT and SIGTERM signals:

- Catches the signal and logs a shutdown message
- Sends SIGTERM to the process group to clean up child processes
- Exits with the conventional exit code (130 for SIGINT, 143 for SIGTERM)

## Configuration Reference

All security settings live under the `[security]` section in your config file:

```toml
[security]
command_timeout = 30       # Command timeout in seconds (default: 30)
max_output_bytes = 10485760  # Max bytes per output stream (default: 10MB)
redact_secrets = true      # Redact secrets from output (default: true)
```

Config file locations:
- Global: `~/.config/gotk/config.toml`
- Local (takes precedence): `./gotk.toml`

## Recommendations for Production Use

1. **Keep secret redaction enabled.** It is on by default. Only disable it if you are certain no secrets will appear in command output.

2. **Set appropriate timeouts.** The default 30-second timeout is suitable for most commands. Increase it for long-running builds or tests.

3. **Review the audit log.** When running GoTK as an MCP server, monitor stderr for the commands being executed.

4. **Use `GOTK_SHELL` to control the shell.** Set `GOTK_SHELL=/bin/bash` (or another trusted shell) to explicitly control which shell executes commands.

5. **Run with least privilege.** Run GoTK under a user account with minimal permissions appropriate for the task.

6. **Do not rely solely on the command denylist.** The denylist catches obviously destructive commands, but it cannot catch all possible harmful commands. Use OS-level controls (containers, sandboxes, restricted users) for defense in depth.
