# GoTK - filtered command output

Bash output may pass through GoTK, which strips noise (ANSI codes,
duplicate lines, progress output, framework stack frames) before you read
it. A filtered run ends with a `[gotk] ✔ exit 0` or `[gotk] ✘ exit N` line.

- Errors, warnings, and stack frames from project code are never removed:
  do not re-run a command just to see "the full output".
- Long output keeps its head and tail around a `[... N lines omitted ...]`
  marker. If the part you need was cut, narrow the command or re-run it
  with `GOTK_PASSTHROUGH=1 <cmd>` to get it unfiltered.
- Markers such as `[... 11 node_modules frames]` stand for removed lines;
  they are counts, not errors.
- `gotk ctx <pattern>` searches the codebase with compact output
  (`--def` for declarations, `-d 5` for 5-line context windows).
