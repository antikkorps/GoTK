package filter

import (
	"strings"

	"github.com/antikkorps/GoTK/internal/classify"
)

// MinLevel sets the minimum importance level for the priority filter.
// Lines below this level are removed. Default is Debug (keeps everything
// except Noise). Set to classify.Info for aggressive mode.
var MinLevel = int(classify.Debug)

// PriorityFilter removes lines below the configured minimum importance level.
// Error and Critical lines are never removed regardless of MinLevel.
// Context lines (1 before and 1 after Error/Critical) are also preserved.
func PriorityFilter(input string) string {
	lines, levels := classify.ClassifyLines(input)
	if len(lines) == 0 {
		return input
	}

	minLevel := classify.Level(MinLevel)

	// First pass: mark which lines to keep.
	keep := make([]bool, len(lines))
	for i, level := range levels {
		// Always keep Error and Critical lines.
		if level >= classify.Error {
			keep[i] = true
			// Preserve 1 line of context before and after.
			if i > 0 {
				keep[i-1] = true
			}
			if i < len(lines)-1 {
				keep[i+1] = true
			}
			continue
		}
		// Keep lines at or above the minimum level.
		if level >= minLevel {
			keep[i] = true
		}
	}

	var result []string
	for i, line := range lines {
		if keep[i] {
			result = append(result, line)
		}
	}

	return strings.Join(result, "\n")
}
