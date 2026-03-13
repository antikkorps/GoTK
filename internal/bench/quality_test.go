package bench

import (
	"strings"
	"testing"

	"github.com/antikkorps/GoTK/internal/config"
)

func TestMeasureQualityReturnsAllFixtures(t *testing.T) {
	cfg := config.Default()
	report := MeasureQuality(cfg)

	fixtures := allFixtures()
	if len(report.Results) != len(fixtures) {
		t.Errorf("expected %d results, got %d", len(fixtures), len(report.Results))
	}

	for i, r := range report.Results {
		if r.Name != fixtures[i].name {
			t.Errorf("result %d: expected name %q, got %q", i, fixtures[i].name, r.Name)
		}
		if r.Score < 0 || r.Score > 100 {
			t.Errorf("result %q: score %.2f out of range [0, 100]", r.Name, r.Score)
		}
		if r.PreservedLines > r.ImportantLines {
			t.Errorf("result %q: preserved (%d) > important (%d)", r.Name, r.PreservedLines, r.ImportantLines)
		}
	}
}

func TestMeasureQualityAggregates(t *testing.T) {
	cfg := config.Default()
	report := MeasureQuality(cfg)

	sumImportant := 0
	sumPreserved := 0
	for _, r := range report.Results {
		sumImportant += r.ImportantLines
		sumPreserved += r.PreservedLines
	}

	if report.TotalImportant != sumImportant {
		t.Errorf("TotalImportant %d != sum %d", report.TotalImportant, sumImportant)
	}
	if report.TotalPreserved != sumPreserved {
		t.Errorf("TotalPreserved %d != sum %d", report.TotalPreserved, sumPreserved)
	}
	if report.OverallScore < 0 || report.OverallScore > 100 {
		t.Errorf("OverallScore %.2f out of range", report.OverallScore)
	}
	if report.AllFixturesPerfect != (report.TotalImportant == report.TotalPreserved) {
		t.Error("AllFixturesPerfect flag mismatch")
	}
}

func TestMeasureQualityNoImportantLinesYields100(t *testing.T) {
	// Fixtures with no warnings/errors should score 100%
	cfg := config.Default()
	report := MeasureQuality(cfg)

	for _, r := range report.Results {
		if r.ImportantLines == 0 && r.Score != 100 {
			t.Errorf("fixture %q has no important lines but score is %.2f, expected 100", r.Name, r.Score)
		}
	}
}

func TestMeasureQualityMissedLinesAreActuallyMissing(t *testing.T) {
	cfg := config.Default()
	report := MeasureQuality(cfg)

	for _, r := range report.Results {
		expectedMissed := r.ImportantLines - r.PreservedLines
		if len(r.MissedLines) != expectedMissed {
			t.Errorf("fixture %q: missed lines count %d != expected %d (important=%d, preserved=%d)",
				r.Name, len(r.MissedLines), expectedMissed, r.ImportantLines, r.PreservedLines)
		}
	}
}

func TestFormatQualityContainsAllFixtures(t *testing.T) {
	cfg := config.Default()
	report := MeasureQuality(cfg)
	output := FormatQuality(report)

	if !strings.Contains(output, "Quality Score Report") {
		t.Error("text output should contain header")
	}
	if !strings.Contains(output, "Total") {
		t.Error("text output should contain Total row")
	}
	for _, f := range allFixtures() {
		if !strings.Contains(output, f.name) {
			t.Errorf("text output should contain fixture name %q", f.name)
		}
	}
}

func TestFormatQualityJSON(t *testing.T) {
	cfg := config.Default()
	report := MeasureQuality(cfg)
	output := FormatQualityJSON(report)

	if !strings.HasPrefix(output, "{") {
		t.Error("JSON output should start with '{'")
	}
	for _, field := range []string{"total_important", "total_preserved", "overall_score", "all_perfect", "results"} {
		if !strings.Contains(output, "\""+field+"\"") {
			t.Errorf("JSON output should contain %q field", field)
		}
	}
}

func TestExtractCoreContent(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"src/file.go:22: TODO fix this", "TODO fix this"},
		{"main.go:1: error here", "error here"},
		{"no colons here", ""},
		{"one:colon", ""},
	}
	for _, tt := range tests {
		got := extractCoreContent(tt.input)
		if got != tt.want {
			t.Errorf("extractCoreContent(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
