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
- [x] Tag v0.2.0

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

- [x] Tag v0.3.0

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
- [x] Tag v0.4.0

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

## Sprint 6 — Polish + Release `DONE`

### Build

- [x] Project-specific pattern learning
- [x] Full documentation update

### Deliver

- [x] Tag v0.2.0, v0.3.0, v0.4.0, v0.5.0, v0.6.0 (catch-up)
- [ ] Tag v1.0.0 (after final review)
- [x] Project landing page (Astro site in `site/`, i18n EN/FR, GitHub Pages deploy)

---

## Sprint 8 — Extended Command Support + Filter Quality

> Improve filtering efficiency by adding detectors for common commands and fixing quality gaps in existing compiler filters.

### Build — New command detectors

- [x] `curl`/`wget` filter (strip progress bars, compress headers, keep response body + status)
- [x] `jq` filter (detect JSON output, compact verbose formatting)
- [x] `tar`/`zip`/`unzip` filter (compress file listing, strip verbose metadata)
- [x] `python`/`python3` filter (traceback compression, pip noise removal)
- [x] `node` filter (experimental/deprecation warnings, webpack/vite noise, internal frames)
- [x] `tree` filter (compress deep nesting, chain compression)
- [x] `terraform` filter (compress plan output, strip refresh lines, keep changes)
- [x] `kubectl`/`helm` filter (compress status output, strip managed fields)
- [x] `ssh`/`scp` filter (strip remote ANSI, compress connection banners)

### Build — Improve existing filters

- [x] Go build: preserve package headers and error context
- [x] Rust/cargo: preserve `note:` and `help:` lines (classified as Warning via classifier)
- [x] npm audit: keep all critical/high severity details (only truncate low/moderate)
- [x] Docker build: annotate FROM image names in multi-stage builds as stage boundaries
- [x] Python traceback: preserve import chain in ImportError/ModuleNotFoundError
- [x] JSON/YAML parse errors: classify as error-level

### Build — Auto-detection improvements

- [x] Increase auto-detect sample from 20 to 50 lines (catch late-appearing patterns)
- [x] Add detection patterns for: python traceback, terraform plan, kubectl output, curl verbose
- [x] Improve cross-command detection (weighted scoring, 20% fallback for single-candidate)

### Measure

- [x] Benchmark new detectors with realistic fixtures (7 new: curl, python, terraform, kubectl, tar, ssh, node)
- [x] Quality score: verify no regression on existing filters (all tests pass)
- [x] Per-filter contribution analysis for new filters (available via `gotk bench --per-filter`)

### Deliver

- [x] Unit tests for each new detector
- [x] Golden-file tests for each new detector (curl, python, terraform, kubectl, docker, tar, ssh)
- [x] Update filter catalog documentation
- [x] Update architecture documentation
- [x] Update man page with new supported commands
- [x] Tag v1.2.0

---

## Sprint 7 — Context Search (`gotk ctx`) `DONE`

> Integrate smart search capabilities inspired by `ai-ctx` into GoTK.
> GoTK already excels at cleaning output — this sprint adds the ability to **generate** token-optimized search results directly.

### Build — Core search engine

- [x] `gotk ctx <keyword>` — scan mode: list files + occurrence count + truncated matches (default)
- [x] `gotk ctx <keyword> -d [N]` — detail mode: show N lines of context around each match (default: 3)
- [x] `gotk ctx <keyword> --def` — definition mode: target `func|class|struct|type|interface|const|var|trait|impl` declarations
- [x] `gotk ctx <keyword> --tree` — structural skeleton of matching files (imports, types, functions)
- [x] `gotk ctx <keyword> --summary` — occurrence distribution by directory with file counts
- [x] Filtering options: `-t <type>` (file type), `-g <glob>` (glob pattern), `-m N` (max results)
- [x] Search path option: `-p <path>` (default: `.`)
- [x] Built-in exclusions (node_modules, .git, dist, vendor, __pycache__, lock files, .venv, coverage, .next)
- [x] Fallback: `--def` with no results retries as standard keyword search

