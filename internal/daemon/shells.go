package daemon

// zshInit is the zsh init script template injected into the daemon shell session.
// Placeholders: GOTK_BIN_PLACEHOLDER, SESSION_ID_PLACEHOLDER, ORIG_ZDOTDIR_PLACEHOLDER
const zshInit = `
# GoTK daemon — zsh init
# This file is auto-generated. Do not edit.

# Restore original ZDOTDIR and source user config
ZDOTDIR="ORIG_ZDOTDIR_PLACEHOLDER"
[[ -f "${ZDOTDIR:-$HOME}/.zshrc" ]] && source "${ZDOTDIR:-$HOME}/.zshrc"

# Daemon environment
export GOTK_DAEMON=1
export GOTK_SESSION_ID="SESSION_ID_PLACEHOLDER"
export GOTK_BIN="GOTK_BIN_PLACEHOLDER"

# Prompt modification
PROMPT="[gotk] ${PROMPT}"

# Interactive commands that need direct terminal access
typeset -A _gotk_interactive
for cmd in vim vi nvim nano emacs less more man top htop btop watch ssh mosh tmux screen fzf nnn ranger mc lazygit lazydocker k9s; do
  _gotk_interactive[$cmd]=1
done

# Trivial commands (no output to filter)
typeset -A _gotk_trivial
for cmd in cd pwd echo printf export source which type true false set unset alias hash read return exit logout exec popd pushd dirs bg fg jobs disown; do
  _gotk_trivial[$cmd]=1
done

# gotk wrapper: intercept "gotk exit" to leave daemon session
gotk() {
  if [[ "$1" == "exit" ]]; then
    exit "${2:-0}"
  fi
  command "$GOTK_BIN" "$@"
}

# Override accept-line to intercept commands
_gotk_accept_line() {
  local cmd="$BUFFER"

  # Empty command
  [[ -z "${cmd// /}" ]] && { zle .accept-line; return; }

  # Extract first word (skip env var assignments)
  local first_word
  for word in ${(z)cmd}; do
    if [[ "$word" != *=* || "$word" == */* ]]; then
      first_word="${word:t}"
      break
    fi
  done

  # Skip interactive and trivial commands
  if [[ -n "${_gotk_interactive[$first_word]}" ]] || [[ -n "${_gotk_trivial[$first_word]}" ]]; then
    zle .accept-line
    return
  fi

  # Skip gotk itself (prevent recursion)
  if [[ "$first_word" == "gotk" ]]; then
    zle .accept-line
    return
  fi

  # Skip if already piped through gotk
  if [[ "$cmd" == *"| $GOTK_BIN"* ]] || [[ "$cmd" == *"|$GOTK_BIN"* ]] || [[ "$cmd" == *"| gotk"* ]]; then
    zle .accept-line
    return
  fi

  # Wrap with gotk -c (handles stderr pass-through, exit codes)
  BUFFER="$GOTK_BIN --measure -c ${(qq)cmd}"
  zle .accept-line
}
zle -N accept-line _gotk_accept_line

# Print session summary on exit
_gotk_cleanup() {
  "$GOTK_BIN" daemon summary --session "$GOTK_SESSION_ID" 2>/dev/null
}
trap '_gotk_cleanup' EXIT
`

// bashInit is the bash init script template injected into the daemon shell session.
// Placeholders: GOTK_BIN_PLACEHOLDER, SESSION_ID_PLACEHOLDER
const bashInit = `
# GoTK daemon — bash init
# This file is auto-generated. Do not edit.

# Source user config
[ -f ~/.bashrc ] && source ~/.bashrc

# Daemon environment
export GOTK_DAEMON=1
export GOTK_SESSION_ID="SESSION_ID_PLACEHOLDER"
export GOTK_BIN="GOTK_BIN_PLACEHOLDER"

# Prompt modification
PS1="[gotk] $PS1"

# Interactive commands that need direct terminal access
_gotk_interactive=" vim vi nvim nano emacs less more man top htop btop watch ssh mosh tmux screen fzf nnn ranger mc lazygit lazydocker k9s "

# Trivial commands (no output to filter)
_gotk_trivial=" cd pwd echo printf export source which type true false set unset alias hash read return exit logout exec popd pushd dirs bg fg jobs disown "

# gotk wrapper: intercept "gotk exit" to leave daemon session
gotk() {
  if [ "$1" = "exit" ]; then
    exit "${2:-0}"
  fi
  command "$GOTK_BIN" "$@"
}

_gotk_should_skip() {
  local cmd="$1"
  local first_word

  # Extract first word, skipping env var assignments
  for word in $cmd; do
    case "$word" in
      *=*) continue ;;  # skip VAR=val
      *)   first_word="${word##*/}"; break ;;  # basename
    esac
  done

  [ -z "$first_word" ] && return 0

  # Skip gotk itself
  [ "$first_word" = "gotk" ] && return 0

  # Skip interactive and trivial
  [[ "$_gotk_interactive" == *" $first_word "* ]] && return 0
  [[ "$_gotk_trivial" == *" $first_word "* ]] && return 0

  # Skip if already piped through gotk
  [[ "$cmd" == *"| $GOTK_BIN"* ]] && return 0
  [[ "$cmd" == *"|$GOTK_BIN"* ]] && return 0
  [[ "$cmd" == *"| gotk"* ]] && return 0

  return 1
}

# Use extdebug + DEBUG trap to intercept commands
shopt -s extdebug
_gotk_last_exit=0

_gotk_debug_trap() {
  # Only intercept top-level commands
  [ "$BASH_SUBSHELL" -gt 0 ] && return 0

  # Skip during completion
  [ -n "$COMP_LINE" ] && return 0

  local cmd="$BASH_COMMAND"

  _gotk_should_skip "$cmd" && return 0

  # Run through gotk, then skip shell execution (return 1)
  eval "$GOTK_BIN --measure -c $(printf '%q' "$cmd")"
  _gotk_last_exit=$?
  return 1
}

trap '_gotk_debug_trap' DEBUG

# Restore exit code after interception
_gotk_orig_prompt_command="${PROMPT_COMMAND}"
PROMPT_COMMAND='(exit $_gotk_last_exit) 2>/dev/null; '"${_gotk_orig_prompt_command}"

# Print session summary on exit
_gotk_cleanup() {
  "$GOTK_BIN" daemon summary --session "$GOTK_SESSION_ID" 2>/dev/null
}
trap '_gotk_cleanup' EXIT
`
