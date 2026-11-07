# GoTK - LLM Output Proxy

## Project

CLI tool (Go) that acts as a proxy to clean command output before sending it
to LLMs. Goal: ~80% token reduction by stripping noise, without losing signal.

Personal Go learning project, inspired by RTK (a similar tool written in Rust).
Intentionally a lighter take — not a 1:1 port — scoped to the features actually
needed while exploring Go.

See @README.md for usage and @docs/architecture.md for the internals
(filter chain, command registry, stderr policy).

**All code, comments, documentation, backlog, and commit messages MUST be
written in English.**

## Commands

```bash
go build -ldflags "-s -w -X main.Version=$(git describe --tags --always)" -o gotk ./cmd/gotk/
go test ./...                        # unit + e2e (e2e_test.go at repo root)
go test -race ./...                  # what CI runs (plus a 70% coverage gate)
gofmt -l .                           # must print nothing
go vet ./...
golangci-lint run ./...
./gotk bench --quality               # filter quality score (CI gate: >= 40%)
./gotk bench --per-filter            # per-filter compression report

./gotk --stats grep -rn "pattern" .  # try a filter on a real command
echo "test" | ./gotk --stats         # pipe mode
./gotk --debug echo hello            # show config / detection / chain
./gotk config show                   # effective config

cd site && npm run dev               # landing page (Astro), deployed by pages.yml
```

## Before pushing

```bash
test -z "$(gofmt -l .)" && go vet ./... && golangci-lint run ./... && go test -race ./...
```

Coverage (>= 70%) and bench thresholds (quality >= 40%, avg reduction >= 50%,
Linux/Windows parity) are enforced by CI only.

## Workflow

- Plan first, then TDD: write the failing test and show the RED output before
  implementing; re-run and show the GREEN output after.
- One function, one responsibility. Functions under 30 lines, files under ~200
  lines; split by responsibility, never arbitrarily.
- DRY with judgement: factor out on the third repetition; no abstraction with a
  single caller. Flag DRY violations.
- Use current best practices for the versions already installed. No deprecated
  APIs. Never add a dependency without saying so and why.
- Comments in English, explaining why, not what.
- At the end of each feature, delegate a code review to a subagent that
  receives only the diff and the spec.
- Before every push: run the "Before pushing" command from the repo's
  CLAUDE.md; do not push if it fails; never use `--no-verify`; paste the test
  output in the summary.

## Code structure

- `cmd/gotk/` — CLI entrypoint (`main.go`: flags, pipe detection, output
  emission), subcommand handlers (`subcommands.go`), help text (`usage.go`).
- `internal/filter/` — Filter chain and generic filters (ANSI, whitespace,
  dedup, paths, trim, truncate, secrets, stack traces, summary).
- `internal/detect/` — Command registry (CmdType -> {name, binaries, filters})
  and command-specific filters (`filters_<tool>.go`).
- `internal/exec/` — Command execution and stdout/stderr capture.
- `internal/proxy/` — Shell mode and chain builders (incl. `BuildStderrChain`).
- `internal/` other packages — one concern each (config, ctx, hook, install,
  mcp, daemon, dashboard, bench, learn, measure, classify, cache, watch...).
  List them with `ls internal/` rather than trusting a stale list here.
- `testdata/golden/` — before/after golden files for filter tests.
- `site/` — Astro landing page, independent npm project.
- `BACKLOG.md` — sprint backlog; update it when closing a sprint.

## Project-specific rules

### Design principles

- Filters are composable `func(string) string` chained via `filter.Chain`;
  use `AddNamed` so `Names()`/`Len()` introspection and `bench --per-filter`
  keep working.
- Command detection goes through the single registry map — never add a
  parallel lookup.
- Generic filters always apply; command-specific filters add on top.
- Stderr gets only the narrow stderr chain (see Stderr Policy in
  `docs/architecture.md`); any exception must be an allowlist of noise
  patterns, never a generic chain.
- `--quiet` suppresses info/warning messages; `--debug` shows
  config/detection/chain details.

### Search before creating

Before writing a new filter, helper, or utility, grep the codebase for
existing implementations with similar names or purposes. Prefer extending an
existing one over creating a parallel implementation. Past pain:
`compressNodeOutput` already existed when `CollapseNodeWarnings` was added —
the duplication had to be reconciled later.

### Quality-first filtering

**Output quality is the #1 priority — token reduction is secondary.** GoTK
must NEVER degrade LLM responses by removing semantically important
information:

1. **Never remove error messages, warnings, or diagnostic info.**
2. **Never remove context needed to understand a problem** — file paths tied
   to errors, line numbers in stack traces, variable values in assertions.
3. **Preserve structural meaning** — indentation, tree hierarchy, test
   grouping.
4. **Only remove pure noise** — ANSI codes, decorative separators, duplicate
   lines, redundant path prefixes, excessive blank lines.
5. **When in doubt, keep it** — keeping noise costs far less than dropping
   signal.
6. **Truncation must keep head AND tail** — errors and summaries often appear
   at the end.
7. **Every filter needs golden-file tests** (before/after) to prevent quality
   regressions.

### Git and GitHub

- Never push to `main`: one feature branch + one PR per change.
- Close the related issue from the PR body (`Closes #N`).
- No `Co-Authored-By` trailer in commits, no "Generated with Claude Code"
  footer in PRs.
- No package publishing (Homebrew, npm, etc.) for now — too early.
