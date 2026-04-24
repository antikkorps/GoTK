package filter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/antikkorps/GoTK/internal/classify"
)

// SummaryThreshold is the minimum number of lines before a summary is generated.
const SummaryThreshold = 100

// filePathPattern matches file paths like path/to/file.ext: or standalone paths.
var filePathPattern = regexp.MustCompile(`(?:^|\s)((?:[a-zA-Z]:)?(?:[a-zA-Z0-9_./-]+/)+[a-zA-Z0-9_.-]+\.[a-zA-Z0-9]+)(?::\d+)?`)

// resultFailPattern detects failure signals in output.
var resultFailPattern = regexp.MustCompile(`(?i)\b(FAIL|FAILED|(?:^|\s)ERROR(?:\s|$)|panic)\b`)

// resultPassPattern detects success signals in output.
var resultPassPattern = regexp.MustCompile(`(?i)\b(PASS|(?:^|\s)ok(?:\s|$)|SUCCESS|0 errors)\b`)

// Test-runner summary anchors (authoritative when present, see detectRunnerResult).
var (
	// Jest: "Tests:       5 failed, 1615 passed, 1620 total"
	jestTotalsLine = regexp.MustCompile(`^\s*Tests:\s+.*\btotal\b`)
	// Cargo: "test result: ok. 42 passed; 0 failed; ..."
	cargoResultLine = regexp.MustCompile(`^\s*test result:\s+(ok|FAILED)\.\s+\d+\s+passed;\s+(\d+)\s+failed`)
	// Go test: "ok  \tpkg/path\t0.123s" (pass), "FAIL\tpkg/path" (fail at EOL of test run)
	goTestOkLine   = regexp.MustCompile(`^ok\s+\S+\s+[\d.]+s`)
	goTestFailLine = regexp.MustCompile(`^FAIL\s+\S+\s`)
	// pytest: "======= 42 passed in 1.23s =======" or "==== 3 failed, 40 passed in 2s ===="
	pytestSummaryLine = regexp.MustCompile(`^=+.*\b(passed|failed|error)\b.*=+$`)
	// Shared counters used to extract numbers from Jest / pytest summary lines.
	failedCount = regexp.MustCompile(`(\d+)\s+(?:failed|failures|errors)`)

	// Jest's "console.<method>" block header (appears above console.log output).
	jestConsoleHeader = regexp.MustCompile(`^\s*console\.(log|warn|error|info|debug|trace)\s*$`)
	// Generic single stack-frame line (node / jest style).
	stackFrameLine = regexp.MustCompile(`^\s+at\s+`)
	// Error / exception banner preceding a real stack trace.
	errorBannerLine = regexp.MustCompile(`(?i)^(\s*[A-Z]\w*(Error|Exception):|Error:|●\s)`)
)

// formatNumber formats an integer with comma separators for readability.
func formatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var result []byte
	remainder := len(s) % 3
	if remainder > 0 {
		result = append(result, s[:remainder]...)
	}
	for i := remainder; i < len(s); i += 3 {
		if len(result) > 0 {
			result = append(result, ',')
		}
		result = append(result, s[i:i+3]...)
	}
	return string(result)
}

// resolveResult picks PASS/FAIL/unknown from the strongest signal available.
// Priority (most → least authoritative):
//  1. Runner summary on stdout (Jest/pytest/go test/cargo test totals).
//  2. Runner summary on stderr (Jest writes its reporter to stderr).
//  3. Exit code 0 → PASS. Per issue #39, a clean exit means the command
//     succeeded; any "error"-looking content is typically console output from
//     inside tests that exercise error paths.
//  4. Exit code > 0 → FAIL (the process itself reported failure).
//  5. Exit code unknown (-1): fall back to heuristic FAIL/PASS anchors.
func resolveResult(lines []string, exitCode int, stderr string, hasFail, hasPass bool) string {
	if runnerResult, ok := detectRunnerResult(lines); ok {
		return runnerResult
	}
	if stderr != "" {
		stderrLines := strings.Split(stderr, "\n")
		if runnerResult, ok := detectRunnerResult(stderrLines); ok {
			return runnerResult
		}
	}
	if exitCode == 0 {
		return "PASS"
	}
	if exitCode > 0 {
		return "FAIL"
	}
	if hasFail {
		return "FAIL"
	}
	if hasPass {
		return "PASS"
	}
	return "unknown"
}

// detectRunnerResult scans for an authoritative test-runner result marker.
// Returns ("PASS"|"FAIL", true) when one is found; ("", false) otherwise.
// When present, these markers override inferred pass/fail signals from
// generic anchor-word matching, since they are emitted by the runner itself.
func detectRunnerResult(lines []string) (string, bool) {
	result := ""
	found := false

	// Scan from the end backwards — runner totals are emitted at the bottom.
	for i := len(lines) - 1; i >= 0; i-- {
		line := lines[i]

		if jestTotalsLine.MatchString(line) {
			if m := failedCount.FindStringSubmatch(line); m != nil && m[1] != "0" {
				return "FAIL", true
			}
			return "PASS", true
		}
		if m := cargoResultLine.FindStringSubmatch(line); m != nil {
			if m[1] == "ok" && m[2] == "0" {
				return "PASS", true
			}
			return "FAIL", true
		}
		if pytestSummaryLine.MatchString(line) {
			if failedCount.MatchString(line) {
				return "FAIL", true
			}
			if strings.Contains(line, "passed") {
				return "PASS", true
			}
		}
		if goTestFailLine.MatchString(line) {
			return "FAIL", true
		}
		if goTestOkLine.MatchString(line) && !found {
			// Keep scanning — a later FAIL would override.
			result = "PASS"
			found = true
		}
	}

	return result, found
}

