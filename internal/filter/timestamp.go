package filter

import (
	"regexp"
	"strings"
)

// timestampPrefix matches a leading timestamp on an output line: optional
// indentation, an optional opening bracket, an optional ISO date part, a
// HH:MM:SS clock with optional fractional seconds and timezone, an optional
// closing bracket, and exactly one separator character.
//
// Two deliberate choices:
//
//   - Anchored at line start, so a clock appearing mid-line — a duration
//     column (`time=00:00:04.00`), a value inside a message — is never touched.
//   - Exactly one separator character is consumed. Runners like Astro align
//     nested output with extra spaces after the timestamp (`12:35:03   ├─ …`);
//     eating them all would flatten the hierarchy the indentation encodes.
var timestampPrefix = regexp.MustCompile(
	`^([ \t]*)\[?(?:\d{4}[-/]\d{2}[-/]\d{2}[T ])?\d{2}:\d{2}:\d{2}(?:[.,]\d{1,9})?(?:Z|[+-]\d{2}:?\d{2})?\]?[ \t]`)

// minTimestampLines is how many prefixed lines are required before stripping
// happens at all. A log prefix repeats on every line; a lone line that happens
// to open with a clock-like token is data, and data is left alone.
const minTimestampLines = 3

// StripTimestamps removes leading timestamp prefixes from output lines.
//
// Timestamps are the highest-volume, lowest-signal token in build and test
// output: every line of an Astro, Nitro, or docker-compose run carries one,
// they carry no information an LLM can act on, and they defeat Dedup — two
// otherwise-identical lines differ only by their clock, so the run never
// collapses.
//
// The filter never drops a line, only a prefix, so no error, warning, or
// diagnostic content can be lost. Three guards keep it honest: the pattern is
// anchored at line start, at least minTimestampLines lines must carry a prefix
// before anything is stripped, and a line whose entire content is a timestamp
// is left intact rather than blanked.
func StripTimestamps(input string) string {
	if input == "" {
		return input
	}

	lines := strings.Split(input, "\n")

	prefixed := 0
	for _, line := range lines {
		if timestampPrefix.MatchString(line) {
			prefixed++
			if prefixed >= minTimestampLines {
				break
			}
		}
	}
	if prefixed < minTimestampLines {
		return input
	}

	for i, line := range lines {
		if stripped, ok := stripTimestampPrefix(line); ok {
			lines[i] = stripped
		}
	}

	return strings.Join(lines, "\n")
}

// stripTimestampPrefix removes the timestamp prefix from a single line,
// reporting whether anything was stripped. Shared by StripTimestamps (batch)
// and StreamFilter (line-at-a-time) so both modes strip identically.
func stripTimestampPrefix(line string) (string, bool) {
	loc := timestampPrefix.FindStringSubmatchIndex(line)
	if loc == nil {
		return line, false
	}
	rest := line[loc[1]:]
	if strings.TrimSpace(rest) == "" {
		// Timestamp with nothing after it: stripping would leave a blank line
		// where the original carried at least a "something happened here"
		// marker. Keep it.
		return line, false
	}
	// Preserve any indentation that preceded the timestamp (capture group 1).
	return line[loc[2]:loc[3]] + rest, true
}

// hasTimestampPrefix reports whether a line opens with a timestamp, without
// modifying it. Used by the streaming filter to reach minTimestampLines.
func hasTimestampPrefix(line string) bool {
	return timestampPrefix.MatchString(line)
}
