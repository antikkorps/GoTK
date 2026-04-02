package filter

import (
	"strconv"
	"strings"
)

// Dedup removes consecutive duplicate lines.
// Keeps the first occurrence and replaces runs with a count marker.
func Dedup(input string) string {
	if input == "" {
		return input
	}

	lines := strings.Split(input, "\n")

	var result []string
	prev := lines[0]
	result = append(result, prev)
	dupCount := 0

	lines = lines[1:]

	for _, line := range lines {
		if line == prev {
			dupCount++
			continue
		}

		if dupCount > 0 {
			result = append(result, formatDupMarker(dupCount))
			dupCount = 0
		}

		result = append(result, line)
		prev = line
	}

	// Handle trailing duplicates
	if dupCount > 0 {
		result = append(result, formatDupMarker(dupCount))
	}

	return strings.Join(result, "\n")
}

func formatDupMarker(count int) string {
	if count == 1 {
		return "  ... (1 duplicate line)"
	}
	return "  ... (" + strconv.Itoa(count) + " duplicate lines)"
}
