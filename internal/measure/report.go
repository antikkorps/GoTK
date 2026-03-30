package measure

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// Report holds aggregated measurement data.
type Report struct {
	Period           string                       `json:"period"`
	TotalInvocations int                          `json:"total_invocations"`
	TotalRawTokens   int                          `json:"total_raw_tokens"`
	TotalCleanTokens int                          `json:"total_clean_tokens"`
	TotalTokensSaved int                          `json:"total_tokens_saved"`
	AvgReduction     float64                      `json:"avg_reduction"`
	AvgQualityScore  float64                      `json:"avg_quality_score"`
	TotalReRequests  int                          `json:"total_rerequests"`
	ReRequestRate    float64                      `json:"rerequest_rate"`
	ByCommandType    map[string]*CommandTypeStats `json:"by_command_type"`
	BySessions       []SessionStats               `json:"by_sessions"`
	Insights         []Insight                    `json:"insights,omitempty"`
}

// CommandTypeStats holds stats for a single command type.
type CommandTypeStats struct {
	Invocations   int     `json:"invocations"`
	RawTokens     int     `json:"raw_tokens"`
	CleanTokens   int     `json:"clean_tokens"`
	TokensSaved   int     `json:"tokens_saved"`
	AvgReduction  float64 `json:"avg_reduction"`
	ReRequests    int     `json:"rerequests"`
	ReRequestRate float64 `json:"rerequest_rate"`
}

// Insight is a quality feedback suggestion based on measurement data.
type Insight struct {
	Level   string `json:"level"`   // "warning", "info"
	Command string `json:"command"` // command type concerned
	Message string `json:"message"`
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
		if e.ReRequest {
			r.TotalReRequests++
		}

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
		if e.ReRequest {
			stats.ReRequests++
		}

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
		r.ReRequestRate = float64(r.TotalReRequests) / float64(r.TotalInvocations) * 100
	}

	// Compute per-command-type averages and re-request rates
	for _, stats := range r.ByCommandType {
		if stats.RawTokens > 0 {
			stats.AvgReduction = float64(stats.TokensSaved) / float64(stats.RawTokens) * 100
		}
		if stats.Invocations > 0 {
			stats.ReRequestRate = float64(stats.ReRequests) / float64(stats.Invocations) * 100
		}
	}

	// Flatten sessions
	for _, ss := range sessionMap {
		r.BySessions = append(r.BySessions, *ss)
	}

	// Generate quality insights
	r.Insights = generateInsights(r)

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
	if r.TotalReRequests > 0 {
		fmt.Fprintf(&b, "Re-requests:        %d (%.1f%% of invocations)\n", r.TotalReRequests, r.ReRequestRate)
	}

	if len(r.ByCommandType) > 0 {
		fmt.Fprintf(&b, "\nBy command type:\n")
		fmt.Fprintf(&b, "  %-12s %6s %10s %10s %8s %6s\n", "TYPE", "COUNT", "RAW TOK", "SAVED", "REDUC", "RE-REQ")
		fmt.Fprintf(&b, "  %s\n", strings.Repeat("-", 58))
		for name, s := range r.ByCommandType {
			reReqStr := "-"
			if s.ReRequests > 0 {
				reReqStr = fmt.Sprintf("%d", s.ReRequests)
			}
			fmt.Fprintf(&b, "  %-12s %6d %10d %10d %7.1f%% %6s\n",
				name, s.Invocations, s.RawTokens, s.TokensSaved, s.AvgReduction, reReqStr)
		}
	}

	if len(r.BySessions) > 0 {
		fmt.Fprintf(&b, "\nSessions: %d\n", len(r.BySessions))
	}

	if len(r.Insights) > 0 {
		fmt.Fprintf(&b, "\nQuality Insights:\n")
		for _, ins := range r.Insights {
			prefix := "  [!]"
			if ins.Level == "info" {
				prefix = "  [i]"
			}
			fmt.Fprintf(&b, "%s %s\n", prefix, ins.Message)
		}
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
		reTag := ""
		if e.ReRequest {
			reTag = " [RE]"
		}
		maxCmd := 28 - len(reTag)
		if len(cmd) > maxCmd {
			cmd = cmd[:maxCmd-3] + "..."
		}
		cmd += reTag

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

// generateInsights analyzes measurement data and produces actionable suggestions.
func generateInsights(r Report) []Insight {
	var insights []Insight

	// Collect command types with high re-request rates (>10%, at least 3 invocations)
	type cmdRate struct {
		name string
		rate float64
		reqs int
	}
	var highReReq []cmdRate
	for name, stats := range r.ByCommandType {
		if stats.Invocations >= 3 && stats.ReRequestRate > 10 {
			highReReq = append(highReReq, cmdRate{name, stats.ReRequestRate, stats.ReRequests})
		}
	}
	// Sort by rate descending
	sort.Slice(highReReq, func(i, j int) bool { return highReReq[i].rate > highReReq[j].rate })

	for _, cr := range highReReq {
		msg := fmt.Sprintf("%s has %.0f%% re-request rate (%d/%d) — consider increasing truncation limit: [truncation] %s = <higher value>",
			cr.name, cr.rate, cr.reqs, r.ByCommandType[cr.name].Invocations, cr.name)
		insights = append(insights, Insight{Level: "warning", Command: cr.name, Message: msg})
	}

	// Check for commands with very high reduction that also have re-requests
	for name, stats := range r.ByCommandType {
		if stats.AvgReduction > 90 && stats.ReRequests > 0 && stats.Invocations >= 3 {
			// Skip if already flagged above
			alreadyFlagged := false
			for _, cr := range highReReq {
				if cr.name == name {
					alreadyFlagged = true
					break
				}
			}
			if !alreadyFlagged {
				msg := fmt.Sprintf("%s has %.0f%% reduction with %d re-request(s) — aggressive filtering may be removing useful content",
					name, stats.AvgReduction, stats.ReRequests)
				insights = append(insights, Insight{Level: "info", Command: name, Message: msg})
			}
		}
	}

	// Overall health check
	if r.TotalInvocations >= 10 && r.ReRequestRate > 15 {
		msg := fmt.Sprintf("Overall re-request rate is %.0f%% — consider switching to a less aggressive profile or mode", r.ReRequestRate)
		insights = append(insights, Insight{Level: "warning", Command: "", Message: msg})
	}

	// Positive feedback when things look good
	if r.TotalInvocations >= 10 && r.TotalReRequests == 0 && r.AvgReduction > 50 {
		msg := fmt.Sprintf("No re-requests detected with %.0f%% avg reduction — filtering quality is excellent", r.AvgReduction)
		insights = append(insights, Insight{Level: "info", Command: "", Message: msg})
	}

	return insights
}

// FormatReportJSON returns the report as indented JSON.
func FormatReportJSON(r Report) string {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Sprintf(`{"error": %q}`, err.Error())
	}
	return string(data) + "\n"
}
