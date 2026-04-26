package bench

import (
	"strings"
	"testing"
	"time"

	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/detect"
)

func TestAllFixturesGenerateNonEmptyData(t *testing.T) {
	fixtures := allFixtures()
	if len(fixtures) == 0 {
		t.Fatal("allFixtures returned no fixtures")
	}

	for _, f := range fixtures {
		t.Run(f.name, func(t *testing.T) {
			input := f.gen()
			if len(input) == 0 {
				t.Errorf("fixture %q generated empty data", f.name)
			}
			lines := strings.Split(input, "\n")
			// At least 10 non-empty lines for any realistic fixture
			nonEmpty := 0
			for _, l := range lines {
				if strings.TrimSpace(l) != "" {
					nonEmpty++
				}
			}
			if nonEmpty < 10 {
				t.Errorf("fixture %q generated only %d non-empty lines, expected at least 10", f.name, nonEmpty)
			}
		})
	}
}

func TestRunBenchmarksReturnsAllResults(t *testing.T) {
	cfg := config.Default()
	report := RunBenchmarks(cfg)

	fixtures := allFixtures()
	if len(report.Results) != len(fixtures) {
		t.Errorf("expected %d results, got %d", len(fixtures), len(report.Results))
	}

	for i, r := range report.Results {
		if r.Name != fixtures[i].name {
			t.Errorf("result %d: expected name %q, got %q", i, fixtures[i].name, r.Name)
		}
		if r.RawBytes <= 0 {
			t.Errorf("result %q: RawBytes should be positive, got %d", r.Name, r.RawBytes)
		}
		if r.CleanBytes <= 0 {
			t.Errorf("result %q: CleanBytes should be positive, got %d", r.Name, r.CleanBytes)
		}
		// Duration uses time.Now diffs, which has ~15ms resolution on
		// Windows — small fixtures legitimately measure 0. RawBytes /
		// LinesRaw already prove the benchmark ran; Duration < 0 is the
		// only real bug.
		if r.Duration < 0 {
			t.Errorf("result %q: Duration should be non-negative, got %v", r.Name, r.Duration)
		}
		if r.LinesRaw <= 0 {
			t.Errorf("result %q: LinesRaw should be positive, got %d", r.Name, r.LinesRaw)
		}
		if r.LinesClean <= 0 {
			t.Errorf("result %q: LinesClean should be positive, got %d", r.Name, r.LinesClean)
		}
	}

	if report.TotalRaw <= 0 {
		t.Error("TotalRaw should be positive")
	}
	if report.TotalClean <= 0 {
		t.Error("TotalClean should be positive")
	}
	if report.TotalDuration <= 0 {
		t.Error("TotalDuration should be positive")
	}
	// There should be some reduction overall
	if report.AvgReduction <= 0 {
		t.Errorf("AvgReduction should be positive (some filtering happened), got %.2f", report.AvgReduction)
	}
}

func TestRunSingleBenchmark(t *testing.T) {
	cfg := config.Default()
	input := generateGrepFixture()
	r := RunSingleBenchmark(cfg, "test-grep", input, detect.CmdGrep)

	if r.Name != "test-grep" {
		t.Errorf("expected name 'test-grep', got %q", r.Name)
	}
	if r.RawBytes != len(input) {
		t.Errorf("expected RawBytes %d, got %d", len(input), r.RawBytes)
	}
	if r.CleanBytes >= r.RawBytes {
		t.Errorf("expected CleanBytes < RawBytes (filtering should reduce), got clean=%d raw=%d", r.CleanBytes, r.RawBytes)
	}
	if r.Reduction <= 0 {
		t.Errorf("expected positive Reduction, got %.2f", r.Reduction)
	}
}

func TestMeasureFiltersReturnsContributions(t *testing.T) {
	cfg := config.Default()
	input := generateGrepFixture()
	contributions := MeasureFilters(cfg, input, detect.CmdGrep)

	if len(contributions) == 0 {
		t.Fatal("MeasureFilters returned no contributions")
	}

	// Should have at least the generic filters (StripANSI, NormalizeWhitespace, Dedup)
	// plus command-specific ones plus TrimEmpty, CompressStackTraces, RedactSecrets, Summarize, Truncate
	if len(contributions) < 5 {
		t.Errorf("expected at least 5 filter contributions, got %d", len(contributions))
	}

	for _, c := range contributions {
		if c.FilterName == "" {
			t.Error("FilterName should not be empty")
		}
		if c.BytesBefore < 0 {
			t.Errorf("filter %q: BytesBefore should be non-negative, got %d", c.FilterName, c.BytesBefore)
		}
		if c.BytesAfter < 0 {
			t.Errorf("filter %q: BytesAfter should be non-negative, got %d", c.FilterName, c.BytesAfter)
		}
		if c.Duration < 0 {
			t.Errorf("filter %q: Duration should be non-negative, got %v", c.FilterName, c.Duration)
		}
	}

	// The first filter should have BytesBefore equal to the input size
	if contributions[0].BytesBefore != len(input) {
		t.Errorf("first filter BytesBefore should equal input length %d, got %d",
			len(input), contributions[0].BytesBefore)
	}
}

