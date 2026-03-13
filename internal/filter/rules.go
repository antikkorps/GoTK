package filter

import (
	"regexp"
	"strings"
)

// RemoveByRules returns a filter that removes lines matching any of the given patterns.
// Invalid regex patterns are silently skipped.
func RemoveByRules(patterns []string) func(string) string {
	compiled := compilePatterns(patterns)
	if len(compiled) == 0 {
		return func(s string) string { return s }
	}

	return func(input string) string {
		lines := strings.Split(input, "\n")
		var result []string
		for _, line := range lines {
			if !matchesAny(line, compiled) {
				result = append(result, line)
			}
		}
		return strings.Join(result, "\n")
	}
}

// KeepByRules returns a filter that restores lines from the original input
// if they match any keep pattern and are missing from the filtered output.
// This should wrap the final output of the chain to ensure important lines survive.
func KeepByRules(patterns []string, originalInput string) func(string) string {
	compiled := compilePatterns(patterns)
	if len(compiled) == 0 {
		return func(s string) string { return s }
	}

	return func(filtered string) string {
		// Find lines from original that match keep patterns
		origLines := strings.Split(originalInput, "\n")
		var mustKeep []string
		for _, line := range origLines {
			if matchesAny(line, compiled) {
				mustKeep = append(mustKeep, line)
			}
		}
		if len(mustKeep) == 0 {
			return filtered
		}

		// Check which kept lines are already present
		var missing []string
		for _, kl := range mustKeep {
			if !strings.Contains(filtered, kl) {
				missing = append(missing, kl)
			}
		}
		if len(missing) == 0 {
			return filtered
		}

		// Append missing lines at the end with a marker
		result := filtered
		if !strings.HasSuffix(result, "\n") && result != "" {
			result += "\n"
		}
		result += "[gotk: preserved by always_keep rule]\n"
		result += strings.Join(missing, "\n")
		return result
	}
}

func compilePatterns(patterns []string) []*regexp.Regexp {
	var compiled []*regexp.Regexp
	for _, p := range patterns {
		if re, err := regexp.Compile(p); err == nil {
			compiled = append(compiled, re)
		}
	}
	return compiled
}

func matchesAny(line string, patterns []*regexp.Regexp) bool {
	for _, re := range patterns {
		if re.MatchString(line) {
			return true
		}
	}
	return false
}
