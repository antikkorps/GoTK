package bench

import (
	"fmt"
	"strings"
)

// FormatPerFilter formats per-filter contribution data as a human-readable table.
func FormatPerFilter(fixtureName string, contributions []FilterContribution) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Per-Filter Contribution (%s input)\n", fixtureName)
	sb.WriteString(strings.Repeat("=", 50+len(fixtureName)) + "\n\n")
	fmt.Fprintf(&sb, "%-24s %10s %10s %10s %10s\n", "Filter", "Before", "After", "Reduction", "Time")

	for _, c := range contributions {
		fmt.Fprintf(&sb, "%-24s %10s %10s %9.1f%% %10s\n",
			c.FilterName,
			formatBytes(c.BytesBefore),
			formatBytes(c.BytesAfter),
			-c.Reduction,
			formatDuration(c.Duration),
		)
	}

	return sb.String()
}

// FormatQuality formats a QualityReport as a human-readable table.
func FormatQuality(report QualityReport) string {
	var sb strings.Builder
	sb.WriteString("Quality Score Report\n")
	sb.WriteString("====================\n\n")
	fmt.Fprintf(&sb, "%-24s %10s %10s %10s\n", "Fixture", "Important", "Preserved", "Score")

	for _, r := range report.Results {
		marker := ""
		if r.Score < 100 {
			marker = " <!>"
		}
		fmt.Fprintf(&sb, "%-24s %10d %10d %9.1f%%%s\n",
			r.Name,
			r.ImportantLines,
			r.PreservedLines,
			r.Score,
			marker,
		)
	}

	fmt.Fprintf(&sb, "\n%-24s %10d %10d %9.1f%%\n",
		"Total",
		report.TotalImportant,
		report.TotalPreserved,
		report.OverallScore,
	)

	if !report.AllFixturesPerfect {
		sb.WriteString("\nMissed lines:\n")
		for _, r := range report.Results {
			for _, line := range r.MissedLines {
				fmt.Fprintf(&sb, "  [%s] %s\n", r.Name, line)
			}
		}
	}

	return sb.String()
}

// FormatLatency formats a LatencyReport as a human-readable summary.
func FormatLatency(report LatencyReport) string {
	var sb strings.Builder
	sb.WriteString("Latency Report\n")
	sb.WriteString("==============\n\n")
	fmt.Fprintf(&sb, "  Iterations: %d\n", report.Iterations)
	fmt.Fprintf(&sb, "  Input size: %s bytes\n", formatBytes(report.InputBytes))
	fmt.Fprintf(&sb, "  Min:        %s\n", formatDuration(report.Min))
	fmt.Fprintf(&sb, "  Max:        %s\n", formatDuration(report.Max))
	fmt.Fprintf(&sb, "  Mean:       %s\n", formatDuration(report.Mean))
	fmt.Fprintf(&sb, "  P50:        %s\n", formatDuration(report.P50))
	fmt.Fprintf(&sb, "  P95:        %s\n", formatDuration(report.P95))
	fmt.Fprintf(&sb, "  P99:        %s\n", formatDuration(report.P99))
	return sb.String()
}
