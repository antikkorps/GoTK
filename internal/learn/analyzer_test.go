package learn

import (
	"testing"

	"github.com/antikkorps/GoTK/internal/classify"
)

func TestAnalyzeNotReady_NoData(t *testing.T) {
	result := Analyze(nil, DefaultAnalyzerConfig())
	if !result.NotReady {
		t.Error("expected NotReady with nil observations")
	}
}

func TestAnalyzeNotReady_FewSessions(t *testing.T) {
	obs := []Observation{
		{SessionID: "s1", Normalized: "line", Level: int(classify.Noise)},
		{SessionID: "s2", Normalized: "line", Level: int(classify.Noise)},
	}
	result := Analyze(obs, DefaultAnalyzerConfig())
	if !result.NotReady {
		t.Error("expected NotReady with only 2 sessions (default min is 3)")
	}
}

func TestAnalyzeFindsNoisePatterns(t *testing.T) {
	// Create observations with a recurring noise pattern across 3 sessions
	var obs []Observation
	for _, sid := range []string{"s1", "s2", "s3"} {
		// Noise pattern: appears frequently
		for i := 0; i < 10; i++ {
			obs = append(obs, Observation{
				SessionID:  sid,
				Normalized: "Compiling serde <VERSION>",
				Level:      int(classify.Noise),
				Command:    "cargo build",
			})
		}
		// Info lines: enough to keep noise pattern under 30% breadth
		for i := 0; i < 40; i++ {
			obs = append(obs, Observation{
				SessionID:  sid,
				Normalized: "info line " + sid,
				Level:      int(classify.Info),
				Command:    "cargo build",
			})
		}
	}

	acfg := DefaultAnalyzerConfig()
	result := Analyze(obs, acfg)

	if result.NotReady {
		t.Fatalf("expected ready, got NotReady: %s", result.NotReadyMsg)
	}

	if len(result.Candidates) == 0 {
		t.Fatal("expected at least one candidate pattern")
	}

	// The noise pattern should be suggested
	found := false
	for _, c := range result.Candidates {
		if c.NoiseScore == 1.0 {
			found = true
			if c.Frequency < 0.05 {
				t.Errorf("noise pattern frequency too low: %.2f", c.Frequency)
			}
		}
	}
	if !found {
		t.Error("expected to find a candidate with 100% noise score")
	}
}

func TestAnalyzeRejectsWarningPatterns(t *testing.T) {
	var obs []Observation
	for _, sid := range []string{"s1", "s2", "s3"} {
		// Pattern that appears in both noise and warning lines
		for i := 0; i < 8; i++ {
			obs = append(obs, Observation{
				SessionID:  sid,
				Normalized: "deprecated: use NewAPI instead",
				Level:      int(classify.Noise),
				Command:    "make",
			})
		}
		// Same normalized pattern but classified as Warning
		obs = append(obs, Observation{
			SessionID:  sid,
			Normalized: "deprecated: use NewAPI instead",
			Level:      int(classify.Warning),
			Command:    "make",
		})
		// Filler
		for i := 0; i < 30; i++ {
			obs = append(obs, Observation{
				SessionID:  sid,
				Normalized: "filler line " + sid,
				Level:      int(classify.Info),
				Command:    "make",
			})
		}
	}

	result := Analyze(obs, DefaultAnalyzerConfig())

	// The pattern that has Warning-level lines should NOT be suggested
	for _, c := range result.Candidates {
		if c.Description == "deprecated: use NewAPI instead" {
			t.Errorf("pattern with Warning-level matches should not be suggested")
		}
	}
}

func TestAnalyzeRejectsTooWide(t *testing.T) {
	var obs []Observation
	for _, sid := range []string{"s1", "s2", "s3"} {
		// One pattern that matches >30% of all lines
		for i := 0; i < 40; i++ {
			obs = append(obs, Observation{
				SessionID:  sid,
				Normalized: "noise everywhere",
				Level:      int(classify.Noise),
				Command:    "cmd",
			})
		}
		for i := 0; i < 60; i++ {
			obs = append(obs, Observation{
				SessionID:  sid,
				Normalized: "info " + sid,
				Level:      int(classify.Info),
				Command:    "cmd",
			})
		}
	}

	result := Analyze(obs, DefaultAnalyzerConfig())

	for _, c := range result.Candidates {
		if c.Frequency > 0.30 {
			t.Errorf("candidate with frequency %.2f should not be suggested (max 0.30)", c.Frequency)
		}
	}
}

func TestFormatSuggestions(t *testing.T) {
	r := &AnalysisResult{
		TotalLines: 100,
		Sessions:   5,
		Candidates: []Candidate{
			{
				Pattern:     `^\s*Compiling\s`,
				Description: "Compiling serde <VERSION>",
				Frequency:   0.23,
				NoiseScore:  0.95,
				MatchCount:  23,
				Samples:     []string{"Compiling serde v1.0.152"},
				Commands:    []string{"cargo build"},
			},
		},
	}

	output := FormatSuggestions(r)
	if output == "" {
		t.Error("FormatSuggestions returned empty string")
	}

	// Should contain the pattern
	if !containsStr(output, "Compiling") {
		t.Error("output should mention the pattern")
	}
	// Should contain TOML snippet
	if !containsStr(output, "always_remove") {
		t.Error("output should contain always_remove TOML snippet")
	}
}

func TestFormatSuggestionsNotReady(t *testing.T) {
	r := &AnalysisResult{
		NotReady:    true,
		NotReadyMsg: "Need more sessions.",
	}
	output := FormatSuggestions(r)
	if !containsStr(output, "Need more sessions") {
		t.Error("NotReady output should contain the message")
	}
}

func containsStr(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 && contains(s, substr)
}

func contains(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
