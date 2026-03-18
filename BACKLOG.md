# GoTK Backlog

> Principles: DRY (Don't Repeat Yourself) · BMAD (Build, Measure, Adjust, Deliver)

## Legend

- `[x]` Done
- `[~]` In progress
- `[ ]` To do
- `[!]` Blocked

---

## Sprint 1 — MVP (Foundations) `DONE`

### Build

- [x] Init Go module + project structure
- [x] Exec runner — capture stdout/stderr/exit code
- [x] Filter chain — composable pipeline `func(string) string`
- [x] Generic filters: ANSI strip, whitespace normalize, dedup, paths compress, trim decorative
- [x] Command detection by binary name
- [x] CLI 3 modes: direct (`gotk cmd`), explicit (`gotk exec -- cmd`), pipe (`cmd | gotk`)
- [x] `--stats` flag to measure reduction

### Measure

- [x] Initial benchmarks: grep -95%, git log -90%, find -70%, ls -51%

### Adjust

- [x] Smart head+tail truncation (`--max-lines`)
- [x] Auto-detect command type in pipe mode
- [x] Improved grep filters (file grouping with `>> file` headers)
- [x] Improved ls filters (permissions+size+name only)
- [x] Improved git filters (strip redundant diff metadata)
- [x] Go test filters (summarize passing packages with names)

### Deliver

- [x] README.md
- [x] Documentation (architecture + filter catalog)
- [x] Unit tests
- [x] Config file (~/.config/gotk/config.toml)
- [x] Proxy shell mode (`--shell`, `-c "cmd"`)
- [x] Tag v0.1.0

---

## Sprint 2 — Quality Assurance + Advanced Filters `DONE`

### Build — Quality gates

- [x] Golden-file test framework (testdata/ with .input and .expected for each filter)
- [x] Quality validation: ensure no error/warning lines are ever removed
- [x] Quality validation: ensure file paths tied to errors are preserved
- [x] Quality validation: ensure structural indentation is preserved
- [x] Semantic line classifier (error / warning / info / debug / noise)
- [x] Priority-based filtering: errors and warnings are NEVER removed

### Build — Advanced filters

- [x] Go stack trace filter (condense repetitive traces, keep cause + top frame)
- [x] Python stack trace filter (same approach)
- [x] Node.js stack trace filter (same approach)
- [x] `docker` filter (build output, pull progress, compose)
- [x] `npm/yarn` filter (install output, audit, deprecation warnings)
- [x] `cargo` filter (build summary, download summary, preserve errors)
- [x] `make` filter (strip entering/leaving directory, compress gcc commands)

### Measure

- [x] Automated benchmark suite (golden files)
- [x] CI: measure reduction on realistic command corpus
- [x] Per-filter reduction report (which filter contributes how much)
- [x] Quality score: % of semantically important lines preserved

### Deliver

- [x] Documented integration examples
- [ ] Tag v0.2.0

---

## Sprint 3 — LLM Integrations `DONE`

### Build

- [x] MCP Server mode (Model Context Protocol) for Claude Code
- [x] Claude Code shell hook
- [x] Aider plugin/wrapper
- [x] Cursor plugin/wrapper
- [x] Continue.dev plugin/wrapper
- [x] Streaming mode (real-time filtering, not batch-only)

### Measure

- [x] Measure real token consumption impact (before/after on full sessions)
- [x] `gotk measure last [N]` — quick view of last N invocations
- [x] Auto-enable measurement in MCP mode (no config needed)
- [x] A/B test: mode comparison (`gotk bench --abtest`) — conservative/balanced/aggressive
- [x] Track cases where LLM re-requests info that was filtered out (re-request detection in MCP)

### Adjust

- [x] Feedback loop: quality insights in `gotk measure report` (re-request rate per command, actionable suggestions)
- [x] Whitelist/blacklist patterns to always keep/remove
- [x] Per-LLM profiles (Claude, GPT, Gemini) with MCP auto-detection

### Deliver

- [ ] Tag v0.3.0

---

## Sprint 4 — Security + Best Practices

### Build — Security hardening

- [x] Command denylist for MCP `gotk_exec` (block destructive commands)
- [x] Input size limits on all entry points (10MB cap, prevent OOM)
- [x] Output sanitization: redact secrets (API keys, tokens, passwords, JWTs)
- [x] MCP server: validate all JSON-RPC inputs strictly
- [x] MCP audit logging (all executed commands logged to stderr)
- [x] Sandbox mode: restrict executable commands to read-only operations
- [x] File-based audit log (opt-in)

### Build — Best practices

- [x] Output buffer size limits (LimitedBuffer, 10MB cap)
- [x] Graceful signal handling (SIGINT, SIGTERM) for clean shutdown
- [x] Timeout for command execution (30s default, configurable)
- [x] Context propagation for cancellation (RunWithTimeout, RunStream with ctx)
- [x] Eliminate global variable race conditions (TruncateWithLimit closure)
- [x] Cache os.Getwd/UserHomeDir at init (performance)
- [x] Package-level regex compilation (performance)
- [x] MCP denylist with word-boundary matching (no false positives)
- [x] Proper error types instead of raw strings
- [x] Fuzz testing on all filters (go test -fuzz)

### Build — CLI documentation & usage

- [x] Man page (`gotk.1`) — structured manual, installable
- [x] Improved `--help` — structured by subcommands with `gotk help <cmd>`
- [x] Guide "CLI vs MCP" (`docs/cli-vs-mcp.md`) — why CLI-first is more token-efficient
- [x] Quick-start recipes (`docs/quickstart.md`) per AI agent in CLI mode

### Deliver

- [x] Security documentation (docs/security.md)
- [x] Full audit + all fixes applied
- [ ] Tag v0.4.0

---

## Sprint 5 — Intelligence `DONE`

### Build

- [x] Structured summary for large outputs (error/warning counts, file paths, key error lines)
- [x] Watch mode: `gotk watch -- make test` (re-run + filter on file changes)
- [x] Cache: skip re-filtering identical output (content-hash based)

### Measure

- [x] Benchmark suite with 12 realistic fixtures (`gotk bench`)
- [x] Per-filter contribution analysis (`gotk bench --per-filter`)
- [x] Latency measurement with P50/P95/P99 (`gotk bench --json`)
- [x] Results: -87.5% avg reduction, 66ms total for 137KB corpus

### Adjust

- [x] Per-project config (.gotk.toml in repo, parent directory traversal)
- [ ] Project-specific pattern learning

### Deliver

- [ ] Tag v1.0.0
- [ ] Full documentation update
- [x] End-to-end integration tests (17 e2e tests on compiled binary)

---

## Sprint 6 — Polish + Release

### Build

- [ ] Project-specific pattern learning
- [ ] Full documentation update

### Deliver

- [ ] Tag v0.2.0, v0.3.0, v0.4.0 (catch-up)
- [ ] Tag v1.0.0
- [ ] Project landing page

---

## Sprint 7 — Context Search (`gotk ctx`)

> Integrate smart search capabilities inspired by `ai-ctx` into GoTK.
> GoTK already excels at cleaning output — this sprint adds the ability to **generate** token-optimized search results directly.

### Build — Core search engine

- [ ] `gotk ctx <keyword>` — scan mode: list files + occurrence count + truncated matches (default)
- [ ] `gotk ctx <keyword> -d [N]` — detail mode: show N lines of context around each match (default: 10)
- [ ] `gotk ctx <keyword> --def` — definition mode: target `func|class|struct|type|interface|const|var|trait|impl` declarations
- [ ] `gotk ctx <keyword> --tree` — structural skeleton of matching files (imports, types, functions)
- [ ] `gotk ctx <keyword> --summary` — occurrence distribution by directory with file counts
- [ ] Filtering options: `-t <type>` (file type), `-g <glob>` (glob pattern), `-m N` (max results)
- [ ] Search path option: `-p <path>` (default: `.`)
- [ ] Built-in exclusions (node_modules, .git, dist, vendor, __pycache__, lock files, .venv, coverage, .next)
- [ ] Fallback: `--def` with no results retries as standard keyword search

### Build — GoTK integration

- [ ] Apply existing GoTK filters to search output (ANSI strip, path compression, secret redaction)
- [ ] `--stats` works with `gotk ctx` (show token reduction vs raw grep output)
- [ ] MCP tool `gotk_ctx` — expose context search to LLM agents via MCP protocol
- [ ] Respect `.gotk.toml` config (custom exclusions, max-lines, filter mode)
- [ ] Overlap merging in detail mode: adjacent matches in same file are merged (no duplicate lines)

### Build — Smart output formatting

- [ ] Compact scan output: `3× src/auth/handler.go` + indented matches truncated to 120 chars
- [ ] Detail output: `--- file:line (lines X-Y/total) ---` headers with numbered context
- [ ] Tree output: language-aware skeleton extraction (Go, Python, JS/TS, Rust, Java)
- [ ] Summary output: directory breakdown table with example first match per top file
- [ ] All output modes apply existing truncation (head+tail) for large result sets

### Measure

- [ ] Benchmark: `gotk ctx` vs `grep -rn | gotk` — measure token savings of native search
- [ ] Benchmark: `gotk ctx --def` vs raw grep for definition search precision
- [ ] Add `gotk ctx` fixtures to `gotk bench` suite

### Deliver

- [ ] Man page section for `gotk ctx`
- [ ] `gotk help ctx` with examples
- [ ] Integration tests for all 5 modes (scan, detail, def, tree, summary)
- [ ] Tag v1.1.0

---

## Backlog (Unprioritized)

- [x] `--aggressive` / `--balanced` / `--conservative` filter modes
- [x] Per-command truncation threshold tuning
- [x] Whitelist/blacklist patterns to always keep/remove
- [x] Per-LLM profiles (Claude, GPT, Gemini)
- [x] Rate limiting in MCP server
- [x] CI pipeline with automated benchmarks

---

## Icebox (Ideas to Explore)

- [ ] Daemon mode to intercept all shell commands
- [ ] Windows support (PowerShell output patterns)
- [ ] HTTP API for remote integration
- [ ] VSCode plugin to visualize raw vs cleaned
- [ ] Prometheus/OpenTelemetry metrics (tokens saved over time)
- [ ] Multi-language detection support (localized error messages)
- [ ] Semantic compression via local lightweight LLM (summarize before sending to main model)
- [ ] Homebrew / AUR / scoop package publishing
