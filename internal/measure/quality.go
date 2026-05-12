package measure

import (
	"strings"

	"github.com/antikkorps/GoTK/internal/classify"
)

// ComputeQualityScore returns the percentage of important lines (Warning+)
// from rawOutput that are preserved in cleanOutput. Returns 100.0 if there
// are no important lines.
func ComputeQualityScore(rawOutput, cleanOutput string) (score float64, importantCount int) {
	_, levels := classify.ClassifyLines(rawOutput)
	rawLines := strings.Split(rawOutput, "\n")

	// Count important lines
	var importantLines []string
	for i, level := range levels {
		if level >= classify.Warning {
			importantLines = append(importantLines, strings.TrimSpace(rawLines[i]))
		}
	}

	importantCount = len(importantLines)
	if importantCount == 0 {
		return 100.0, 0
	}

	// Check how many survived in the clean output
	preserved := 0
	for _, imp := range importantLines {
		if imp == "" {
			preserved++
			continue
		}
		if strings.Contains(cleanOutput, imp) {
			preserved++
			continue
		}
		// Try extracting core content (strip file:line: prefix)
		core := extractCore(imp)
		if core != "" && strings.Contains(cleanOutput, core) {
			preserved++
			continue
		}
		// Filter intentionally drops universally-noisy banners — credit
		// them as preserved so the score reflects intent, not the lexical
		// "Warning:" prefix the classifier keyed on. See issue #60.
		if IsDropSafeNoise(imp) {
			preserved++
		}
	}

	// Credit lines that were compressed into structural summary markers
	// ("... and 14 more npm warnings", etc.) — they survive semantically,
	// just not verbatim. Cap at the number of dropped lines so a marker
	// can't push the score above 100%.
	dropped := importantCount - preserved
	if credit := SummaryMarkerCredit(cleanOutput); credit > 0 && dropped > 0 {
		if credit > dropped {
			credit = dropped
		}
		preserved += credit
	}

	score = float64(preserved) / float64(importantCount) * 100
	return score, importantCount
}

// extractCore strips common structural prefixes (file:line:) to find
// the semantically important part of a line.
func extractCore(line string) string {
	parts := strings.SplitN(line, ":", 3)
	if len(parts) == 3 {
		content := strings.TrimSpace(parts[2])
		if content != "" {
			return content
		}
	}
	return ""
}

// CountLines returns the number of lines in s.
func CountLines(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}
