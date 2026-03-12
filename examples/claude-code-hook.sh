#!/bin/sh
# GoTK hook for Claude Code — shell command output filter
#
# This hook filters all command output through GoTK before sending to Claude,
# reducing token usage by stripping ANSI codes, deduplicating lines,
# compressing paths, and truncating excessive output.
#
# Installation:
#   1. Build gotk:  go build -o gotk ./cmd/gotk/
#   2. Place gotk in your PATH (e.g., /usr/local/bin/gotk)
#   3. Add this hook to your Claude Code settings (~/.claude/settings.json):
#
#      {
#        "hooks": {
#          "shell_command_output": [
#            {
#              "matcher": "",
#              "command": "/path/to/examples/claude-code-hook.sh"
#            }
#          ]
#        }
#      }
#
#   Or use gotk directly as the command (simpler, no wrapper needed):
#
#      {
#        "hooks": {
#          "shell_command_output": [
#            {
#              "matcher": "",
#              "command": "gotk"
#            }
#          ]
#        }
#      }
#
# How it works:
#   Claude Code sends command output on stdin to this hook.
#   GoTK filters the output (strip ANSI, normalize whitespace, dedup,
#   compress paths, truncate) and writes the cleaned result to stdout.
#   Claude receives the filtered output instead of the raw version.
#
# Environment variables:
#   GOTK_PASSTHROUGH=1   Disable all filtering (escape hatch)
#   GOTK_SHELL=/bin/bash Override the shell gotk uses internally

set -e

# Locate gotk binary
GOTK_BIN="${GOTK_BIN:-gotk}"

if command -v "$GOTK_BIN" >/dev/null 2>&1; then
    # Pipe stdin through gotk (pipe mode: auto-detects command type)
    exec "$GOTK_BIN"
else
    # Fallback: if gotk is not installed, pass through raw output
    exec cat
fi
