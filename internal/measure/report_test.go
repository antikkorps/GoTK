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
