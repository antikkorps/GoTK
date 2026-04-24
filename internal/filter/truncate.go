package filter

import (
	"fmt"
	"strings"
)

// DefaultMaxLines is the default maximum number of lines to keep.
const DefaultMaxLines = 50

// HeadTailRatio controls how many lines come from head vs tail.
// 0.7 means 70% head, 30% tail.
const HeadTailRatio = 0.7

// TruncateWithLimit returns a FilterFunc that limits output to maxLines,
// keeping head and tail with a summary. 0 or negative means no limit.
//
// Test-runner summary anchors (jest/vitest/pytest/cargo/go test totals and
// duration) are pinned: if they would fall in the omitted middle, they are
// inserted just after the head so the LLM always sees the final counts. See
// issue #40.
func TruncateWithLimit(maxLines int) FilterFunc {
	return func(input string) string {
		if maxLines <= 0 {
			return input
		}

		lines := strings.Split(input, "\n")

		// Remove trailing empty line from split
		if len(lines) > 0 && lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}

		if len(lines) <= maxLines {
			return input
		}

		headCount := int(float64(maxLines) * HeadTailRatio)
		tailCount := maxLines - headCount
		tailStart := len(lines) - tailCount

		// Collect summary anchors that land in the omitted middle.
		var pinned []string
		for _, idx := range findSummaryAnchors(lines) {
			if idx >= headCount && idx < tailStart {
				pinned = append(pinned, lines[idx])
			}
		}

		omitted := tailStart - headCount
		var result []string
		result = append(result, lines[:headCount]...)
		if len(pinned) > 0 {
			result = append(result, fmt.Sprintf("\n[... %d lines omitted; summary pinned below ...]", omitted))
			result = append(result, pinned...)
			result = append(result, "\n")
		} else {
			result = append(result, fmt.Sprintf("\n[... %d lines omitted ...]\n", omitted))
		}
		result = append(result, lines[tailStart:]...)

		return strings.Join(result, "\n") + "\n"
	}
}