### Build — GoTK integration

- [x] Apply existing GoTK filters to search output (ANSI strip, path compression, secret redaction)
- [x] `--stats` works with `gotk ctx` (show token reduction vs raw grep output)
- [x] MCP tool `gotk_ctx` — expose context search to LLM agents via MCP protocol
- [x] Respect `.gotk.toml` config (custom exclusions, max-lines, filter mode)
- [x] Overlap merging in detail mode: adjacent matches in same file are merged (no duplicate lines)

### Build — Smart output formatting

- [x] Compact scan output: `3x src/auth/handler.go` + indented matches truncated to 120 chars
- [x] Detail output: `--- file:line ---` headers with numbered context, `>` marker on match lines
- [x] Tree output: language-aware skeleton extraction (Go, Python, JS/TS, Rust, Java, Ruby, C/C++, Shell)
- [x] Summary output: directory breakdown table sorted by match count
- [x] All output modes apply existing truncation (head+tail) for large result sets

### Measure

- [x] Token savings benchmarks: scan -48 to -96%, def -50 to -96%, summary -89 to -98% vs raw grep
- [x] Add `gotk ctx` fixtures to `gotk bench` suite (ctx scan + ctx detail)

### Deliver

- [x] Man page section for `gotk ctx` (synopsis, modes, subcommands, examples)
- [x] `gotk help ctx` with examples
- [x] Unit tests for all components (14 tests: ParseFlags, WalkFiles, Search, all 5 formatters, merge windows)
- [x] Integration tests for all 5 modes (8 E2E tests: scan, detail, def, tree, summary, stats, no-match, help)
- [ ] Tag v1.1.0

---

## Sprint 9 — Claude Code Hook Integration

> Native Claude Code hooks integration via PreToolUse event.
> Auto-install command for zero-config setup.

### Build — Hook protocol handler

- [x] `internal/hook/` package — Parse Claude Code hook JSON, wrap Bash commands with `| gotk`
- [x] `gotk hook` subcommand — Called by Claude Code PreToolUse event
- [x] Smart command wrapping: skip trivial commands (cd, pwd, echo, etc.)
- [x] Double-wrap prevention: detect existing `| gotk` in command
- [x] Self-invocation prevention: don't wrap `gotk` commands
- [x] Exit code preservation via `set -o pipefail`
- [x] Env var prefix handling (`LANG=C sort` correctly wrapped)

### Build — Install command

- [x] `gotk install claude` — Auto-configure PreToolUse hook in settings.json
- [x] `--global` flag: install in `~/.claude/settings.json`
- [x] `--project` flag: install in `.claude/settings.json` (default)
- [x] `--uninstall` flag: remove GoTK hook from settings
- [x] `--status` flag: check installation status
- [x] Preserves existing settings.json content (permissions, other hooks)
- [x] Idempotent: detects if hook already installed

### Build — Daemon mode

- [x] `gotk daemon` subcommand — spawn filtered shell session
- [x] Zsh integration via `accept-line` ZLE widget override
- [x] Bash integration via `shopt -s extdebug` + DEBUG trap
- [x] Interactive command detection (vim, ssh, less, tmux, fzf, etc.)
- [x] Trivial command skip (reuse `hook.TrivialCommands`)
- [x] Self-invocation prevention (skip gotk commands)
- [x] Double-wrap prevention (skip already-piped commands)
- [x] Prompt modification (`[gotk]` prefix)
- [x] Nested daemon prevention (`GOTK_DAEMON=1` check)
- [x] Session summary on exit (commands filtered, tokens saved)
- [x] `gotk daemon status` — check if inside a daemon session
- [x] `gotk daemon init` — print shell init code for manual eval
- [x] `gotk daemon summary` — print session stats (called on exit)
- [x] Signal forwarding to child shell (SIGINT, SIGTERM)
- [x] Auto-enable measurement during daemon sessions

