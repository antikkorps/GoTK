package detect

import (
	"regexp"
	"strings"
)

var (
	// npm progress bar lines
	npmProgressPattern = regexp.MustCompile(`^npm http`)
	// npm timing detail lines
	npmTimingPattern = regexp.MustCompile(`^npm timing`)
	// npm warn lines
	npmWarnPattern = regexp.MustCompile(`^npm warn`)
	// npm audit individual package detail lines (indented package info)
	npmAuditPkgPattern = regexp.MustCompile(`^\s+(Severity|Vulnerable|Patched|Dependency|Path|More info):`)
	// yarn resolving progress
	yarnResolvingPattern = regexp.MustCompile(`^\[[\d/]+\] Resolving packages`)
	yarnFetchingPattern  = regexp.MustCompile(`^\[[\d/]+\] Fetching packages`)
	yarnLinkingPattern   = regexp.MustCompile(`^\[[\d/]+\] Linking dependencies`)
)

// compressNpmOutput removes redundant npm/yarn install output.
func compressNpmOutput(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	warnCount := 0
	firstWarn := ""
	auditDetails := 0
	auditSummaryStarted := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Skip npm progress and timing lines
		if npmProgressPattern.MatchString(trimmed) {
			continue
		}
		if npmTimingPattern.MatchString(trimmed) {
			continue
		}

		// Compress npm warn deprecation spam: keep first, count rest
		if npmWarnPattern.MatchString(trimmed) {
			warnCount++
			if warnCount == 1 {
				firstWarn = line
			}
			continue
		}

		// Flush accumulated warnings before a non-warn line
		if warnCount > 0 {
			result = append(result, firstWarn)
			if warnCount > 1 {
				result = append(result, "... and "+itoa(warnCount-1)+" more npm warnings")
			}
			warnCount = 0
			firstWarn = ""
		}

		// npm audit: compress individual package details when there are many
		if strings.Contains(trimmed, "vulnerabilities") || strings.Contains(trimmed, "found 0") {
			// This is a summary line, keep it
			auditSummaryStarted = true
			result = append(result, line)
			continue
		}
		if !auditSummaryStarted && npmAuditPkgPattern.MatchString(trimmed) {
			auditDetails++
			if auditDetails <= 10 {
				result = append(result, line)
			} else if auditDetails == 11 {
				result = append(result, "  ... additional vulnerability details omitted")
			}
			continue
		}

		// Yarn: skip resolving/fetching/linking progress lines (keep last one)
		if yarnResolvingPattern.MatchString(trimmed) ||
			yarnFetchingPattern.MatchString(trimmed) ||
			yarnLinkingPattern.MatchString(trimmed) {
			continue
		}

		// Keep errors, summaries, and everything else
		result = append(result, line)
	}

	// Flush trailing warnings
	if warnCount > 0 {
		result = append(result, firstWarn)
		if warnCount > 1 {
			result = append(result, "... and "+itoa(warnCount-1)+" more npm warnings")
		}
	}

	return strings.Join(result, "\n")
}
