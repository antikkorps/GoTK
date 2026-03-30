package filter

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/antikkorps/GoTK/internal/classify"
)

// SummaryThreshold is the minimum number of lines before a summary is generated.
const SummaryThreshold = 100

// filePathPattern matches file paths like path/to/file.ext: or standalone paths.
var filePathPattern = regexp.MustCompile(`(?:^|\s)((?:[a-zA-Z]:)?(?:[a-zA-Z0-9_./-]+/)+[a-zA-Z0-9_.-]+\.[a-zA-Z0-9]+)(?::\d+)?`)

// resultFailPattern detects failure signals in output.
var resultFailPattern = regexp.MustCompile(`(?i)\b(FAIL|FAILED|(?:^|\s)ERROR(?:\s|$)|panic)\b`)

// resultPassPattern detects success signals in output.
var resultPassPattern = regexp.MustCompile(`(?i)\b(PASS|(?:^|\s)ok(?:\s|$)|SUCCESS|0 errors)\b`)

// formatNumber formats an integer with comma separators for readability.
func formatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	var result []byte
	remainder := len(s) % 3
	if remainder > 0 {
		result = append(result, s[:remainder]...)
	}
	for i := remainder; i < len(s); i += 3 {
		if len(result) > 0 {
			result = append(result, ',')
		}
		result = append(result, s[i:i+3]...)
	}
	return string(result)
}

// Summarize analyzes the full output and prepends a structured summary.
// Only activates when output exceeds SummaryThreshold lines.
func Summarize(input string) string {
	lines := strings.Split(input, "\n")

	// Remove trailing empty line from split artifact
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	if len(lines) < SummaryThreshold {
		return input
	}

	totalBytes := len(input)

	var errorCount, warningCount int
	var errorLines []string
	var warningLines []string

	uniqueFiles := make(map[string]struct{})
	hasFail := false
	hasPass := false

	for _, line := range lines {
		level := classify.Classify(line)

		switch {
		case level >= classify.Error:
			errorCount++
			if len(errorLines) < 3 {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					errorLines = append(errorLines, trimmed)
				}
			}
		case level == classify.Warning:
			warningCount++
			if len(warningLines) < 2 {
				trimmed := strings.TrimSpace(line)
				if trimmed != "" {
					warningLines = append(warningLines, trimmed)
				}
			}
		}

		// Detect file paths
		matches := filePathPattern.FindAllStringSubmatch(line, -1)
		for _, m := range matches {
			if len(m) > 1 {
				uniqueFiles[m[1]] = struct{}{}
			}
		}

		// Detect result signals
		trimmed := strings.TrimSpace(line)
		if resultFailPattern.MatchString(trimmed) {
			hasFail = true
		}
		if resultPassPattern.MatchString(trimmed) {
			hasPass = true
		}
	}

	// Determine overall result
	result := "unknown"
	if hasFail {
		result = "FAIL"
	} else if hasPass {
		result = "PASS"
	}

	// Build the summary header
	var sb strings.Builder
	sb.WriteString("[gotk summary]\n")
	fmt.Fprintf(&sb, "  total: %s lines (%s bytes)\n",
		formatNumber(len(lines)), formatNumber(totalBytes))
	fmt.Fprintf(&sb, "  errors: %s\n", formatNumber(errorCount))

	// Show key error lines
	for _, el := range errorLines {
		// Truncate long lines to keep summary compact
		if len(el) > 120 {
			el = el[:120]
		}
		fmt.Fprintf(&sb, "   → %s\n", el)
	}

	fmt.Fprintf(&sb, "  warnings: %s\n", formatNumber(warningCount))

	// Show key warning lines
	for _, wl := range warningLines {
		if len(wl) > 120 {
			wl = wl[:120]
		}
		fmt.Fprintf(&sb, "   → %s\n", wl)
	}

	if len(uniqueFiles) > 0 {
		fmt.Fprintf(&sb, "  files: %s unique paths mentioned\n",
			formatNumber(len(uniqueFiles)))
	}

	fmt.Fprintf(&sb, "  result: %s\n", result)
	sb.WriteString("[/gotk summary]\n\n")

	return sb.String() + input
}