### Deliver

- [x] Unit tests for hook protocol (12 tests: JSON parse, wrapping, skip logic)
- [x] Unit tests for install (7 tests: create, merge, uninstall, idempotency)
- [x] Unit tests for daemon (14 tests: skip logic, script generation, init files)
- [x] Updated `examples/claude-code-hook.sh` to new PreToolUse format
- [x] Help text for `gotk help install`, `gotk help hook`, `gotk help daemon`
- [ ] Tag v1.3.0

---

## Sprint 10 — Security Hardening + Quality (from audit 2026-03-29)

> Full security audit + quality audit identified 15 security findings and 30 quality improvements.
> This sprint addresses critical/high security issues and P1 quality issues.

### Build — Security fixes (Critical + High)

- [x] Fix temp file permissions: `0600` instead of `0644` in `daemon/daemon.go`
- [x] MCP `gotk_read`: validate path is under project root, block traversal (`../../etc/passwd`)
- [x] MCP `gotk_grep`: same path validation as `gotk_read`
- [x] MCP `gotk_ctx`: skip symlinks in `ctx/walk.go` to prevent symlink-based exfiltration
- [x] Validate `GOTK_BIN` path in daemon shell scripts (check absolute path, file exists)
- [x] Audit log: validate path is not a symlink, parent dir not world-writable

### Build — Security fixes (Medium)

- [x] Secret redaction: add patterns for short JWTs, `bearer` tokens, OPENSSH keys
- [x] Logger rotation: use atomic file ops (temp file + rename) instead of read-truncate-write
- [x] Learn store rotation: same atomic file ops fix
- [x] Settings file permissions: `0600` instead of `0644` in `install/claude.go`

### Build — Security fixes (Low)

- [x] MCP denylist: add `curl|bash`, `wget|sh`, `truncate`, `iptables`, `ufw`
- [x] ReDoS: limit repetition in private key regex pattern (`[A-Z0-9 ]{0,20}`)
- [x] Rate limiting: log violations to audit log

### Build — Quality fixes (P1)

- [x] Fix hardcoded version `v0.1.0` → use `-ldflags "-X main.Version=$(git describe --tags)"`
- [x] Add `go vet` + `golangci-lint` + `gofmt` check to CI pipeline
- [x] Add test coverage reporting to CI (fail if < 70%)
- [x] Increase daemon test coverage (33% → 70%+): test shell scripts, edge cases
- [x] Increase install test coverage (44% → 70%+): JSON merge edge cases, error paths
- [x] Update BACKLOG.md tag status to match actual git tags

### Build — Quality fixes (P2)

- [x] Extract `cmd/gotk/main.go` (1244 lines) into separate files: subcommands.go, usage.go
- [x] Unify command classification: move `TrivialCommands` + `InteractiveCommands` to shared package
- [x] Replace custom `itoa()` in `detect/detect.go` with `strconv.Itoa()`
- [x] Fix swallowed errors in install/measure paths — return errors instead of logging warnings
- [x] Add `Filter` interface with `Name()` method for chain introspection
- [x] Command registry pattern: unify `Identify()` + `FiltersFor()` into single registry
- [x] Pre-compile regex patterns in `filter/rules.go` (already optimal — compiled once per BuildChain)
- [x] Add `gotk config show` subcommand (show loaded config files + effective config)
- [x] Add `--quiet` and `--debug` flags
- [x] Improve error messages: include recovery steps

### Build — CI/CD (P2)

- [x] Add release automation with goreleaser (multi-platform binaries on tag push)
- [x] Add benchmark regression detection (compare results to baseline)
- [x] Build with `-ldflags="-s -w"` to reduce binary size (4.7MB → 3.3MB)

### Build — Documentation (P2)

