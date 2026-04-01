package detect

import (
	"strconv"
	"strings"
)

// compressJqOutput compresses verbose JSON output from jq/yq.
// When JSON has deeply nested arrays with many elements, it collapses
// array elements beyond a threshold. Preserves structure and errors.
func compressJqOutput(input string) string {
	lines := strings.Split(input, "\n")

	if len(lines) <= 50 {
		return input // small output, keep as-is
	}

	var result []string
	arrayDepth := 0
	arrayElementCount := 0
	suppressedCount := 0
	suppressIndent := -1

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track array depth
		if trimmed == "[" || strings.HasSuffix(trimmed, "[") {
			arrayDepth++
			arrayElementCount = 0
			result = append(result, line)
			continue
		}

		if trimmed == "]" || trimmed == "]," {
			if suppressedCount > 0 {
				indent := strings.Repeat(" ", suppressIndent)
				result = append(result, indent+"  ... "+strconv.Itoa(suppressedCount)+" more elements")
				suppressedCount = 0
				suppressIndent = -1
			}
			arrayDepth--
			arrayElementCount = 0
			result = append(result, line)
			continue
		}

		// Inside an array, count top-level elements (lines starting with "{" at this depth)
		if arrayDepth > 0 && (trimmed == "{" || strings.HasPrefix(trimmed, "{")) {
			arrayElementCount++

			// After 10 elements in the same array, suppress
			if arrayElementCount > 10 {
				if suppressIndent == -1 {
					suppressIndent = len(line) - len(strings.TrimLeft(line, " \t"))
				}
				// Skip all lines until we find the closing of this object
				suppressedCount++
				continue
			}
		}

		// If we're suppressing, skip until we hit the array end or a new top-level element
		if suppressedCount > 0 && arrayDepth > 0 {
			// Keep going until we find a line at the array's indent level
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}
