package filter

import (
	"fmt"
	"regexp"
	"strings"
)

// AutoEscalateMode controls how truncation adapts when failure signals
// are detected in the output.
type AutoEscalateMode string

const (
	// EscalateOff disables escalation — truncation always keeps head + tail
	// regardless of failure signals.
	EscalateOff AutoEscalateMode = "off"
	// EscalateHint keeps the head + tail layout but appends a one-line footer
	// inviting the caller to re-run with fuller output when failures are seen.
	EscalateHint AutoEscalateMode = "hint"
	// EscalateWindow (default) keeps head + tail + ±N lines around every
	// detected failure anchor, so critical error context is never dropped
	// into the omitted middle.
	EscalateWindow AutoEscalateMode = "window"
	// EscalateConservative skips truncation entirely when a failure is
	// detected, at the cost of a larger output.
	EscalateConservative AutoEscalateMode = "conservative"
)

// DefaultEscalateWindow is the number of lines kept before and after each
// failure anchor in window mode.
const DefaultEscalateWindow = 10

// escalateCapMultiplier caps the total kept lines at N × maxLines when many
// anchors are present, to prevent the output from ballooning past reasonable
// LLM context sizes. The caller still gets head + tail + first + last anchor
// windows if the cap is hit.
const escalateCapMultiplier = 2

// failureAnchors match lines that indicate a real test/build/runtime failure.
// These are deliberately tight to avoid false positives (e.g. "FAIL" in code
// strings): each anchor requires a language/tool-specific prefix or structure.
var failureAnchors = []*regexp.Regexp{
	// Test runners
	regexp.MustCompile(`^FAIL\s+\S`),                 // "FAIL tests/foo.test.js" (Jest, go test file prefix)
	regexp.MustCompile(`^---\s+FAIL:`),               // go test: "--- FAIL: TestFoo"
	regexp.MustCompile(`^FAILED\b`),                  // generic "FAILED"
	regexp.MustCompile(`^\s*●\s`),                    // Jest failure marker
	regexp.MustCompile(`\b[1-9]\d*\s+failed\b`),      // "N failed" with N > 0 (jest/pytest/cargo totals)
	regexp.MustCompile(`(?i)^FAILED\s*\([a-z]+=\d+`), // Python unittest: "FAILED (failures=3)"

	// Exceptions / runtime errors
	regexp.MustCompile(`^Error:`),
	regexp.MustCompile(`^\s*[A-Z]\w*(Error|Exception):`),
	regexp.MustCompile(`^panic:`),
	regexp.MustCompile(`^fatal:`),
	regexp.MustCompile(`^Traceback \(most recent call`),

	// Compilers / build tools
	regexp.MustCompile(`(?i)\berror\s+TS\d+`), // TypeScript (tsc)
	regexp.MustCompile(`\berror\[E\d+\]`),     // Rust
	regexp.MustCompile(`^BUILD FAILED`),       // Gradle, Maven
	regexp.MustCompile(`^\s*error:`),          // Go/cc/clang generic
	regexp.MustCompile(`^##\[error\]`),        // GitHub Actions error annotation
}

// ParseAutoEscalate converts a string to an AutoEscalateMode, returning
// EscalateWindow for unknown values.
func ParseAutoEscalate(s string) AutoEscalateMode {
	switch strings.ToLower(s) {
	case "off", "none", "disabled":
		return EscalateOff
	case "hint":
		return EscalateHint
	case "window":
		return EscalateWindow
	case "conservative", "full":
		return EscalateConservative
	default:
		return EscalateWindow
	}
}

// findFailureAnchors returns the indices of lines matching any failure anchor.
// The indices are returned in ascending order.
func findFailureAnchors(lines []string) []int {
	var idx []int
	for i, line := range lines {
		if isFailureAnchor(line) {
			idx = append(idx, i)
		}
	}
	return idx
}

// isFailureAnchor reports whether a line matches any failure anchor pattern.
func isFailureAnchor(line string) bool {
	for _, re := range failureAnchors {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}

// TruncateWithEscalation returns a FilterFunc that limits output to maxLines
// with behavior adapted to whether failure signals are detected in the input.
// When no failures are present, the output is head + tail (identical to
// TruncateWithLimit). When failures are present, the mode decides how to
// preserve failure context — see AutoEscalateMode values.
//
// window ≤ 0 falls back to DefaultEscalateWindow. mode == off delegates to
// TruncateWithLimit unconditionally.
func TruncateWithEscalation(maxLines int, mode AutoEscalateMode, window int) FilterFunc {
	if mode == EscalateOff {
		return TruncateWithLimit(maxLines)
	}
	if window <= 0 {
		window = DefaultEscalateWindow
	}
	legacy := TruncateWithLimit(maxLines)

	return func(input string) string {
		if maxLines <= 0 {
			return input
		}

		lines := strings.Split(input, "\n")
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}

		if len(lines) <= maxLines {
			return input
		}

		anchors := findFailureAnchors(lines)
		if len(anchors) == 0 {
			return legacy(input)
		}

		switch mode {
		case EscalateHint:
			return legacy(input) + "[gotk] failure signals detected — re-run with --auto-escalate=window or GOTK_PASSTHROUGH=1 for full failure context.\n"
		case EscalateConservative:
			return input
		case EscalateWindow:
			return windowTruncate(lines, anchors, maxLines, window)
		}
		return legacy(input)
	}
}

// windowTruncate keeps head + tail + ±window lines around each anchor index,
// producing the same "[... N lines omitted ...]" markers as TruncateWithLimit
// between kept ranges.
func windowTruncate(lines []string, anchors []int, maxLines, window int) string {
	n := len(lines)
	keep := make([]bool, n)

	headCount := int(float64(maxLines) * HeadTailRatio)
	tailCount := maxLines - headCount
	if headCount > n {
		headCount = n
	}
	if tailCount > n {
		tailCount = n
	}

	markKeep := func(lo, hi int) {
		if lo < 0 {
			lo = 0
		}
		if hi >= n {
			hi = n - 1
		}
		for i := lo; i <= hi; i++ {
			keep[i] = true
		}
	}

	markKeep(0, headCount-1)
	markKeep(n-tailCount, n-1)
	for _, a := range anchors {
		markKeep(a-window, a+window)
	}

	// Safety cap: if too many anchors would keep most of the output, fall back
	// to head + tail + first-anchor window + last-anchor window.
	cap := escalateCapMultiplier * maxLines
	kept := 0
	for _, k := range keep {
		if k {
			kept++
		}
	}
	if kept > cap && len(anchors) > 2 {
		for i := range keep {
			keep[i] = false
		}
		markKeep(0, headCount-1)
		markKeep(n-tailCount, n-1)
		markKeep(anchors[0]-window, anchors[0]+window)
		markKeep(anchors[len(anchors)-1]-window, anchors[len(anchors)-1]+window)
	}

	var out []string
	gapStart := -1
	flushGap := func(end int) {
		if gapStart < 0 {
			return
		}
		omitted := end - gapStart
		if omitted > 0 {
			out = append(out, fmt.Sprintf("\n[... %d lines omitted ...]\n", omitted))
		}
		gapStart = -1
	}

	for i := 0; i < n; i++ {
		if keep[i] {
			flushGap(i)
			out = append(out, lines[i])
		} else if gapStart < 0 {
			gapStart = i
		}
	}
	flushGap(n)

	return strings.Join(out, "\n") + "\n"
}
