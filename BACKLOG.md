# GoTK Backlog

> Principles: DRY (Don't Repeat Yourself) · BMAD (Build, Measure, Adjust, Deliver)

## Legend

- `[x]` Done
- `[~]` In progress
- `[ ]` To do
- `[!]` Blocked

---

## Sprint 1 — MVP (Foundations)

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
- [x] Improved git filters (strip diff headers, Author/Date)
- [x] Go test filters (summarize passing packages)

### Deliver

- [x] README.md
- [x] Documentation (architecture + filter catalog)
- [x] Unit tests
- [x] Config file (~/.config/gotk/config.toml)
- [x] Proxy shell mode (`--shell`, `-c "cmd"`)
- [ ] First tag v0.1.0

---

## Sprint 2 — Quality Assurance + Advanced Filters

### Build — Quality gates

- [ ] Golden-file test framework (fixtures/ with .input and .expected for each filter)
- [ ] Quality validation: ensure no error/warning lines are ever removed
- [ ] Quality validation: ensure file paths tied to errors are preserved
- [ ] Quality validation: ensure structural indentation is preserved
- [ ] Semantic line classifier (error / warning / info / debug / noise)
- [ ] Priority-based filtering: errors and warnings are NEVER removed

### Build — Advanced filters

- [ ] Go stack trace filter (condense repetitive traces, keep cause + top frame)
- [ ] Python stack trace filter (same approach)
- [ ] Node.js stack trace filter (same approach)
- [ ] `docker` filter (build output, logs)
- [ ] `npm/yarn` filter (install output, audit)
- [ ] `cargo` filter (build, test)
- [ ] `make` filter (strip entering/leaving directory, keep errors)

### Measure

- [ ] Automated benchmark suite (fixtures + golden files)
- [ ] CI: measure reduction on realistic command corpus
- [ ] Per-filter reduction report (which filter contributes how much)
- [ ] Quality score: % of semantically important lines preserved (target: 100%)

### Adjust

- [ ] Per-command truncation threshold tuning
- [ ] `--aggressive` option for maximum reduction (acceptable info loss)
- [ ] `--conservative` option for minimal reduction (zero info loss)

### Deliver

- [ ] Tag v0.2.0
- [ ] Documented integration examples

---

## Sprint 3 — LLM Integrations

### Build

- [ ] MCP Server mode (Model Context Protocol) for Claude Code
- [ ] Claude Code shell hook
- [ ] Aider plugin/wrapper
- [ ] Cursor plugin/wrapper
- [ ] Continue.dev plugin/wrapper
- [ ] Streaming mode (real-time filtering, not batch-only)

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

## Sprint 4 — Intelligence

### Build

- [ ] Semantic content detection (error vs info vs debug)
- [ ] Priority filtering: keep errors and warnings over informational content
- [ ] Structured summary for very large outputs (>1000 lines): `[summary: 3 errors, 47 warnings, 2341 ok]`
- [ ] Cache: skip re-filtering identical output
- [ ] Watch mode: `gotk watch -- make test` (re-run + filter continuously)

### Measure

- [ ] Real token usage benchmarks via tiktoken/claude tokenizer
- [ ] Latency overhead measurement (target: <10ms for <1MB)

### Adjust

- [ ] Per-project config (.gotk.toml in repo)
- [ ] Project-specific pattern learning

### Deliver

- [ ] Tag v1.0.0
- [ ] Full documentation
- [ ] End-to-end integration tests

---

## Icebox (Ideas to Explore)

- [ ] Daemon mode to intercept all shell commands
- [ ] Windows support (PowerShell output patterns)
- [ ] HTTP API for remote integration
- [ ] VSCode plugin to visualize raw vs cleaned
- [ ] Prometheus/OpenTelemetry metrics (tokens saved over time)
- [ ] Multi-language detection support (localized error messages)
- [ ] Semantic compression via local lightweight LLM (summarize before sending to main model)
