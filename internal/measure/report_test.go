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
