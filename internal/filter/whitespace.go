package filter

import (
	"regexp"
	"strings"
)

var (
	// trailingSpaces matches trailing whitespace on each line
	trailingSpaces = regexp.MustCompile(`[ \t]+\n`)
	// multipleBlankLines matches 3+ consecutive blank lines → collapse to 1
	multipleBlankLines = regexp.MustCompile(`\n{3,}`)
)

// NormalizeWhitespace cleans up excessive whitespace without losing structure.
func NormalizeWhitespace(input string) string {
	// Remove trailing spaces on each line
	result := trailingSpaces.ReplaceAllString(input, "\n")

	// Collapse multiple blank lines into one
	result = multipleBlankLines.ReplaceAllString(result, "\n\n")

	// Remove leading/trailing whitespace from entire output
	result = strings.TrimSpace(result)

	if result != "" {
		result += "\n"
	}

	return result
}
