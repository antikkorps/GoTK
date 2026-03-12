#!/bin/sh
# GoTK wrapper for Cursor and Continue.dev
#
# Uses GoTK as the shell so all command output from AI-driven tools is
# filtered before being sent to the LLM, reducing token usage.
#
# ──────────────────────────────────────────────────────────────────
# Cursor Setup
# ──────────────────────────────────────────────────────────────────
#
# Option A: Set terminal shell in Cursor settings (settings.json):
#
#   {
#     "terminal.integrated.shell.linux": "/usr/local/bin/gotk",
#     "terminal.integrated.shell.osx": "/usr/local/bin/gotk",
#     "terminal.integrated.shellArgs.linux": ["--shell"],
#     "terminal.integrated.shellArgs.osx": ["--shell"]
#   }
#
# Option B: Use the terminal profile approach:
#
#   {
#     "terminal.integrated.profiles.osx": {
#       "gotk": {
#         "path": "/usr/local/bin/gotk",
#         "args": ["--shell"]
#       }
#     },
#     "terminal.integrated.defaultProfile.osx": "gotk"
#   }
#
# Option C: Run this wrapper script as the shell:
#   Set terminal.integrated.shell.* to the path of this script.
#
# ──────────────────────────────────────────────────────────────────
# Continue.dev Setup
# ──────────────────────────────────────────────────────────────────
#
# In your Continue.dev config (~/.continue/config.json), set the
# shell command to use gotk:
#
#   {
#     "allowAnonymousTelemetry": false,
#     "models": [ ... ],
#     "customCommands": [],
#     "contextProviders": [
#       {
#         "name": "terminal",
#         "params": {
#           "shell": "/usr/local/bin/gotk -c"
#         }
#       }
#     ]
#   }
#
# Alternatively, set SHELL in your environment before launching your editor:
#
#   export SHELL=/usr/local/bin/gotk
#   export GOTK_SHELL=/bin/bash
#   cursor .
#
# ──────────────────────────────────────────────────────────────────
# Usage
# ──────────────────────────────────────────────────────────────────
#
#   ./cursor-gotk.sh              # launches Cursor with gotk as shell
#   ./cursor-gotk.sh /path/to/dir # opens a specific directory
#
# Environment variables:
#   GOTK_PASSTHROUGH=1   Disable all filtering (escape hatch)
#   GOTK_SHELL=/bin/bash Override the real shell gotk uses internally

set -e

# Locate gotk binary
GOTK_BIN="${GOTK_BIN:-gotk}"

if ! command -v "$GOTK_BIN" >/dev/null 2>&1; then
    echo "Error: gotk not found in PATH. Build it with: go build -o gotk ./cmd/gotk/" >&2
    echo "Then place it in your PATH or set GOTK_BIN=/path/to/gotk" >&2
    exit 1
fi

GOTK_PATH="$(command -v "$GOTK_BIN")"

# Set SHELL to gotk
export SHELL="$GOTK_PATH"

# Preserve the real shell for gotk's internal use
if [ -z "$GOTK_SHELL" ]; then
    export GOTK_SHELL="${SHELL_ORIG:-/bin/bash}"
fi

# Launch Cursor (adjust the command if cursor is installed elsewhere)
if command -v cursor >/dev/null 2>&1; then
    exec cursor "$@"
else
    echo "Error: cursor not found in PATH." >&2
    echo "You can also source this script to set the environment:" >&2
    echo "  . ./cursor-gotk.sh && cursor ." >&2
    exit 1
fi
