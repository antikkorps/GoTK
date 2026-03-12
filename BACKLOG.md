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

## Sprint 2 — Quality Assurance + Advanced Filters

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
- [ ] `docker` filter (build output, logs)
- [ ] `npm/yarn` filter (install output, audit)
- [ ] `cargo` filter (build, test)
- [ ] `make` filter (strip entering/leaving directory, keep errors)

### Measure

- [x] Automated benchmark suite (golden files)
- [ ] CI: measure reduction on realistic command corpus
- [ ] Per-filter reduction report (which filter contributes how much)
- [ ] Quality score: % of semantically important lines preserved (target: 100%)

### Adjust

- [ ] Per-command truncation threshold tuning
- [ ] `--aggressive` option for maximum reduction (acceptable info loss)
- [ ] `--conservative` option for minimal reduction (zero info loss)

### Deliver

- [ ] Tag v0.2.0
- [x] Documented integration examples

---

## Sprint 3 — LLM Integrations

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

## Sprint 4 — Security + Best Practices

### Build — Security hardening

- [~] Command allowlist for MCP `gotk_exec` (prevent arbitrary command execution)
- [~] Input size limits on all entry points (prevent memory exhaustion)
- [~] Output sanitization: never leak secrets (.env values, tokens, API keys) in filtered output
- [~] MCP server: validate all JSON-RPC inputs strictly
- [~] Rate limiting in MCP server (prevent abuse in shared environments)
- [ ] Sandbox mode: restrict executable commands to read-only operations
- [ ] Audit log: log all commands executed via MCP/proxy (opt-in, to file)

### Build — Best practices

- [~] Output buffer size limits (prevent OOM on huge outputs)
- [~] Graceful signal handling (SIGINT, SIGTERM) for clean shutdown
- [~] Timeout for command execution (prevent hanging commands)
- [ ] Proper error types instead of raw strings
- [ ] Context propagation (context.Context) for cancellation
- [ ] Goroutine leak prevention in streaming mode

### Measure

- [ ] Fuzz testing on all filters (go test -fuzz)
- [ ] Security scan with gosec
- [ ] Memory profiling under large inputs (10MB+, 100MB+)

### Adjust

- [ ] Configurable command timeout (default: 30s)
- [ ] Configurable max output size (default: 10MB)
- [ ] Configurable MCP allowed commands

### Deliver

- [ ] Security documentation (docs/security.md)
- [ ] Tag v0.4.0

---

## Sprint 5 — Intelligence

### Build

- [ ] Structured summary for very large outputs (>1000 lines): `[summary: 3 errors, 47 warnings, 2341 ok]`
- [ ] Cache: skip re-filtering identical output (content-hash based)
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
