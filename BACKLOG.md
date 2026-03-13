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

- [ ] Measure real token consumption impact (before/after on full sessions)
- [ ] A/B test: LLM response quality with/without GoTK
- [ ] Track cases where LLM re-requests info that was filtered out (quality regression signal)

### Adjust

- [ ] Feedback loop: are some removed lines re-requested by the LLM?
- [ ] Whitelist/blacklist patterns to always keep/remove
- [ ] Per-LLM profiles (Claude, GPT, Gemini — different needs?)

### Deliver

- [ ] Tag v0.3.0
- [ ] Project landing page
- [ ] Package registry publishing (Homebrew, AUR, etc.)

---

## Sprint 4 — Security + Best Practices `DONE`

### Build — Security hardening

- [x] Command denylist for MCP `gotk_exec` (block destructive commands)
- [x] Input size limits on all entry points (10MB cap, prevent OOM)
- [x] Output sanitization: redact secrets (API keys, tokens, passwords, JWTs)
- [x] MCP server: validate all JSON-RPC inputs strictly
- [x] MCP audit logging (all executed commands logged to stderr)
- [ ] Sandbox mode: restrict executable commands to read-only operations
- [ ] File-based audit log (opt-in)

### Build — Best practices

- [x] Output buffer size limits (LimitedBuffer, 10MB cap)
- [x] Graceful signal handling (SIGINT, SIGTERM) for clean shutdown
- [x] Timeout for command execution (30s default, configurable)
- [x] Context propagation for cancellation (RunWithTimeout, RunStream with ctx)
- [x] Eliminate global variable race conditions (TruncateWithLimit closure)
- [x] Cache os.Getwd/UserHomeDir at init (performance)
- [x] Package-level regex compilation (performance)
- [x] MCP denylist with word-boundary matching (no false positives)
- [ ] Proper error types instead of raw strings
- [ ] Fuzz testing on all filters (go test -fuzz)

### Deliver

- [x] Security documentation (docs/security.md)
- [x] Full audit + all fixes applied
- [ ] Tag v0.4.0

---

## Sprint 5 — Intelligence `DONE`

### Build

- [x] Structured summary for large outputs (error/warning counts, file paths, key error lines)
- [x] Watch mode: `gotk watch -- make test` (re-run + filter on file changes)
- [ ] Cache: skip re-filtering identical output (content-hash based)

### Measure

- [x] Benchmark suite with 12 realistic fixtures (`gotk bench`)
- [x] Per-filter contribution analysis (`gotk bench --per-filter`)
- [x] Latency measurement with P50/P95/P99 (`gotk bench --json`)
- [x] Results: -87.5% avg reduction, 66ms total for 137KB corpus

### Adjust

- [ ] Per-project config (.gotk.toml in repo)
- [ ] Project-specific pattern learning

### Deliver

- [ ] Tag v1.0.0
- [ ] Full documentation update
- [ ] End-to-end integration tests

---

## Backlog (Unprioritized)

- [ ] `--aggressive` option for maximum reduction (acceptable info loss)
- [ ] `--conservative` option for minimal reduction (zero info loss)
- [ ] Per-command truncation threshold tuning
- [ ] Whitelist/blacklist patterns to always keep/remove
- [ ] Per-LLM profiles (Claude, GPT, Gemini)
- [ ] Rate limiting in MCP server
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
- [ ] Project landing page
