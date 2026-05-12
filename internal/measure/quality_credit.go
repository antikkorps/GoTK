package measure

import (
	"regexp"
	"strconv"
	"strings"
)

// summaryMarkers are regexes whose capture group is the number of
// classifier-important lines that a filter intentionally compressed away.
// They cover the specific phrases emitted by our own filters; we keep the
// list narrow on purpose to avoid crediting unrelated text that happens to
// look summary-shaped. See issue #60.
var summaryMarkers = []*regexp.Regexp{
	// npm filter: "... and 14 more npm warnings"
	regexp.MustCompile(`\.\.\. and (\d+) more npm warnings`),
	// node filter: "... and 7 more deprecation warnings"
	regexp.MustCompile(`\.\.\. and (\d+) more deprecation warnings`),
	// node filter (parallel workers): "... and 3 identical warnings from other workers"
	regexp.MustCompile(`\.\.\. and (\d+) identical warnings? from other workers`),
	// node filter: "(5 experimental warnings)"
	regexp.MustCompile(`\((\d+) experimental warnings\)`),
	// stacktrace filter: "[... 4 more errors with identical stack]"
	regexp.MustCompile(`\[\.\.\. (\d+) more \S+ with identical stack\]`),
}

// SummaryMarkerCredit scans cleanOutput for compression-summary markers
// (e.g. "... and 14 more npm warnings") and returns the total number of
// classifier-important lines that the filter chose to compress. Callers
// should credit these as preserved when computing quality scores.
func SummaryMarkerCredit(cleanOutput string) int {
	if cleanOutput == "" {
		return 0
	}
	total := 0
	for _, re := range summaryMarkers {
		for _, m := range re.FindAllStringSubmatch(cleanOutput, -1) {
			if n, err := strconv.Atoi(m[1]); err == nil {
				total += n
			}
		}
	}
	return total
}

// dropSafeNoise matches classifier-elevated lines that carry no signal for
// an LLM and that our filters intentionally remove. The classifier elevates
// them on lexical grounds ("Warning:" prefix), but their meaning in context
// is pure banner / connection chatter.
var dropSafeNoise = []*regexp.Regexp{
	// ssh: "Warning: Permanently added 'host' (ED25519) to the list of known hosts."
	regexp.MustCompile(`^Warning: Permanently added .* known hosts`),
}

// IsDropSafeNoise reports whether a classifier-important line is universally
// drop-safe. When such a line is absent from the cleaned output, callers
// should treat it as preserved (the filter did the right thing by dropping
// it). See issue #60.
func IsDropSafeNoise(line string) bool {
	line = strings.TrimSpace(line)
	for _, re := range dropSafeNoise {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}
