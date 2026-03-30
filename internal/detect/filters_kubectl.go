package detect

import (
	"regexp"
	"strings"
)

var (
	// kubectl managed fields block in YAML output
	kubeManagedFields = regexp.MustCompile(`^\s+managedFields:`)
	// kubectl annotation noise (long base64 or JSON in annotations)
	kubeLongAnnotation = regexp.MustCompile(`^\s+kubectl\.kubernetes\.io/(last-applied-configuration|rollout-status):\s*`)
	// helm verbose output
	helmDebugLine = regexp.MustCompile(`^(client|server)\.go:\d+:`)
)

// compressKubectlOutput compresses kubectl/helm output.
// Preserves: resource status, errors, events, conditions.
// Removes: managedFields blocks, last-applied-configuration annotations, helm debug.
func compressKubectlOutput(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	inManagedFields := false
	managedFieldsIndent := 0
	inLastApplied := false
	lastAppliedIndent := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track managedFields block — skip until we return to same or lower indent
		if kubeManagedFields.MatchString(line) {
			inManagedFields = true
			managedFieldsIndent = leadingSpaces(line)
			result = append(result, strings.Repeat(" ", managedFieldsIndent)+"managedFields: (omitted)")
			continue
		}
		if inManagedFields {
			if trimmed == "" || (trimmed != "" && leadingSpaces(line) > managedFieldsIndent) {
				continue // still inside managedFields
			}
			inManagedFields = false
		}

		// Skip last-applied-configuration annotation (can be very long JSON)
		if kubeLongAnnotation.MatchString(line) {
			inLastApplied = true
			lastAppliedIndent = leadingSpaces(line)
			result = append(result, strings.Repeat(" ", lastAppliedIndent)+"kubectl.kubernetes.io/last-applied-configuration: (omitted)")
			continue
		}
		if inLastApplied {
			if trimmed == "" {
				inLastApplied = false
				result = append(result, line)
				continue
			}
			currentIndent := leadingSpaces(line)
			if currentIndent > lastAppliedIndent {
				continue // still part of the annotation value
			}
			inLastApplied = false
		}

		// Skip helm debug lines
		if helmDebugLine.MatchString(trimmed) {
			continue
		}

		// Keep everything else
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// leadingSpaces returns the number of leading spaces in a line.
func leadingSpaces(s string) int {
	count := 0
	for _, ch := range s {
		switch ch {
		case ' ':
			count++
		case '\t':
			count += 4
		default:
			return count
		}
	}
	return count
}
