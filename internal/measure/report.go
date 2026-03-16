package measure

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Report holds aggregated measurement data.
type Report struct {
	Period           string                     `json:"period"`
	TotalInvocations int                        `json:"total_invocations"`
	TotalRawTokens   int                        `json:"total_raw_tokens"`
	TotalCleanTokens int                        `json:"total_clean_tokens"`
	TotalTokensSaved int                        `json:"total_tokens_saved"`
	AvgReduction     float64                    `json:"avg_reduction"`
	AvgQualityScore  float64                    `json:"avg_quality_score"`
	ByCommandType    map[string]*CommandTypeStats `json:"by_command_type"`
	BySessions       []SessionStats             `json:"by_sessions"`
}

// CommandTypeStats holds stats for a single command type.
type CommandTypeStats struct {
	Invocations  int     `json:"invocations"`
	RawTokens    int     `json:"raw_tokens"`
	CleanTokens  int     `json:"clean_tokens"`
	TokensSaved  int     `json:"tokens_saved"`
	AvgReduction float64 `json:"avg_reduction"`
}

// SessionStats holds stats for a single session.
type SessionStats struct {
	SessionID   string `json:"session_id"`
	Invocations int    `json:"invocations"`
	RawTokens   int    `json:"raw_tokens"`
	CleanTokens int    `json:"clean_tokens"`
	TokensSaved int    `json:"tokens_saved"`
}

// GenerateReport aggregates entries into a report.
// period is a label like "today", "7d", "30d", "all".
func GenerateReport(entries []Entry, period string) Report {
	r := Report{
		Period:        period,
		ByCommandType: make(map[string]*CommandTypeStats),
	}

	sessionMap := make(map[string]*SessionStats)

	for _, e := range entries {
		r.TotalInvocations++
		r.TotalRawTokens += e.RawTokens
		r.TotalCleanTokens += e.CleanTokens
		r.TotalTokensSaved += e.TokensSaved
		r.AvgQualityScore += e.QualityScore

		// By command type
		ct := e.CommandType
		if ct == "" {
			ct = "unknown"
		}
		stats, ok := r.ByCommandType[ct]
		if !ok {
			stats = &CommandTypeStats{}
			r.ByCommandType[ct] = stats
		}
		stats.Invocations++
		stats.RawTokens += e.RawTokens
		stats.CleanTokens += e.CleanTokens
		stats.TokensSaved += e.TokensSaved

		// By session
		sid := e.SessionID
		if sid == "" {
			sid = "unknown"
		}
		ss, ok := sessionMap[sid]
		if !ok {
			ss = &SessionStats{SessionID: sid}
			sessionMap[sid] = ss
		}
		ss.Invocations++
		ss.RawTokens += e.RawTokens
		ss.CleanTokens += e.CleanTokens
		ss.TokensSaved += e.TokensSaved
	}

	// Compute averages
	if r.TotalInvocations > 0 {
		r.AvgQualityScore /= float64(r.TotalInvocations)
		if r.TotalRawTokens > 0 {
			r.AvgReduction = float64(r.TotalTokensSaved) / float64(r.TotalRawTokens) * 100
		}
	}

	// Compute per-command-type averages
	for _, stats := range r.ByCommandType {
		if stats.RawTokens > 0 {
			stats.AvgReduction = float64(stats.TokensSaved) / float64(stats.RawTokens) * 100
		}
	}

	// Flatten sessions
	for _, ss := range sessionMap {
		r.BySessions = append(r.BySessions, *ss)
	}

	return r
}

// FilterEntriesByPeriod filters entries based on a period string.
func FilterEntriesByPeriod(entries []Entry, period string) []Entry {
	if period == "all" || period == "" {
		return entries
	}

	now := time.Now().UTC()
	var since time.Time

	switch period {
	case "today":
		since = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	case "7d":
		since = now.AddDate(0, 0, -7)
	case "30d":
		since = now.AddDate(0, 0, -30)
	default:
		return entries
	}

	sinceStr := since.Format(time.RFC3339)
	var filtered []Entry
	for _, e := range entries {
		if e.Timestamp >= sinceStr {
			filtered = append(filtered, e)
		}
	}
	return filtered
}

