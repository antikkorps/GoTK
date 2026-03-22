package detect

import (
	"regexp"
	"strings"
)

var (
	// terraform refreshing state lines
	tfRefreshPattern = regexp.MustCompile(`^(\S+\.\S+): Refreshing state\.\.\.`)
	// terraform reading lines
	tfReadingPattern = regexp.MustCompile(`^(\S+\.\S+): Reading\.\.\.`)
	// terraform read complete
	tfReadComplete = regexp.MustCompile(`^(\S+\.\S+): Read complete after`)
	// terraform apply progress

	tfStillCreating  = regexp.MustCompile(`^(\S+\.\S+): Still creating\.\.\.`)
	tfStillModifying = regexp.MustCompile(`^(\S+\.\S+): Still modifying\.\.\.`)
	tfStillDestroying = regexp.MustCompile(`^(\S+\.\S+): Still destroying\.\.\.`)
)

// compressTerraformOutput compresses terraform plan/apply output.
// Preserves: plan changes, errors, warnings, summary.
// Removes: refreshing state lines, "still creating" progress, read complete lines.
func compressTerraformOutput(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	refreshCount := 0
	readCount := 0
	stillCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			flushTfCounters(&result, &refreshCount, &readCount, &stillCount)
			result = append(result, line)
			continue
		}

		// Count and skip "Refreshing state..." lines
		if tfRefreshPattern.MatchString(trimmed) {
			refreshCount++
			continue
		}

		// Count and skip "Reading..." and "Read complete" lines
		if tfReadingPattern.MatchString(trimmed) || tfReadComplete.MatchString(trimmed) {
			readCount++
			continue
		}

		// Count and skip "Still creating/modifying/destroying..." progress lines
		if tfStillCreating.MatchString(trimmed) || tfStillModifying.MatchString(trimmed) || tfStillDestroying.MatchString(trimmed) {
			stillCount++
			continue
		}

		// Flush counters before non-noise content
		flushTfCounters(&result, &refreshCount, &readCount, &stillCount)

		// Keep everything else: plan changes, creating/destroying actions, errors, summaries
		result = append(result, line)
	}

	flushTfCounters(&result, &refreshCount, &readCount, &stillCount)

	return strings.Join(result, "\n")
}

func flushTfCounters(result *[]string, refreshCount, readCount, stillCount *int) {
	if *refreshCount > 0 {
		*result = append(*result, "Refreshed "+itoa(*refreshCount)+" resources")
		*refreshCount = 0
	}
	if *readCount > 0 {
		*result = append(*result, "Read "+itoa(*readCount)+" data sources")
		*readCount = 0
	}
	if *stillCount > 0 {
		*result = append(*result, "("+itoa(*stillCount)+" progress updates)")
		*stillCount = 0
	}
}