func TestMeasureLatencyReturnsValidStats(t *testing.T) {
	cfg := config.Default()
	input := generateGrepFixture()
	iterations := 10
	report := MeasureLatency(cfg, input, detect.CmdGrep, iterations)

	if report.Iterations != iterations {
		t.Errorf("expected %d iterations, got %d", iterations, report.Iterations)
	}
	if report.InputBytes != len(input) {
		t.Errorf("expected InputBytes %d, got %d", len(input), report.InputBytes)
	}
	if report.Min <= 0 {
		t.Errorf("Min should be positive, got %v", report.Min)
	}
	if report.Max < report.Min {
		t.Errorf("Max (%v) should be >= Min (%v)", report.Max, report.Min)
	}
	if report.Mean < report.Min || report.Mean > report.Max {
		t.Errorf("Mean (%v) should be between Min (%v) and Max (%v)", report.Mean, report.Min, report.Max)
	}
	if report.P50 < report.Min || report.P50 > report.Max {
		t.Errorf("P50 (%v) should be between Min (%v) and Max (%v)", report.P50, report.Min, report.Max)
	}
	if report.P95 < report.P50 {
		t.Errorf("P95 (%v) should be >= P50 (%v)", report.P95, report.P50)
	}

	// Performance target: processing should take less than 10ms per iteration for this input
	if report.P99 > 10*time.Millisecond {
		t.Logf("WARNING: P99 latency (%v) exceeds 10ms target", report.P99)
	}
}

func TestFormatReportJSON(t *testing.T) {
	cfg := config.Default()
	report := RunBenchmarks(cfg)
	jsonOutput := FormatReportJSON(report)

	if !strings.HasPrefix(jsonOutput, "{") {
		t.Error("JSON output should start with '{'")
	}
	if !strings.Contains(jsonOutput, "\"total_raw\"") {
		t.Error("JSON output should contain 'total_raw' field")
	}
	if !strings.Contains(jsonOutput, "\"results\"") {
		t.Error("JSON output should contain 'results' field")
	}
	if !strings.Contains(jsonOutput, "\"reduction\"") {
		t.Error("JSON output should contain 'reduction' field in results")
	}

	// Check all fixture names appear
	fixtures := allFixtures()
	for _, f := range fixtures {
		if !strings.Contains(jsonOutput, f.name) {
			t.Errorf("JSON output should contain fixture name %q", f.name)
		}
	}
}

func TestFormatPerFilterJSON(t *testing.T) {
	cfg := config.Default()
	input := generateGrepFixture()
	contributions := MeasureFilters(cfg, input, detect.CmdGrep)
	jsonOutput := FormatPerFilterJSON("grep large", contributions)

	if !strings.HasPrefix(jsonOutput, "{") {
		t.Error("JSON output should start with '{'")
	}
	if !strings.Contains(jsonOutput, "\"fixture\"") {
		t.Error("JSON output should contain 'fixture' field")
	}
	if !strings.Contains(jsonOutput, "\"filters\"") {
		t.Error("JSON output should contain 'filters' field")
	}
	if !strings.Contains(jsonOutput, "grep large") {
		t.Error("JSON output should contain fixture name")
	}
}

func TestFormatLatencyJSON(t *testing.T) {
	cfg := config.Default()
	input := generateGrepFixture()
	report := MeasureLatency(cfg, input, detect.CmdGrep, 5)
	jsonOutput := FormatLatencyJSON(report)

	if !strings.HasPrefix(jsonOutput, "{") {
		t.Error("JSON output should start with '{'")
	}
	for _, field := range []string{"iterations", "input_bytes", "min_us", "max_us", "mean_us", "p50_us", "p95_us", "p99_us"} {
		if !strings.Contains(jsonOutput, "\""+field+"\"") {
			t.Errorf("JSON output should contain %q field", field)
		}
	}
}

func TestFormatReport(t *testing.T) {
	cfg := config.Default()
	report := RunBenchmarks(cfg)
	output := FormatReport(report)

	if !strings.Contains(output, "GoTK Benchmark Report") {
		t.Error("text output should contain header")
	}
	if !strings.Contains(output, "Total") {
		t.Error("text output should contain Total row")
	}
	// Check all fixture names appear
	fixtures := allFixtures()
	for _, f := range fixtures {
		if !strings.Contains(output, f.name) {
			t.Errorf("text output should contain fixture name %q", f.name)
		}
	}
}

func TestFormatPerFilter(t *testing.T) {
	cfg := config.Default()
	input := generateGrepFixture()
	contributions := MeasureFilters(cfg, input, detect.CmdGrep)
	output := FormatPerFilter("grep large", contributions)

	if !strings.Contains(output, "Per-Filter Contribution") {
		t.Error("text output should contain header")
	}
	if !strings.Contains(output, "grep large") {
		t.Error("text output should contain fixture name")
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{999, "999"},
		{1000, "1,000"},
		{12345, "12,345"},
		{1234567, "1,234,567"},
	}

	for _, tt := range tests {
		result := formatBytes(tt.input)
		if result != tt.expected {
			t.Errorf("formatBytes(%d) = %q, want %q", tt.input, result, tt.expected)
		}
	}
}
