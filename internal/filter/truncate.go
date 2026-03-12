package filter

import (
	"fmt"
	"strings"
)

// MaxLines is the default maximum number of lines to keep.
// Can be overridden via config. 0 means no limit.
var MaxLines = 50

// HeadTailRatio controls how many lines come from head vs tail.
// 0.7 means 70% head, 30% tail.
const HeadTailRatio = 0.7

// Truncate limits output to MaxLines, keeping head and tail with a summary.
func Truncate(input string) string {
	if MaxLines <= 0 {
		return input
	}

	lines := strings.Split(input, "\n")

	// Remove trailing empty line from split
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) <= MaxLines {
		return input
	}

	headCount := int(float64(MaxLines) * HeadTailRatio)
	tailCount := MaxLines - headCount
	omitted := len(lines) - headCount - tailCount

	var result []string
	result = append(result, lines[:headCount]...)
	result = append(result, fmt.Sprintf("\n[... %d lines omitted ...]\n", omitted))
	result = append(result, lines[len(lines)-tailCount:]...)

	return strings.Join(result, "\n") + "\n"
}