// FormatReport returns a human-readable table report.
func FormatReport(r Report) string {
	var b strings.Builder

	fmt.Fprintf(&b, "GoTK Measurement Report — %s\n", r.Period)
	fmt.Fprintf(&b, "%s\n\n", strings.Repeat("=", 50))

	fmt.Fprintf(&b, "Total invocations:  %d\n", r.TotalInvocations)
	fmt.Fprintf(&b, "Total raw tokens:   %d\n", r.TotalRawTokens)
	fmt.Fprintf(&b, "Total clean tokens: %d\n", r.TotalCleanTokens)
	fmt.Fprintf(&b, "Total tokens saved: %d\n", r.TotalTokensSaved)
	fmt.Fprintf(&b, "Avg reduction:      %.1f%%\n", r.AvgReduction)
	fmt.Fprintf(&b, "Avg quality score:  %.1f%%\n", r.AvgQualityScore)

	if len(r.ByCommandType) > 0 {
		fmt.Fprintf(&b, "\nBy command type:\n")
		fmt.Fprintf(&b, "  %-12s %6s %10s %10s %8s\n", "TYPE", "COUNT", "RAW TOK", "SAVED", "REDUC")
		fmt.Fprintf(&b, "  %s\n", strings.Repeat("-", 50))
		for name, s := range r.ByCommandType {
			fmt.Fprintf(&b, "  %-12s %6d %10d %10d %7.1f%%\n",
				name, s.Invocations, s.RawTokens, s.TokensSaved, s.AvgReduction)
		}
	}

	if len(r.BySessions) > 0 {
		fmt.Fprintf(&b, "\nSessions: %d\n", len(r.BySessions))
	}

	return b.String()
}

// FormatLast returns a human-readable table of the last N entries with a totals row.
func FormatLast(entries []Entry, n int) string {
	if len(entries) == 0 {
		return "No measurement entries found.\n"
	}

	// Take the last N entries
	start := 0
	if len(entries) > n {
		start = len(entries) - n
	}
	subset := entries[start:]

	var b strings.Builder
	fmt.Fprintf(&b, "GoTK — Last %d invocations\n", len(subset))
	fmt.Fprintf(&b, "%s\n\n", strings.Repeat("=", 78))
	fmt.Fprintf(&b, "  %-4s %-5s  %-28s %7s %7s %7s %6s %7s\n",
		"#", "TIME", "COMMAND", "RAW", "CLEAN", "SAVED", "REDUC", "QUAL")
	fmt.Fprintf(&b, "  %s\n", strings.Repeat("-", 74))

	var totalRaw, totalClean, totalSaved int
	for i, e := range subset {
		// Parse timestamp to show just HH:MM
		timeStr := "??:??"
		if t, err := time.Parse(time.RFC3339, e.Timestamp); err == nil {
			timeStr = t.Local().Format("15:04")
		}

		// Truncate command for display
		cmd := e.Command
		if len(cmd) > 28 {
			cmd = cmd[:25] + "..."
		}

		fmt.Fprintf(&b, "  %-4d %-5s  %-28s %7d %7d %7d %5.1f%% %6.1f%%\n",
			i+1, timeStr, cmd,
			e.RawTokens, e.CleanTokens, e.TokensSaved,
			e.ReductionPct, e.QualityScore)

		totalRaw += e.RawTokens
		totalClean += e.CleanTokens
		totalSaved += e.TokensSaved
	}

	fmt.Fprintf(&b, "  %s\n", strings.Repeat("-", 74))
	totalReduc := 0.0
	if totalRaw > 0 {
		totalReduc = float64(totalSaved) / float64(totalRaw) * 100
	}
	fmt.Fprintf(&b, "  %-4s %-5s  %-28s %7d %7d %7d %5.1f%%\n",
		"", "", "TOTAL", totalRaw, totalClean, totalSaved, totalReduc)

	return b.String()
}

// FormatReportJSON returns the report as indented JSON.
func FormatReportJSON(r Report) string {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	return string(data) + "\n"
}
