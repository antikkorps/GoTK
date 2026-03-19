package learn

import (
	"regexp"
	"strings"
)

// Normalization replaces variable parts of a line with regex-friendly placeholders.
// This allows grouping lines that differ only in values (hashes, timestamps, numbers, etc.)
// into a single pattern.

var (
	// UUID: 8-4-4-4-12 hex format
	reUUID = regexp.MustCompile(`[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}`)

	// Hex hashes: 7+ hex chars (git SHAs, build hashes)
	reHexHash = regexp.MustCompile(`\b[0-9a-f]{7,40}\b`)

	// ISO8601 timestamps: 2024-01-15T10:30:00Z or 2024-01-15 10:30:00
	reISO8601 = regexp.MustCompile(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}[\w:.+-]*`)

	// Common log timestamps: [2024/01/15 10:30:00] or 2024/01/15
	reLogDate = regexp.MustCompile(`\d{4}[/-]\d{2}[/-]\d{2}`)

	// Time-only patterns: 10:30:00.123
	reTime = regexp.MustCompile(`\d{2}:\d{2}:\d{2}[.\d]*`)

	// Numeric values: standalone numbers (integers, floats, hex with 0x prefix)
	reNumber = regexp.MustCompile(`\b\d+(\.\d+)?\b`)

	// File sizes with units: 1.5MB, 200KB, 3GB
	reFileSize = regexp.MustCompile(`\b\d+(\.\d+)?\s*(B|KB|MB|GB|TB|KiB|MiB|GiB|TiB|bytes)\b`)

	// Duration patterns: 1.5s, 200ms, 3m2s
	reDuration = regexp.MustCompile(`\b\d+(\.\d+)?\s*(ms|µs|ns|us|s|m|h)\b`)

	// IP addresses
	reIP = regexp.MustCompile(`\b\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(:\d+)?\b`)

	// Version numbers: v1.2.3, 1.2.3-beta
	reVersion = regexp.MustCompile(`v?\d+\.\d+\.\d+[-\w.]*`)
)

// Normalize replaces variable parts of a line with placeholders.
// The result can be used as a grouping key for frequency analysis.
func Normalize(line string) string {
	s := strings.TrimSpace(line)
	if s == "" {
		return ""
	}

	// Apply replacements in order from most specific to least specific
	// to avoid partial matches.
	s = reUUID.ReplaceAllString(s, "<UUID>")
	s = reIP.ReplaceAllString(s, "<IP>")
	s = reISO8601.ReplaceAllString(s, "<TIMESTAMP>")
	s = reLogDate.ReplaceAllString(s, "<DATE>")
	s = reTime.ReplaceAllString(s, "<TIME>")
	s = reFileSize.ReplaceAllString(s, "<SIZE>")
	s = reDuration.ReplaceAllString(s, "<DURATION>")
	s = reVersion.ReplaceAllString(s, "<VERSION>")
	s = reHexHash.ReplaceAllString(s, "<HASH>")
	s = reNumber.ReplaceAllString(s, "<N>")

	return s
}

// ToRegex converts a normalized line back into a regex pattern
// suitable for use in always_remove rules.
func ToRegex(normalized string) string {
	// Escape regex metacharacters in the literal parts
	parts := strings.Split(normalized, "<")
	var result strings.Builder

	result.WriteString("^\\s*")

	for i, part := range parts {
		if i == 0 {
			result.WriteString(regexEscape(part))
			continue
		}

		closeBracket := strings.IndexByte(part, '>')
		if closeBracket < 0 {
			result.WriteString("<")
			result.WriteString(regexEscape(part))
			continue
		}

		placeholder := part[:closeBracket]
		rest := part[closeBracket+1:]

		switch placeholder {
		case "UUID":
			result.WriteString(`[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}`)
		case "HASH":
			result.WriteString(`[0-9a-f]+`)
		case "TIMESTAMP":
			result.WriteString(`\d{4}-\d{2}-\d{2}[T ]\d{2}:\d{2}:\d{2}\S*`)
		case "DATE":
			result.WriteString(`\d{4}[-/]\d{2}[-/]\d{2}`)
		case "TIME":
			result.WriteString(`\d{2}:\d{2}:\d{2}\S*`)
		case "SIZE":
			result.WriteString(`\d+(\.\d+)?\s*(B|KB|MB|GB|TB|KiB|MiB|GiB|TiB|bytes)`)
		case "DURATION":
			result.WriteString(`\d+(\.\d+)?\s*(ms|µs|ns|us|s|m|h)`)
		case "IP":
			result.WriteString(`\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}(:\d+)?`)
		case "VERSION":
			result.WriteString(`v?\d+\.\d+\.\d+[-\w.]*`)
		case "N":
			result.WriteString(`\d+(\.\d+)?`)
		default:
			result.WriteString("<")
			result.WriteString(regexEscape(placeholder))
			result.WriteString(">")
		}

		result.WriteString(regexEscape(rest))
	}

	return result.String()
}

// regexEscape escapes regex metacharacters in a literal string.
func regexEscape(s string) string {
	var b strings.Builder
	for _, c := range s {
		switch c {
		case '.', '*', '+', '?', '(', ')', '[', ']', '{', '}', '|', '^', '$', '\\':
			b.WriteByte('\\')
		}
		b.WriteRune(c)
	}
	return b.String()
}
