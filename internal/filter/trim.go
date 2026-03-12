package filter

import "strings"

// TrimEmpty removes lines that are purely whitespace or decorative.
// This is the last filter in the chain — final cleanup pass.
func TrimEmpty(input string) string {
	lines := strings.Split(input, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip purely decorative lines (only dashes, equals, underscores, etc.)
		if isDecorative(trimmed) {
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// isDecorative returns true if the line is purely visual decoration.
// Conservative: only matches lines that are EXCLUSIVELY repeated separator chars
// with no semantic content whatsoever.
func isDecorative(line string) bool {
	if line == "" {
		return false // Keep intentional blank lines (already normalized)
	}

	// Lines made entirely of repeated decorative chars
	decorChars := "-=_~"
	first := line[0]
	if strings.IndexByte(decorChars, first) == -1 {
		return false
	}

	for _, c := range line {
		if c != rune(first) && c != ' ' {
			return false
		}
	}

	// Only decorative if long enough to clearly be a visual separator (10+ chars).
	// Short lines like "---" can be semantically meaningful (markdown, go test, diffs).
	return len(strings.TrimSpace(line)) >= 10
}
