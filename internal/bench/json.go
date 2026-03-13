package bench

import (
	"fmt"
	"strings"
)

// FormatReportJSON formats a Report as JSON without external dependencies.
func FormatReportJSON(report Report) string {
	var sb strings.Builder
	sb.WriteString("{\n")
	sb.WriteString(fmt.Sprintf("  \"total_raw\": %d,\n", report.TotalRaw))
	sb.WriteString(fmt.Sprintf("  \"total_clean\": %d,\n", report.TotalClean))
	sb.WriteString(fmt.Sprintf("  \"avg_reduction\": %.2f,\n", report.AvgReduction))
	sb.WriteString(fmt.Sprintf("  \"total_duration_us\": %d,\n", report.TotalDuration.Microseconds()))
	sb.WriteString("  \"results\": [\n")

	for i, r := range report.Results {
		sb.WriteString("    {\n")
		sb.WriteString(fmt.Sprintf("      \"name\": %s,\n", jsonString(r.Name)))
		sb.WriteString(fmt.Sprintf("      \"raw_bytes\": %d,\n", r.RawBytes))
		sb.WriteString(fmt.Sprintf("      \"clean_bytes\": %d,\n", r.CleanBytes))
		sb.WriteString(fmt.Sprintf("      \"reduction\": %.2f,\n", r.Reduction))
		sb.WriteString(fmt.Sprintf("      \"duration_us\": %d,\n", r.Duration.Microseconds()))
		sb.WriteString(fmt.Sprintf("      \"lines_raw\": %d,\n", r.LinesRaw))
		sb.WriteString(fmt.Sprintf("      \"lines_clean\": %d\n", r.LinesClean))
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
	sb.WriteString(fmt.Sprintf("  \"fixture\": %s,\n", jsonString(fixtureName)))
	sb.WriteString("  \"filters\": [\n")

	for i, c := range contributions {
		sb.WriteString("    {\n")
		sb.WriteString(fmt.Sprintf("      \"name\": %s,\n", jsonString(c.FilterName)))
		sb.WriteString(fmt.Sprintf("      \"bytes_before\": %d,\n", c.BytesBefore))
		sb.WriteString(fmt.Sprintf("      \"bytes_after\": %d,\n", c.BytesAfter))
		sb.WriteString(fmt.Sprintf("      \"reduction\": %.2f,\n", c.Reduction))
		sb.WriteString(fmt.Sprintf("      \"duration_us\": %d\n", c.Duration.Microseconds()))
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
	sb.WriteString(fmt.Sprintf("  \"total_important\": %d,\n", report.TotalImportant))
	sb.WriteString(fmt.Sprintf("  \"total_preserved\": %d,\n", report.TotalPreserved))
	sb.WriteString(fmt.Sprintf("  \"overall_score\": %.2f,\n", report.OverallScore))
	sb.WriteString(fmt.Sprintf("  \"all_perfect\": %t,\n", report.AllFixturesPerfect))
	sb.WriteString("  \"results\": [\n")

	for i, r := range report.Results {
		sb.WriteString("    {\n")
		sb.WriteString(fmt.Sprintf("      \"name\": %s,\n", jsonString(r.Name)))
		sb.WriteString(fmt.Sprintf("      \"total_lines\": %d,\n", r.TotalLines))
		sb.WriteString(fmt.Sprintf("      \"important_lines\": %d,\n", r.ImportantLines))
		sb.WriteString(fmt.Sprintf("      \"preserved_lines\": %d,\n", r.PreservedLines))
		sb.WriteString(fmt.Sprintf("      \"score\": %.2f,\n", r.Score))

		sb.WriteString("      \"missed_lines\": [")
		if len(r.MissedLines) > 0 {
			sb.WriteString("\n")
			for j, line := range r.MissedLines {
				sb.WriteString(fmt.Sprintf("        %s", jsonString(line)))
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
	sb.WriteString(fmt.Sprintf("  \"iterations\": %d,\n", report.Iterations))
	sb.WriteString(fmt.Sprintf("  \"input_bytes\": %d,\n", report.InputBytes))
	sb.WriteString(fmt.Sprintf("  \"min_us\": %d,\n", report.Min.Microseconds()))
	sb.WriteString(fmt.Sprintf("  \"max_us\": %d,\n", report.Max.Microseconds()))
	sb.WriteString(fmt.Sprintf("  \"mean_us\": %d,\n", report.Mean.Microseconds()))
	sb.WriteString(fmt.Sprintf("  \"p50_us\": %d,\n", report.P50.Microseconds()))
	sb.WriteString(fmt.Sprintf("  \"p95_us\": %d,\n", report.P95.Microseconds()))
	sb.WriteString(fmt.Sprintf("  \"p99_us\": %d\n", report.P99.Microseconds()))
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
