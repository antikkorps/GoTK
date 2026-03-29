#!/bin/sh
# GoTK hook for Claude Code — PreToolUse Bash output filter
#
# This hook is called by Claude Code before executing Bash commands.
# It wraps commands with "| gotk" so output is filtered before Claude sees it,
# reducing token usage by ~80%.
#
# RECOMMENDED: Use "gotk install claude" instead of manual setup.
#
# Manual installation:
#   1. Build gotk:  go build -o gotk ./cmd/gotk/
#   2. Place gotk in your PATH (e.g., /usr/local/bin/gotk)
#   3. Add this hook to your Claude Code settings:
#
#      ~/.claude/settings.json (global) or .claude/settings.json (project):
#
#      {
#        "hooks": {
#          "PreToolUse": [
#            {
#              "matcher": "Bash",
#              "hooks": [
#                {
#                  "type": "command",
#                  "command": "gotk hook"
#                }
#              ]
#            }
#          ]
#        }
#      }
#
# How it works:
#   1. Claude Code fires a PreToolUse event before running a Bash command
#   2. The hook receives a JSON payload on stdin with the command
#   3. GoTK wraps the command: "set -o pipefail; (cmd) | gotk"
#   4. Claude Code executes the wrapped command
#   5. Claude receives filtered output (ANSI stripped, deduplicated, truncated)
#
# Trivial commands (cd, pwd, echo, etc.) are not wrapped.
# Commands already using gotk are not double-wrapped.
#
# Environment variables:
#   GOTK_PASSTHROUGH=1   Disable all filtering (escape hatch)
#   GOTK_SHELL=/bin/bash Override the shell gotk uses internally

set -e

GOTK_BIN="${GOTK_BIN:-gotk}"

if command -v "$GOTK_BIN" >/dev/null 2>&1; then
    exec "$GOTK_BIN" hook
else
    # Fallback: if gotk is not installed, exit silently (no modification)
    exit 0
fi
