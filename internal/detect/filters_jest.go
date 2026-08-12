package detect

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/antikkorps/GoTK/internal/filter"
)

var (
	// Lone `console.<method>` header as emitted by Jest's default reporter
	// (the test's `console.log(...)` gets wrapped in a header line, a message,
	// and an `at <path>:<line>:<col>` trailer — all on their own lines).
	jestConsoleHeader = regexp.MustCompile(`^\s*console\.(log|warn|error|info|debug)\s*$`)
	// Trailer emitted after a Jest-intercepted console message. Jest emits two
	// shapes depending on whether it could name the calling frame:
	//
	//	at src/utils/foo.ts:42:13                (anonymous)
	//	at log (middlewares/jwt.auth.js:85:13)   (named, parenthesised)
	//
	// Only the first shape used to be recognised, on the assumption that
	// parentheses meant "real stack frame". In practice the named shape is what
	// Jest emits for almost every application log, so every console triplet
	// survived the filter and flooded the output (see issue #79).
	//
	// Parenthesised frames are now accepted, and real stack traces are kept
	// safe by a separate structural check instead: a console trailer is a lone
	// frame, whereas an error stack has consecutive ones (see anyStackFrame).
	jestConsoleTrailer = regexp.MustCompile(`^\s+at (?:[^\s()]+:\d+:\d+|[^\s()]+ \([^()]+:\d+:\d+\))\s*$`)

	// anyStackFrame matches any `at …` frame line. Used purely as a guard: if
	// the line following a trailer candidate is also a frame, the block is a
	// genuine multi-frame stack trace and must be left untouched.
	anyStackFrame = regexp.MustCompile(`^\s+at\s`)
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
//   - the trailer must be a single frame, never part of a consecutive run
//
// When a `console.<method>` header is not followed by a matching trailer
// within a small look-ahead window (10 lines), the original content is
// emitted unchanged.
//
// Consecutive blocks whose message is byte-identical are collapsed to one
// copy plus a count marker. Parallel runners emit the same setup log once per
// worker, so a 6-worker run repeats its banner six times in a row; without
// this the repeats eat the truncation budget that should go to real output.
// The generic Dedup filter cannot do it, because it runs earlier in the chain
// — at that point the messages are still separated by their header and
// trailer lines, so they are not consecutive duplicates yet (see issue #79).
func stripJestConsoleBlocks(input string) string {
	lines := strings.Split(input, "\n")
	result := make([]string, 0, len(lines))

	// On a run the runner itself reports as green, application logs carry no
	// signal an LLM can act on — they are the happy path narrating itself —
	// yet on a large suite they crowd everything else out of the truncation
	// window. Drop them, replaced by one marker stating the count, so the
	// reduction is visible rather than silent (see issue #79).
	//
	// Only the runner's own verdict counts here. Without a totals line the
	// verdict is unknown, and unknown means keep.
	verdict, haveVerdict := filter.DetectRunnerResult(lines)
	dropBlocks := haveVerdict && verdict == "PASS"
	droppedBlocks := 0
	dropMarkerIdx := -1

	// Pending run of identical console messages awaiting a count marker.
	var pending []string
	pendingCount := 0

	flushPending := func() {
		// len(pending) can be 0 when a header is immediately followed by its
		// trailer — an empty log call. Nothing to align a marker against.
		if pendingCount > 1 && len(pending) > 0 {
			result = append(result, fmt.Sprintf("%s... and %d identical console blocks",
				leadingWhitespace(pending[0]), pendingCount-1))
		}
		pending = nil
		pendingCount = 0
	}

	i := 0
	for i < len(lines) {
		if !jestConsoleHeader.MatchString(lines[i]) {
			flushPending()
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
				// A lone frame closes a console block; consecutive frames mean
				// this is a real stack trace, which must survive intact.
				if j+1 < len(lines) && anyStackFrame.MatchString(lines[j+1]) {
					break
				}
				trailerIdx = j
				break
			}
		}

		if trailerIdx < 0 {
			// No match — keep the header as-is and move on.
			flushPending()
			result = append(result, lines[i])
			i++
			continue
		}

		// The message lines between header and trailer are the only part worth
		// keeping. Identical consecutive messages collapse into one.
		message := lines[i+1 : trailerIdx]
		switch {
		case dropBlocks:
			droppedBlocks++
			if dropMarkerIdx < 0 {
				// Reserve the slot where the first dropped block stood, so the
				// marker lands in context instead of at the end of the output.
				dropMarkerIdx = len(result)
				result = append(result, "")
			}
		case pendingCount > 0 && slices.Equal(message, pending):
			pendingCount++
		default:
			flushPending()
			result = append(result, message...)
			pending = message
			pendingCount = 1
		}

		i = trailerIdx + 1
		// Jest separates each block with a blank line; absorb it so the output
		// doesn't end up double-spaced.
		if i < len(lines) && lines[i] == "" {
			i++
		}
	}
	flushPending()

	if dropMarkerIdx >= 0 {
		noun := "blocks"
		if droppedBlocks == 1 {
			noun = "block"
		}
		result[dropMarkerIdx] = fmt.Sprintf("  [gotk: %d console.* log %s dropped — run passed]",
			droppedBlocks, noun)
	}

	return strings.Join(result, "\n")
}

// leadingWhitespace returns the indentation prefix of a line, so a count
// marker can be aligned with the block it summarizes.
func leadingWhitespace(s string) string {
	return s[:len(s)-len(strings.TrimLeft(s, " \t"))]
}

func startsWithWhitespace(s string) bool {
	if s == "" {
		return false
	}
	return s[0] == ' ' || s[0] == '\t'
}
