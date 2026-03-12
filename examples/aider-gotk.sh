#!/bin/sh
# GoTK wrapper for Aider
#
# Uses GoTK as the shell for Aider, so all command output is automatically
# filtered before being sent to the LLM. This reduces token usage by
# stripping noise (ANSI codes, duplicate lines, decorative output, etc.).
#
# Installation:
#   1. Build gotk:  go build -o gotk ./cmd/gotk/
#   2. Place gotk in your PATH (e.g., /usr/local/bin/gotk)
#   3. Run this script instead of aider directly
#
# Usage:
#   ./aider-gotk.sh                          # start aider normally
#   ./aider-gotk.sh --model claude-3-opus    # pass any aider flags
#   ./aider-gotk.sh --dark-mode              # all flags forwarded to aider
#
# How it works:
#   Setting SHELL to gotk means Aider will use `gotk -c "command"` to
#   execute shell commands. GoTK intercepts the output, filters it, and
#   returns the cleaned version. The LLM sees less noise, fewer tokens.
#
# Environment variables:
#   GOTK_PASSTHROUGH=1   Disable all filtering (escape hatch)
#   GOTK_SHELL=/bin/bash Override the real shell gotk uses internally
#                        (defaults to /bin/bash or /bin/sh)

set -e

# Locate gotk binary
GOTK_BIN="${GOTK_BIN:-gotk}"

if ! command -v "$GOTK_BIN" >/dev/null 2>&1; then
    echo "Error: gotk not found in PATH. Build it with: go build -o gotk ./cmd/gotk/" >&2
    echo "Then place it in your PATH or set GOTK_BIN=/path/to/gotk" >&2
    exit 1
fi

# Resolve to absolute path (required for SHELL)
GOTK_PATH="$(command -v "$GOTK_BIN")"

# Set SHELL to gotk so Aider uses it for command execution
export SHELL="$GOTK_PATH"

# Preserve the real shell for gotk's internal use
if [ -z "$GOTK_SHELL" ]; then
    export GOTK_SHELL="${SHELL_ORIG:-/bin/bash}"
fi

exec aider "$@"
