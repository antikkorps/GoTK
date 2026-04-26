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

## Sprint 12 — Jest/Node Live-Install Feedback v2 (2026-04-20)

> Second wave of real-world feedback from a large Jest suite (90+ files, 6 workers, 1620 tests).
> All three issues come from the same run — residual noise and output ordering defects observed with `gotk -s npm test 2>&1`.

### Build

- [x] #27 — Stats line appears mid-output when stdout/stderr are merged. Batch-mode now writes cleaned stdout, then passthrough stderr, then the `[gotk]` marker. Under `2>&1` the marker always lands at the very end. Streaming mode already had the right order (stats after `wait()`).
- [x] #28 — Jest `console.log` + `at <path>:<line>:<col>` annotations. New `stripJestConsoleBlocks` filter in `internal/detect/filters_jest.go`: matches a lone `console.<method>` header, an indented message, and a strict trailer `^\s+at [^\s()]+:\d+:\d+\s*$` (no parens, so real stack frames like `at fn (path:N:N)` pass through untouched). Wired into `CmdNpm` and `CmdNode`.
- [x] #32 — Collapse repeated `(node:PID) Warning:` blocks from multi-worker runs. Extended `compressNodeOutput` with a `nodeGenericWarn` pattern: first occurrence keeps the original line + the `(Use \`node --trace-warnings …\`)` hint, subsequent workers collapse into `… and N identical warnings from other workers`.

### Measure

- [x] Golden fixture (4-worker Jest output) shows -70% reduction on a representative sample.
- [x] `gotk bench` average reduction: 84.78% (baseline ≥ 50% CI gate).
- [x] No regression: all existing golden tests pass.

### Deliver

- [x] Golden-file test for Jest output (`testdata/golden/jest/npm_test.input|expected`).
- [x] Unit tests: 9 new tests (7 for `stripJestConsoleBlocks` covering single/multi-block, multi-line message, real-stack-trace preservation, no-trailer, non-indented break, all 5 console methods; 2 for generic-warning collapse).
- [x] E2E tests: `TestE2E_StatsLandsAtEndOfMergedStream`, `TestE2E_JestConsoleBlocksStripped`, `TestE2E_MultiWorkerNodeWarningsCollapsed`.
- [ ] Tag v1.5.0.

---

## Sprint 13 — Install/Uninstall UX (2026-04-20)

> Symmetric install/uninstall surface. Two distinct uninstall flows: remove the gotk binary entirely, or just detach one integration (e.g. Claude) when the user switches LLM.

### Build

- [x] `gotk uninstall claude` — symmetric alias for `gotk install claude --uninstall`. Accepts the same `--local` / `--project` / `--global` scope flags. The old `--uninstall` flag still works.
- [x] #29 — `gotk uninstall` (no target): plans a full cleanup, prints which Claude hook scopes have a gotk hook + which config files exist, prompts for `[y/N]`. On confirm removes every present hook, deletes the config files, and removes any now-empty GoTK config/data directories. Refuses to self-delete the binary while running — prints the exact `rm` command (with `sudo` hint when the parent dir isn't writable). `--dry-run` and `--yes` / `-y` flags.
- [ ] #31 — Generalize `gotk install <agent>` for multiple agents. Deferred until a concrete second agent is requested — no point abstracting ahead of demand.

### Deliver

- [x] Unit tests: 5 tests covering plan detection, selective hook removal, config file/dir cleanup, non-empty-dir preservation, and the `Confirm` prompt (8 subcases).
- [x] E2E tests: `TestE2E_UninstallClaudeSymmetric`, `TestE2E_UninstallDryRun`, `TestE2E_UninstallYesRemovesConfig`.
- [x] Updated `docs/quickstart.md` with the new `gotk uninstall` flows.
- [x] Help text via `gotk help uninstall`.

---

## Sprint 14 — Windows Support (2026-04-20)

> Moves #30 out of the Icebox. Queued after the CI Node 20 bump per the current roadmap.

### Build

