package filter

import "strings"

// Dedup removes consecutive duplicate lines.
// Keeps the first occurrence and replaces runs with a count marker.
func Dedup(input string) string {
	if input == "" {
		return input
	}

	lines := strings.Split(input, "\n")
	if len(lines) == 0 {
		return input
	}

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
	return "  ... (" + itoa(count) + " duplicate lines)"
}

// Simple int to string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
