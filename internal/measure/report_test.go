package measure

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestGenerateReport(t *testing.T) {
	entries := []Entry{
		{
			SessionID:    "s1",
			Command:      "grep -rn foo .",
			CommandType:  "grep",
			RawTokens:    1000,
			CleanTokens:  200,
			TokensSaved:  800,
			ReductionPct: 80.0,
			QualityScore: 100.0,
		},
		{
			SessionID:    "s1",
			Command:      "find . -name '*.go'",
			CommandType:  "find",
			RawTokens:    500,
			CleanTokens:  100,
			TokensSaved:  400,
			ReductionPct: 80.0,
			QualityScore: 95.0,
		},
		{
			SessionID:    "s2",
			Command:      "grep bar .",
			CommandType:  "grep",
			RawTokens:    300,
			CleanTokens:  60,
			TokensSaved:  240,
			ReductionPct: 80.0,
			QualityScore: 100.0,
		},
	}

	r := GenerateReport(entries, "all")

	if r.TotalInvocations != 3 {
		t.Errorf("TotalInvocations = %d, want 3", r.TotalInvocations)
	}
	if r.TotalRawTokens != 1800 {
		t.Errorf("TotalRawTokens = %d, want 1800", r.TotalRawTokens)
	}
	if r.TotalTokensSaved != 1440 {
		t.Errorf("TotalTokensSaved = %d, want 1440", r.TotalTokensSaved)
	}

	grepStats, ok := r.ByCommandType["grep"]
	if !ok {
		t.Fatal("missing grep in ByCommandType")
	}
	if grepStats.Invocations != 2 {
		t.Errorf("grep invocations = %d, want 2", grepStats.Invocations)
	}

	if len(r.BySessions) != 2 {
		t.Errorf("sessions = %d, want 2", len(r.BySessions))
	}
}

func TestGenerateReportEmpty(t *testing.T) {
	r := GenerateReport(nil, "all")
	if r.TotalInvocations != 0 {
		t.Errorf("TotalInvocations = %d, want 0", r.TotalInvocations)
	}
	if r.AvgReduction != 0 {
		t.Errorf("AvgReduction = %f, want 0", r.AvgReduction)
	}
}

func TestFormatReport(t *testing.T) {
	entries := []Entry{
		{
			SessionID:    "s1",
			CommandType:  "grep",
			RawTokens:    1000,
			CleanTokens:  200,
			TokensSaved:  800,
			QualityScore: 100.0,
		},
	}
	r := GenerateReport(entries, "7d")
	output := FormatReport(r)

	if !strings.Contains(output, "7d") {
		t.Error("report should contain period")
	}
	if !strings.Contains(output, "grep") {
		t.Error("report should contain command type")
	}
	if !strings.Contains(output, "1000") {
		t.Error("report should contain raw tokens")
	}
}

func TestFormatReportJSON(t *testing.T) {
	entries := []Entry{
		{
			SessionID:    "s1",
			CommandType:  "grep",
			RawTokens:    1000,
			CleanTokens:  200,
			TokensSaved:  800,
			QualityScore: 100.0,
		},
	}
	r := GenerateReport(entries, "all")
	output := FormatReportJSON(r)

	var parsed Report
	if err := json.Unmarshal([]byte(output), &parsed); err != nil {
		t.Fatalf("FormatReportJSON produced invalid JSON: %v", err)
	}
	if parsed.TotalInvocations != 1 {
		t.Errorf("parsed TotalInvocations = %d, want 1", parsed.TotalInvocations)
	}
}

func TestFormatLast(t *testing.T) {
	entries := []Entry{
		{Timestamp: "2026-03-16T14:32:00Z", Command: "grep -rn foo .", CommandType: "grep", RawTokens: 1450, CleanTokens: 380, TokensSaved: 1070, ReductionPct: 73.8, QualityScore: 100.0},
		{Timestamp: "2026-03-16T14:33:00Z", Command: "find . -name '*.go'", CommandType: "find", RawTokens: 820, CleanTokens: 210, TokensSaved: 610, ReductionPct: 74.4, QualityScore: 100.0},
		{Timestamp: "2026-03-16T14:35:00Z", Command: "git log --oneline", CommandType: "git", RawTokens: 340, CleanTokens: 290, TokensSaved: 50, ReductionPct: 14.7, QualityScore: 100.0},
	}

	output := FormatLast(entries, 10)

	if !strings.Contains(output, "Last 3 invocations") {
		t.Error("should show actual count when fewer than N")
	}
	if !strings.Contains(output, "grep -rn foo .") {
		t.Error("should contain command name")
	}
	if !strings.Contains(output, "TOTAL") {
		t.Error("should contain totals row")
	}
	if !strings.Contains(output, "1730") {
		t.Errorf("should contain total saved (1070+610+50=1730)")
	}
}

func TestFormatLastLimit(t *testing.T) {
	entries := []Entry{
		{Timestamp: "2026-03-16T14:00:00Z", Command: "cmd1", RawTokens: 100, CleanTokens: 50, TokensSaved: 50, ReductionPct: 50.0},
		{Timestamp: "2026-03-16T14:01:00Z", Command: "cmd2", RawTokens: 200, CleanTokens: 80, TokensSaved: 120, ReductionPct: 60.0},
		{Timestamp: "2026-03-16T14:02:00Z", Command: "cmd3", RawTokens: 300, CleanTokens: 100, TokensSaved: 200, ReductionPct: 66.7},
	}

	output := FormatLast(entries, 2)

	if !strings.Contains(output, "Last 2 invocations") {
		t.Error("should limit to N entries")
	}
	if strings.Contains(output, "cmd1") {
		t.Error("should not contain oldest entry when limited")
	}
	if !strings.Contains(output, "cmd2") || !strings.Contains(output, "cmd3") {
		t.Error("should contain the 2 most recent entries")
	}
}

