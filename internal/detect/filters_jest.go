package detect

import (
	"regexp"
	"strings"
)

var (
	// Lone `console.<method>` header as emitted by Jest's default reporter
	// (the test's `console.log(...)` gets wrapped in a header line, a message,
	// and an `at <path>:<line>:<col>` trailer — all on their own lines).
	jestConsoleHeader = regexp.MustCompile(`^\s*console\.(log|warn|error|info|debug)\s*$`)
	// Trailer emitted after a Jest-intercepted console message: a bare
	// `at <path>:<line>:<col>` with NO parentheses and NO function name.
	// Distinct from real stack frames which use `at <funcName> (<path>:<line>:<col>)`.
	jestConsoleTrailer = regexp.MustCompile(`^\s+at [^\s()]+:\d+:\d+\s*$`)
)

// stripJestConsoleBlocks removes the Jest reporter boilerplate that wraps
// every intercepted `console.log` call:
//
//	console.log
//	  <the logged message>
//	    at src/utils/foo.ts:42:13
//
// On a typical Jest run this pattern dominates the residual noise. The
// filter strips the `console.<method>` header and the `at` trailer but
// preserves the message lines in between.
//
// It is strict on both ends to avoid corrupting real error stack traces:
//   - header must be a lone `console.<method>` on its line
//   - trailer must be `  at <non-space-non-paren>:<N>:<N>` with NO parens
//
// When a `console.<method>` header is not followed by a matching trailer
// within a small look-ahead window (10 lines), the original content is
// emitted unchanged.
func stripJestConsoleBlocks(input string) string {
	lines := strings.Split(input, "\n")
	result := make([]string, 0, len(lines))

	i := 0
	for i < len(lines) {
		if !jestConsoleHeader.MatchString(lines[i]) {
			result = append(result, lines[i])
			i++
			continue
		}

		// Header candidate — look ahead for the bare `at …` trailer within a
		// small window of indented lines. Abort on any non-indented content
		// (preserves real error stack traces and mid-block separators).
		trailerIdx := -1
		maxLook := i + 10
		if maxLook >= len(lines) {
			maxLook = len(lines) - 1
		}
		for j := i + 1; j <= maxLook; j++ {
			line := lines[j]
			if line == "" {
				break
			}
			if !startsWithWhitespace(line) {
				break
			}
			if jestConsoleTrailer.MatchString(line) {
				trailerIdx = j
				break
			}
		}

		if trailerIdx < 0 {
			// No match — keep the header as-is and move on.
			result = append(result, lines[i])
			i++
			continue
		}

		// Emit the message lines (between header and trailer), skip both.
		for k := i + 1; k < trailerIdx; k++ {
			result = append(result, lines[k])
		}
		i = trailerIdx + 1
		// Jest separates each block with a blank line; absorb it so the output
		// doesn't end up double-spaced.
		if i < len(lines) && lines[i] == "" {
			i++
		}
	}

	return strings.Join(result, "\n")
}

func startsWithWhitespace(s string) bool {
	if s == "" {
		return false
	}
	return s[0] == ' ' || s[0] == '\t'
}
