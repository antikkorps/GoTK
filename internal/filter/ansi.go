package filter

import "regexp"

// ansiPattern matches ANSI escape sequences (colors, cursor movement, etc.)
var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]|\x1b\].*?\x07|\x1b[()][AB012]|\x1b\[[\d;]*m`)

// StripANSI removes all ANSI escape codes from the output.
func StripANSI(input string) string {
	return ansiPattern.ReplaceAllString(input, "")
}