func TestFormatLastEmpty(t *testing.T) {
	output := FormatLast(nil, 10)
	if !strings.Contains(output, "No measurement entries") {
		t.Error("should handle empty entries")
	}
}

func TestReRequestPerCommandType(t *testing.T) {
	entries := []Entry{
		{CommandType: "grep", RawTokens: 100, CleanTokens: 20, TokensSaved: 80, ReRequest: false},
		{CommandType: "grep", RawTokens: 100, CleanTokens: 20, TokensSaved: 80, ReRequest: true, ReRequestType: "exact"},
		{CommandType: "grep", RawTokens: 100, CleanTokens: 20, TokensSaved: 80, ReRequest: true, ReRequestType: "escalation"},
		{CommandType: "find", RawTokens: 50, CleanTokens: 10, TokensSaved: 40, ReRequest: false},
	}

	r := GenerateReport(entries, "all")

	grep := r.ByCommandType["grep"]
	if grep.ReRequests != 2 {
		t.Errorf("grep ReRequests = %d, want 2", grep.ReRequests)
	}
	if grep.ReRequestRate < 66 || grep.ReRequestRate > 67 {
		t.Errorf("grep ReRequestRate = %.1f, want ~66.7", grep.ReRequestRate)
	}

	find := r.ByCommandType["find"]
	if find.ReRequests != 0 {
		t.Errorf("find ReRequests = %d, want 0", find.ReRequests)
	}
}

func TestInsights_HighReRequestRate(t *testing.T) {
	// 5 invocations of grep, 3 re-requests → 60% re-request rate
	var entries []Entry
	for i := 0; i < 5; i++ {
		e := Entry{CommandType: "grep", RawTokens: 100, CleanTokens: 20, TokensSaved: 80}
		if i >= 2 {
			e.ReRequest = true
			e.ReRequestType = "exact"
		}
		entries = append(entries, e)
	}

	r := GenerateReport(entries, "all")

	found := false
	for _, ins := range r.Insights {
		if ins.Command == "grep" && ins.Level == "warning" {
			found = true
			if !strings.Contains(ins.Message, "re-request rate") {
				t.Error("insight should mention re-request rate")
			}
			if !strings.Contains(ins.Message, "truncation") {
				t.Error("insight should suggest truncation adjustment")
			}
		}
	}
	if !found {
		t.Error("expected warning insight for grep high re-request rate")
	}
}

func TestInsights_Positive(t *testing.T) {
	// 15 invocations, no re-requests, good reduction
	var entries []Entry
	for i := 0; i < 15; i++ {
		entries = append(entries, Entry{
			CommandType: "grep", RawTokens: 100, CleanTokens: 30,
			TokensSaved: 70, ReRequest: false,
		})
	}

	r := GenerateReport(entries, "all")

	found := false
	for _, ins := range r.Insights {
		if ins.Level == "info" && strings.Contains(ins.Message, "excellent") {
			found = true
		}
	}
	if !found {
		t.Error("expected positive insight when no re-requests and good reduction")
	}
}

func TestInsights_OverallHighRate(t *testing.T) {
	// 10 invocations, 3 re-requests across different types → 30% overall
	var entries []Entry
	for i := 0; i < 10; i++ {
		e := Entry{CommandType: "generic", RawTokens: 100, CleanTokens: 50, TokensSaved: 50}
		if i < 3 {
			e.ReRequest = true
		}
		entries = append(entries, e)
	}

	r := GenerateReport(entries, "all")

	found := false
	for _, ins := range r.Insights {
		if ins.Command == "" && ins.Level == "warning" && strings.Contains(ins.Message, "Overall") {
			found = true
		}
	}
	if !found {
		t.Error("expected overall warning insight for high re-request rate")
	}
}

func TestFormatReport_WithInsights(t *testing.T) {
	// Generate data that triggers insights
	var entries []Entry
	for i := 0; i < 5; i++ {
		e := Entry{CommandType: "grep", RawTokens: 100, CleanTokens: 20, TokensSaved: 80}
		if i >= 2 {
			e.ReRequest = true
		}
		entries = append(entries, e)
	}

	r := GenerateReport(entries, "all")
	output := FormatReport(r)

	if !strings.Contains(output, "Quality Insights") {
		t.Error("report should contain Quality Insights section")
	}
	if !strings.Contains(output, "[!]") {
		t.Error("report should contain warning marker [!]")
	}
}

func TestFilterEntriesByPeriod(t *testing.T) {
	entries := []Entry{
		{Timestamp: "2020-01-01T00:00:00Z", Command: "old"},
		{Timestamp: "2099-12-31T00:00:00Z", Command: "future"},
	}

	filtered := FilterEntriesByPeriod(entries, "today")
	// Only the future entry should remain
	if len(filtered) != 1 {
		t.Fatalf("got %d entries, want 1", len(filtered))
	}
	if filtered[0].Command != "future" {
		t.Errorf("got command %q, want %q", filtered[0].Command, "future")
	}

	// "all" returns everything
	all := FilterEntriesByPeriod(entries, "all")
	if len(all) != 2 {
		t.Errorf("all: got %d, want 2", len(all))
	}
}