- [x] Update CLAUDE.md to reflect current architecture (18 filters, hook, daemon, install)
- [x] Document config merging rules (3-level precedence, nested map behavior)
- [x] Add godoc comments to all public functions in ctx/, measure/, daemon/ (already present)
- [x] Add architecture.md sections: config merging, filter extensibility, performance characteristics
- [x] Document `GOTK_PASSTHROUGH=1` in README and help text

### Build — Testing (P2)

- [x] Add E2E tests: complex filter interactions, large output, signal handling
- [x] Add golden file edge cases: filter combinations, empty lines, binary data, rule conflicts
- [x] Add config precedence integration tests (global/project/local merge)
- [x] Increase MCP test coverage (65% → 83.8%): rate limiting, sandbox edge cases

### Deliver

- [x] All security fixes applied and tested
- [x] CI green with linting + coverage gates
- [ ] Tag v1.3.0

---

## Sprint 11 — Live-Install Feedback (2026-04-17 → 2026-04-18)

> Three issues raised after a live install on a real Node.js project with a 1620-test Jest suite. Each one addressed a concrete failure mode not covered by synthetic fixtures.

### Build

- [x] Fix false-positive `FAIL` on Jest `console.log` trailers (#18, PR #21) — isolated stack frames and trailers below `console.<method>` headers no longer inflate the error count; authoritative test-runner result markers (Jest / pytest / go test / cargo) override heuristic anchor matching.
- [x] Auto-escalate truncation to preserve failure context (#20, PR #22) — new `--auto-escalate` flag with `off|hint|window|conservative` modes; `window` (default) keeps head + tail + ±N lines around every detected failure anchor so a mid-run failure never hides in the dropped middle.
- [x] MCP coverage for `Read`/`Grep`/`Glob`-style operations (#19, PR #23) — new `gotk_glob` tool with directory clustering at ≥30 matches; `gotk_grep` now caps matches per file with a `… N more in this file` marker.

### Deliver

- [x] Unit tests for each fix (3 + 10 + 10)
- [x] `docs/quickstart.md` updated with `--auto-escalate` examples
- [x] `gotk --help` and MCP help text updated with new tools and flags
- [x] Tag v1.4.0 (after final review)

---

## Backlog (Unprioritized)

- [x] `--aggressive` / `--balanced` / `--conservative` filter modes
- [x] Per-command truncation threshold tuning
- [x] Whitelist/blacklist patterns to always keep/remove
- [x] Per-LLM profiles (Claude, GPT, Gemini)
- [x] Rate limiting in MCP server
- [x] CI pipeline with automated benchmarks
- [x] `gotk update` — self-upgrade command (shipped in v1.4.0). Hybrid: GitHub Releases self-replace with `go install @latest` fallback. `--check` for check-only, `--force`, `--from-source`.
- [ ] CI maintenance: bump GitHub Actions off Node.js 20 (deprecated — forced to Node.js 24 on 2026-06-02, removed 2026-09-16). Affects `actions/checkout@v4`, `actions/setup-go@v5`, `goreleaser/goreleaser-action@v6` in `release.yml` and `ci.yml`. Check for newer majors or set `FORCE_JAVASCRIPT_ACTIONS_TO_NODE24=true` as a short-term opt-in.
- [ ] CI cosmetic: `actions/setup-go` cache warns "Dependencies file is not found... go.sum" on the release job because the project has zero external deps. Either add `cache: false` to the step or suppress by creating an empty `go.sum` at checkout. Non-blocking — just noise in the run log.

---

## Icebox (Ideas to Explore)

- [x] Daemon mode to intercept all shell commands (Sprint 9)
- [ ] Windows support (PowerShell output patterns)
- [ ] HTTP API for remote integration
- [ ] VSCode plugin to visualize raw vs cleaned
- [ ] Prometheus/OpenTelemetry metrics (tokens saved over time)
- [ ] Multi-language detection support (localized error messages)
- [ ] Semantic compression via local lightweight LLM (summarize before sending to main model)
- [ ] Homebrew / AUR / scoop package publishing
