package bench

import (
	"fmt"
	"strings"
)

// FormatReportJSON formats a Report as JSON without external dependencies.
func FormatReportJSON(report Report) string {
	var sb strings.Builder
	sb.WriteString("{\n")
	fmt.Fprintf(&sb, "  \"total_raw\": %d,\n", report.TotalRaw)
	fmt.Fprintf(&sb, "  \"total_clean\": %d,\n", report.TotalClean)
	fmt.Fprintf(&sb, "  \"avg_reduction\": %.2f,\n", report.AvgReduction)
	fmt.Fprintf(&sb, "  \"total_duration_us\": %d,\n", report.TotalDuration.Microseconds())
	sb.WriteString("  \"results\": [\n")

	for i, r := range report.Results {
		sb.WriteString("    {\n")
		fmt.Fprintf(&sb, "      \"name\": %s,\n", jsonString(r.Name))
		fmt.Fprintf(&sb, "      \"raw_bytes\": %d,\n", r.RawBytes)
		fmt.Fprintf(&sb, "      \"clean_bytes\": %d,\n", r.CleanBytes)
		fmt.Fprintf(&sb, "      \"reduction\": %.2f,\n", r.Reduction)
		fmt.Fprintf(&sb, "      \"duration_us\": %d,\n", r.Duration.Microseconds())
		fmt.Fprintf(&sb, "      \"lines_raw\": %d,\n", r.LinesRaw)
		fmt.Fprintf(&sb, "      \"lines_clean\": %d\n", r.LinesClean)
		sb.WriteString("    }")
		if i < len(report.Results)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("  ]\n")
	sb.WriteString("}\n")
	return sb.String()
}

// FormatPerFilterJSON formats per-filter contributions as JSON.
func FormatPerFilterJSON(fixtureName string, contributions []FilterContribution) string {
	var sb strings.Builder
	sb.WriteString("{\n")
	fmt.Fprintf(&sb, "  \"fixture\": %s,\n", jsonString(fixtureName))
	sb.WriteString("  \"filters\": [\n")

	for i, c := range contributions {
		sb.WriteString("    {\n")
		fmt.Fprintf(&sb, "      \"name\": %s,\n", jsonString(c.FilterName))
		fmt.Fprintf(&sb, "      \"bytes_before\": %d,\n", c.BytesBefore)
		fmt.Fprintf(&sb, "      \"bytes_after\": %d,\n", c.BytesAfter)
		fmt.Fprintf(&sb, "      \"reduction\": %.2f,\n", c.Reduction)
		fmt.Fprintf(&sb, "      \"duration_us\": %d\n", c.Duration.Microseconds())
		sb.WriteString("    }")
		if i < len(contributions)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("  ]\n")
	sb.WriteString("}\n")
	return sb.String()
}

// FormatQualityJSON formats a QualityReport as JSON.
func FormatQualityJSON(report QualityReport) string {
	var sb strings.Builder
	sb.WriteString("{\n")
	fmt.Fprintf(&sb, "  \"total_important\": %d,\n", report.TotalImportant)
	fmt.Fprintf(&sb, "  \"total_preserved\": %d,\n", report.TotalPreserved)
	fmt.Fprintf(&sb, "  \"overall_score\": %.2f,\n", report.OverallScore)
	fmt.Fprintf(&sb, "  \"all_perfect\": %t,\n", report.AllFixturesPerfect)
	sb.WriteString("  \"results\": [\n")

	for i, r := range report.Results {
		sb.WriteString("    {\n")
		fmt.Fprintf(&sb, "      \"name\": %s,\n", jsonString(r.Name))
		fmt.Fprintf(&sb, "      \"total_lines\": %d,\n", r.TotalLines)
		fmt.Fprintf(&sb, "      \"important_lines\": %d,\n", r.ImportantLines)
		fmt.Fprintf(&sb, "      \"preserved_lines\": %d,\n", r.PreservedLines)
		fmt.Fprintf(&sb, "      \"score\": %.2f,\n", r.Score)

		sb.WriteString("      \"missed_lines\": [")
		if len(r.MissedLines) > 0 {
			sb.WriteString("\n")
			for j, line := range r.MissedLines {
				fmt.Fprintf(&sb, "        %s", jsonString(line))
				if j < len(r.MissedLines)-1 {
					sb.WriteString(",")
				}
				sb.WriteString("\n")
			}
			sb.WriteString("      ]")
		} else {
			sb.WriteString("]")
		}

		sb.WriteString("\n    }")
		if i < len(report.Results)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}

	sb.WriteString("  ]\n")
	sb.WriteString("}\n")
	return sb.String()
}

// FormatLatencyJSON formats a LatencyReport as JSON.
func FormatLatencyJSON(report LatencyReport) string {
	var sb strings.Builder
	sb.WriteString("{\n")
	fmt.Fprintf(&sb, "  \"iterations\": %d,\n", report.Iterations)
	fmt.Fprintf(&sb, "  \"input_bytes\": %d,\n", report.InputBytes)
	fmt.Fprintf(&sb, "  \"min_us\": %d,\n", report.Min.Microseconds())
	fmt.Fprintf(&sb, "  \"max_us\": %d,\n", report.Max.Microseconds())
	fmt.Fprintf(&sb, "  \"mean_us\": %d,\n", report.Mean.Microseconds())
	fmt.Fprintf(&sb, "  \"p50_us\": %d,\n", report.P50.Microseconds())
	fmt.Fprintf(&sb, "  \"p95_us\": %d,\n", report.P95.Microseconds())
	fmt.Fprintf(&sb, "  \"p99_us\": %d\n", report.P99.Microseconds())
	sb.WriteString("}\n")
	return sb.String()
}

// FormatABTestJSON formats the A/B test report as JSON.
func FormatABTestJSON(report ABReport) string {
	var sb strings.Builder
	sb.WriteString("{\n")

	// Summary
	sb.WriteString("  \"summary\": [\n")
	for i, s := range report.Summary {
		sb.WriteString("    {\n")
		fmt.Fprintf(&sb, "      \"mode\": %s,\n", jsonString(s.Mode))
		fmt.Fprintf(&sb, "      \"total_raw_tokens\": %d,\n", s.TotalRawTokens)
		fmt.Fprintf(&sb, "      \"total_clean_tokens\": %d,\n", s.TotalCleanTokens)
		fmt.Fprintf(&sb, "      \"total_saved\": %d,\n", s.TotalSaved)
		fmt.Fprintf(&sb, "      \"avg_reduction\": %.2f,\n", s.AvgReduction)
		fmt.Fprintf(&sb, "      \"avg_quality\": %.2f,\n", s.AvgQuality)
		fmt.Fprintf(&sb, "      \"total_important\": %d,\n", s.TotalImportant)
		fmt.Fprintf(&sb, "      \"total_preserved\": %d\n", s.TotalPreserved)
		sb.WriteString("    }")
		if i < len(report.Summary)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("  ],\n")

	// Fixtures
	sb.WriteString("  \"fixtures\": [\n")
	for i, f := range report.Fixtures {
		sb.WriteString("    {\n")
		fmt.Fprintf(&sb, "      \"name\": %s,\n", jsonString(f.Name))
		sb.WriteString("      \"results\": [\n")
		for j, r := range f.Results {
			sb.WriteString("        {\n")
			fmt.Fprintf(&sb, "          \"mode\": %s,\n", jsonString(r.Mode))
			fmt.Fprintf(&sb, "          \"raw_tokens\": %d,\n", r.RawTokens)
			fmt.Fprintf(&sb, "          \"clean_tokens\": %d,\n", r.CleanTokens)
			fmt.Fprintf(&sb, "          \"tokens_saved\": %d,\n", r.TokensSaved)
			fmt.Fprintf(&sb, "          \"reduction_pct\": %.2f,\n", r.ReductionPct)
			fmt.Fprintf(&sb, "          \"quality_score\": %.2f,\n", r.QualityScore)
			fmt.Fprintf(&sb, "          \"important_lines\": %d,\n", r.ImportantLines)
			fmt.Fprintf(&sb, "          \"preserved_lines\": %d\n", r.PreservedLines)
			sb.WriteString("        }")
			if j < len(f.Results)-1 {
				sb.WriteString(",")
			}
			sb.WriteString("\n")
		}
		sb.WriteString("      ]\n")
		sb.WriteString("    }")
		if i < len(report.Fixtures)-1 {
			sb.WriteString(",")
		}
		sb.WriteString("\n")
	}
	sb.WriteString("  ]\n")

	sb.WriteString("}\n")
	return sb.String()
}

// jsonString returns a JSON-escaped string literal.
func jsonString(s string) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for _, c := range s {
		switch c {
		case '"':
			sb.WriteString("\\\"")
		case '\\':
			sb.WriteString("\\\\")
		case '\n':
			sb.WriteString("\\n")
		case '\r':
			sb.WriteString("\\r")
		case '\t':
			sb.WriteString("\\t")
		default:
			sb.WriteRune(c)
		}
	}
	sb.WriteByte('"')
	return sb.String()
}
