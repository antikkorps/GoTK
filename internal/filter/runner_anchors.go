package filter

import (
	"regexp"
	"strings"
)

// runnerAnchor pairs a regex matching a test-runner summary line with an
// optional verdict extractor. Anchors with a non-nil verdict can resolve a
// PASS/FAIL outcome from the matched line; anchors without a verdict only
// participate in must-keep tracking during truncation.
//
// This single source of truth backs both:
//   - findSummaryAnchors (escalate.go) — which lines must survive truncation.
//   - detectRunnerResult (summary.go) — authoritative PASS/FAIL verdict.
type runnerAnchor struct {
	re      *regexp.Regexp
	verdict func(line string) (verdict string, ok bool)
}

// runnerAnchors are the test-runner total / result / duration lines for the
// supported runners (Jest, Vitest, pytest, go test, cargo). Order matters in
// detectRunnerResult: lines are scanned bottom-up, and on each line the
// anchors are tried in order — the first matching anchor with a verdict wins.
var runnerAnchors = []runnerAnchor{
	// Jest: "Test Suites: 4 passed, 4 total" — kept for truncation only.
	{re: regexp.MustCompile(`^\s*Test Suites:\s`)},
	// Jest: "Tests:       5 failed, 1615 passed, 1620 total"
	{re: regexp.MustCompile(`^\s*Tests:\s+.*\btotal\b`), verdict: failedCountVerdict},
	{re: regexp.MustCompile(`^\s*Snapshots:\s`)},
	{re: regexp.MustCompile(`^\s*Time:\s+\d`)},
	// Vitest: "Test Files  2 failed | 19 passed (21)" / "Tests  2 failed | 282 passed (284)"
	{re: regexp.MustCompile(`^\s*Test Files\s+.*\b(passed|failed)\b`), verdict: failedCountVerdict},
	{re: regexp.MustCompile(`^\s*Tests\s+.*\b(passed|failed)\b`), verdict: failedCountVerdict},
	{re: regexp.MustCompile(`^\s*Duration\s+\d`)},
	// pytest: "======= 42 passed in 1.23s =======" / "==== 3 failed, 40 passed in 2s ===="
	{re: regexp.MustCompile(`^=+.*\b(passed|failed|error)\b.*=+$`), verdict: pytestVerdict},
	// Cargo: "test result: ok. 42 passed; 0 failed; ..."
	{re: regexp.MustCompile(`^\s*test result:\s+(ok|FAILED)\.\s+\d+\s+passed`), verdict: cargoVerdict},
	// Go test FAIL: emitted as "FAIL\tpkg/path\t<duration>" or just "FAIL\tpkg/path".
	// The bare-FAIL form (no duration) appears when a package fails to build, so
	// we accept either trailing whitespace or a duration. Always FAIL.
	{re: regexp.MustCompile(`^FAIL\s+\S+\s`), verdict: constVerdict("FAIL")},
	// Go test OK: "ok  \tpkg/path\t0.123s" — PASS, but a later FAIL would override
	// (caller scans bottom-up and a FAIL hit will return earlier).
	{re: regexp.MustCompile(`^ok\s+\S+\s+[\d.]+s`), verdict: constVerdict("PASS")},
}

// failedCountVerdict resolves PASS/FAIL from a runner summary line by reading
// the "N failed" / "N failures" / "N errors" counter shared by Jest, Vitest,
// and pytest. Missing counter or zero count → PASS; any non-zero → FAIL.
func failedCountVerdict(line string) (string, bool) {
	if m := failedCount.FindStringSubmatch(line); m != nil && m[1] != "0" {
		return "FAIL", true
	}
	return "PASS", true
}

// pytestVerdict resolves PASS/FAIL from the pytest summary banner. The banner
// can carry "failed" or "error" keywords without a leading count when the
// runner aborts, so we treat any "failed" presence as FAIL.
func pytestVerdict(line string) (string, bool) {
	if failedCount.MatchString(line) {
		return "FAIL", true
	}
	if strings.Contains(line, "passed") {
		return "PASS", true
	}
	return "", false
}

// cargoFullResultLine captures both the result word and the failed counter
// when both are present on a cargo summary line.
var cargoFullResultLine = regexp.MustCompile(`^\s*test result:\s+(ok|FAILED)\.\s+\d+\s+passed;\s+(\d+)\s+failed`)

// cargoVerdict resolves PASS/FAIL from a cargo test summary. PASS only when
// the result word is "ok" and zero failures are reported.
func cargoVerdict(line string) (string, bool) {
	if m := cargoFullResultLine.FindStringSubmatch(line); m != nil {
		if m[1] == "ok" && m[2] == "0" {
			return "PASS", true
		}
		return "FAIL", true
	}
	// Shorter "test result: ok." form (no failed count): infer from the word.
	if strings.Contains(line, "test result: ok") {
		return "PASS", true
	}
	return "FAIL", true
}

// constVerdict returns a verdict func that always reports the given outcome
// when the anchor matches. Used for go test ok/FAIL lines.
func constVerdict(v string) func(string) (string, bool) {
	return func(string) (string, bool) { return v, true }
}