// isJestConsoleTrailer reports whether line i is a stack-frame line that
// belongs to a Jest "console.<method>" log block rather than a real error stack.
// Jest prints one isolated `at <loc>` trailer below each console.log body, which
// syntactically looks like a stack frame but is not an error signal.
func isJestConsoleTrailer(lines []string, i int) bool {
	if !stackFrameLine.MatchString(lines[i]) {
		return false
	}
	// A real error stack has ≥2 consecutive frames — skip this check.
	if i+1 < len(lines) && stackFrameLine.MatchString(lines[i+1]) {
		return false
	}
	if i > 0 && stackFrameLine.MatchString(lines[i-1]) {
		return false
	}
	// Look back up to 6 lines for a Jest "console.<method>" header; bail out
	// if we hit an Error/Exception banner first (that would indicate a real stack).
	for j := i - 1; j >= 0 && j >= i-6; j-- {
		if errorBannerLine.MatchString(lines[j]) {
			return false
		}
		if jestConsoleHeader.MatchString(lines[j]) {
			return true
		}
	}
	return false
}

// isIsolatedStackFrame reports whether a stack-frame line appears on its own
// without any preceding error banner or adjacent frames. Such lines are
// almost always log instrumentation (console.log trailers, debug output)
// rather than real error stacks, and should not inflate the error count.
func isIsolatedStackFrame(lines []string, i int) bool {
	if !stackFrameLine.MatchString(lines[i]) {
		return false
	}
	// Part of a multi-frame stack → treat as real.
	if i+1 < len(lines) && stackFrameLine.MatchString(lines[i+1]) {
		return false
	}
	if i > 0 && stackFrameLine.MatchString(lines[i-1]) {
		return false
	}
	// Look back up to 3 lines for an error banner; if found, keep as real.
	for j := i - 1; j >= 0 && j >= i-3; j-- {
		if errorBannerLine.MatchString(lines[j]) {
			return false
		}
	}
	return true
}

// Summarize analyzes the full output and prepends a structured summary.
// Only activates when output exceeds SummaryThreshold lines.
func Summarize(input string) string {
	return summarize(input, -1, "")
}

// SummarizeWithContext returns a FilterFunc that analyzes output with the
// command's exit code and stderr as additional context. When exit code is 0,
// the summary result is forced to PASS unless a runner summary explicitly says
// otherwise — this prevents false FAIL verdicts on runs where tests pass but
// emit console.error output, or where the runner prints its totals to stderr
// (e.g. Jest). Pass -1 as exitCode to disable exit-based inference.
func SummarizeWithContext(exitCode int, stderr string) FilterFunc {
	return func(input string) string {
		return summarize(input, exitCode, stderr)
	}
}

func summarize(input string, exitCode int, stderr string) string {
	lines := strings.Split(input, "\n")

	// Remove trailing empty line from split artifact
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) < SummaryThreshold {
		return input
	}

	totalBytes := len(input)

	var errorCount, warningCount int
	var errorLines []string
	var warningLines []string

	uniqueFiles := make(map[string]struct{})
	hasFail := false
	hasPass := false

	for i, line := range lines {
		level := classify.Classify(line)

		// Downgrade Jest console-log trailers and isolated stack frames:
		// these syntactically look like Critical stack frames but carry no
		// error signal, and counting them inflates the error tally (see #18).
		if level >= classify.Error && stackFrameLine.MatchString(line) {
			if isJestConsoleTrailer(lines, i) || isIsolatedStackFrame(lines, i) {
				level = classify.Info
			}
		}

		switch {
		case level >= classify.Error:
			errorCount++
			if len(errorLines) < 3 {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					errorLines = append(errorLines, trimmed)
				}
			}
		case level == classify.Warning:
			warningCount++
			if len(warningLines) < 2 {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					warningLines = append(warningLines, trimmed)
				}
			}
		}

		// Detect file paths
		matches := filePathPattern.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) > 1 {
				uniqueFiles[m[1]] = struct{}{}
			}
		}

		// Detect result signals
		trimmed := strings.TrimSpace(line)
		if resultFailPattern.MatchString(trimmed) {
			hasFail = true
		}
		if resultPassPattern.MatchString(trimmed) {
			hasPass = true
		}
	}

	result := resolveResult(lines, exitCode, stderr, hasFail, hasPass)

	// Build the summary header
	var sb strings.Builder
	sb.WriteString("[gotk summary]\n")
	fmt.Fprintf(&sb, "  total: %s lines (%s bytes)\n",
		formatNumber(len(lines)), formatNumber(totalBytes))
	fmt.Fprintf(&sb, "  errors: %s\n", formatNumber(errorCount))

	// Show key error lines
	for _, el := range errorLines {
		// Truncate long lines to keep summary compact
		if len(el) > 120 {
			el = el[:120]
		}
		fmt.Fprintf(&sb, "   → %s\n", el)
	}

	fmt.Fprintf(&sb, "  warnings: %s\n", formatNumber(warningCount))

	// Show key warning lines
	for _, wl := range warningLines {
		if len(wl) > 120 {
			wl = wl[:120]
		}
		fmt.Fprintf(&sb, "   → %s\n", wl)
	}

	if len(uniqueFiles) > 0 {
		fmt.Fprintf(&sb, "  files: %s unique paths mentioned\n",
			formatNumber(len(uniqueFiles)))
	}

	fmt.Fprintf(&sb, "  result: %s\n", result)
	sb.WriteString("[/gotk summary]\n\n")

	return sb.String() + input
}
