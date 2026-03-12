package bench

import (
	"fmt"
	"strings"
)

// FormatPerFilter formats per-filter contribution data as a human-readable table.
func FormatPerFilter(fixtureName string, contributions []FilterContribution) string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Per-Filter Contribution (%s input)\n", fixtureName))
	sb.WriteString(strings.Repeat("=", 50+len(fixtureName)) + "\n\n")
	sb.WriteString(fmt.Sprintf("%-24s %10s %10s %10s %10s\n", "Filter", "Before", "After", "Reduction", "Time"))

	for _, c := range contributions {
		sb.WriteString(fmt.Sprintf("%-24s %10s %10s %9.1f%% %10s\n",
			c.FilterName,
			formatBytes(c.BytesBefore),
			formatBytes(c.BytesAfter),
			-c.Reduction,
			formatDuration(c.Duration),
		))
	}

	return sb.String()
}

// FormatLatency formats a LatencyReport as a human-readable summary.
func FormatLatency(report LatencyReport) string {
	var sb strings.Builder
	sb.WriteString("Latency Report\n")
	sb.WriteString("==============\n\n")
	sb.WriteString(fmt.Sprintf("  Iterations: %d\n", report.Iterations))
	sb.WriteString(fmt.Sprintf("  Input size: %s bytes\n", formatBytes(report.InputBytes)))
	sb.WriteString(fmt.Sprintf("  Min:        %s\n", formatDuration(report.Min)))
	sb.WriteString(fmt.Sprintf("  Max:        %s\n", formatDuration(report.Max)))
	sb.WriteString(fmt.Sprintf("  Mean:       %s\n", formatDuration(report.Mean)))
	sb.WriteString(fmt.Sprintf("  P50:        %s\n", formatDuration(report.P50)))
	sb.WriteString(fmt.Sprintf("  P95:        %s\n", formatDuration(report.P95)))
	sb.WriteString(fmt.Sprintf("  P99:        %s\n", formatDuration(report.P99)))
	return sb.String()
}