- [ ] Cross-compile check: `GOOS=windows GOARCH=amd64 go build` passes and binary runs under Windows 10/11 (CMD, PowerShell, Git Bash).
- [ ] Audit `os/exec`, path handling, and temp-file code for Windows-specific quirks (backslashes, `\\?\` long paths, `CRLF` line endings in filters).
- [ ] ANSI rendering: confirm stripping still works and consider enabling VT processing when output goes to a Windows console.
- [ ] Daemon mode: decide support matrix. Zsh/bash interception doesn't apply — evaluate a PowerShell profile hook or mark daemon unsupported on Windows for this release.
- [ ] `gotk update` self-replace: Windows can't rename a running executable. Implement the pending-replace pattern (write `.new`, spawn detached helper that swaps on exit).
- [ ] `gotk install claude`: adjust `~/.claude/settings.json` path resolution for Windows user profile.

### Measure

- [ ] Run the full golden-file test suite on a Windows runner in CI.
- [ ] Verify `gotk bench` numbers are within 5% of Linux/macOS on the same corpus.

### Deliver

- [ ] Goreleaser config: add `windows/amd64` and `windows/arm64` targets, produce `.zip` archives.
- [ ] README + `docs/quickstart.md`: Windows install and shell integration instructions.
- [ ] Document the support matrix (what works, what is deliberately out of scope — e.g. daemon mode if deferred).

---

## Sprint 15 — Filter Tech Debt (from 2026-04-24 assessment)

> Three quality bugs shipped this week (#37 / #39 / #40 → v1.5.2) surfaced recurring structural issues. None are urgent, but leaving them to rot makes future filter work slower. Take this slice before (or alongside) Sprint 14 Windows so the Windows port lands on a cleaner core.

### Build — Stderr parity pass

- [ ] Catalogue every filter and decide its stderr policy (apply / skip / apply a narrow subset). Bugs rooted in "stdout-only filtering" so far: #37 (node warnings), #39 (jest totals that live on stderr), partially #40. Likely next: `RedactSecrets` on stderr — a secret written to stderr today leaks unredacted.
- [ ] Introduce a single place that decides how stderr is handled instead of the current ad-hoc patches (`SummarizeWithContext(...stderr)`, `detectRunnerResult(stderrLines)`, `CollapseNodeWarnings` called directly from `cmd/gotk/main.go`). Candidate: a `StderrPolicy` on each filter, or a second mini-chain applied to stderr before pass-through.
- [ ] Decide the MCP path explicitly: `internal/mcp/server.go` concatenates stdout+stderr into `raw` before filtering, which accidentally covers some of these cases. Document this or align both paths.

### Build — Consolidate duplicate implementations

- [ ] Node worker-warning collapse: two implementations now coexist — `CollapseNodeWarnings` in `internal/filter/nodewarn.go` (generic, runs in the main chain + on stderr) and the `genericWarnCount` branch inside `compressNodeOutput` in `internal/detect/filters_node.go` (runs only on `CmdNpm`/`CmdNode`). Markers are aligned, but the duplication will drift. Pick `CollapseNodeWarnings` as canonical, remove the `compressNodeOutput` branch, and migrate its unit tests.
- [ ] Test-runner summary anchors: `detectRunnerResult` in `internal/filter/summary.go` and `summaryAnchors` in `internal/filter/escalate.go` use near-identical regex sets (jest / vitest / pytest / cargo / go test). Extract a shared `runnerAnchors` package / file and have both call sites consume it.
- [ ] Audit the rest of `internal/detect/filters_*.go` for similar overlap with the generic chain in `internal/filter/`.

### Build — Detection robustness

- [ ] Detection currently keys off `parts[0]` via `filepath.Base` — wrappers (`pnpm exec jest`, `npx vitest`, scripts that `exec` node) slip through as `CmdGeneric`. Look into a lightweight auto-detect from output signature (already partly done in `detect.AutoDetect` for pipe mode) and consider running it when the CLI path yields `CmdGeneric`.
- [ ] Expose detection in `--debug` output so users can see why a filter didn't fire (already partially there via `logDebug`).

### Build — Package review

- [ ] Check whether `internal/learn/`, `internal/classify/`, `internal/cache/`, `internal/cmdclass/` are earning their keep. Evidence-driven: query `gotk measure` data for measurable wins (token savings, cache hit rate, classifier agreement). Merge or drop what doesn't clear the bar. Goal is scope discipline, not a rewrite — be conservative.

### Measure

- [ ] Before touching stderr filtering, add a benchmark fixture that puts test-runner totals on stderr (mirrors the real jest shape). Gate any refactor on: (a) existing golden tests still pass, (b) new stderr fixture produces the expected summary verdict.
- [ ] Track reduction ratio on that fixture before/after consolidation — must not regress.

### Deliver

- [ ] Tag `v1.5.3` after the stderr pass + Node warning consolidation (pure cleanup, no user-visible API change). Summary anchors unification can land in the same tag or the next.
- [ ] Document the stderr policy in `docs/architecture.md`.

---

## Backlog (Unprioritized)

- [x] `--aggressive` / `--balanced` / `--conservative` filter modes
- [x] Per-command truncation threshold tuning
- [x] Whitelist/blacklist patterns to always keep/remove
- [x] Per-LLM profiles (Claude, GPT, Gemini)
- [x] Rate limiting in MCP server
- [x] CI pipeline with automated benchmarks
- [x] `gotk update` — self-upgrade command (shipped in v1.4.0). Hybrid: GitHub Releases self-replace with `go install @latest` fallback. `--check` for check-only, `--force`, `--from-source`.
- [x] CI maintenance: bump GitHub Actions off Node.js 20. `actions/checkout@v4→v6`, `actions/setup-go@v5→v6`, `goreleaser/goreleaser-action@v6→v7` — all three now on `node24`. Applied to both `ci.yml` and `release.yml`.
- [x] CI cosmetic: `actions/setup-go` cache warns "Dependencies file is not found... go.sum". Fixed by adding `cache: false` on every `setup-go` step (zero external deps — no go.sum to cache).
- [ ] Interactive install wizard — `gotk setup` (or `gotk init`): detect shell, detect which LLM CLIs are on PATH (Claude Code, Cursor, Aider, Continue.dev), ask the user which integrations to enable and at which scope (local/project/global), run the corresponding installs, then write a minimal `.gotk.toml` if the user wants project-specific config. Benefit: single entry-point for first-time users instead of three or four docs pages. Non-goals: no TUI framework (keep it stdin/stdout scanner-based to match `gotk uninstall`'s prompt style). Depends on #31 (multi-agent install generalization) for anything beyond Claude.

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
