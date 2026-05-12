package measure

import "testing"

func TestSummaryMarkerCredit(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want int
	}{
		{"empty", "", 0},
		{"no markers", "some output\nno summary here\n", 0},
		{"npm summary", "npm warn deprecated foo\n... and 14 more npm warnings\n", 14},
		{"node deprecations", "(node:1) [DEP0001] DeprecationWarning: ...\n... and 7 more deprecation warnings\n", 7},
		{"node experimental", "(5 experimental warnings)\n", 5},
		{"identical workers (singular)", "... and 1 identical warning from other workers\n", 1},
		{"identical workers (plural)", "... and 3 identical warnings from other workers\n", 3},
		{"stack identical", "[... 4 more errors with identical stack]\n", 4},
		{"multiple markers sum", "... and 14 more npm warnings\n(5 experimental warnings)\n", 19},
		{"non-numeric ignored", "... and many more npm warnings\n", 0},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := SummaryMarkerCredit(tc.in); got != tc.want {
				t.Errorf("SummaryMarkerCredit(%q) = %d, want %d", tc.in, got, tc.want)
			}
		})
	}
}

func TestIsDropSafeNoise(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{"Warning: Permanently added 'host.example.com' (ED25519) to the list of known hosts.", true},
		{"  Warning: Permanently added 'h' (RSA) to the list of known hosts.", true},
		{"Warning: deprecated API endpoint used", false},
		{"npm warn deprecated foo@1.0", false},
		{"", false},
	}
	for _, tc := range cases {
		if got := IsDropSafeNoise(tc.line); got != tc.want {
			t.Errorf("IsDropSafeNoise(%q) = %v, want %v", tc.line, got, tc.want)
		}
	}
}

func TestComputeQualityScore_SummaryMarkerCredit(t *testing.T) {
	// 15 warnings in raw, only 1 kept + a summary marker for the other 14.
	var raw string
	for i := 0; i < 15; i++ {
		raw += "npm warn deprecated pkg-X: not supported\n"
	}
	clean := "npm warn deprecated pkg-X: not supported\n... and 14 more npm warnings\n"

	score, important := ComputeQualityScore(raw, clean)
	if important < 1 {
		t.Fatalf("expected classifier to flag at least one important line, got %d", important)
	}
	if score < 95 {
		t.Errorf("expected score >= 95 with summary credit, got %.1f (important=%d)", score, important)
	}
}

func TestComputeQualityScore_DropSafeBanner(t *testing.T) {
	raw := "Warning: Permanently added 'prod-server' (ED25519) to the list of known hosts.\n" +
		"command output line 1\n"
	clean := "command output line 1\n"

	score, important := ComputeQualityScore(raw, clean)
	if important < 1 {
		t.Fatalf("expected classifier to flag the banner, got %d", important)
	}
	if score != 100 {
		t.Errorf("expected 100%% (drop-safe banner credited), got %.1f", score)
	}
}

func TestComputeQualityScore_PartialCreditCap(t *testing.T) {
	// 3 warnings dropped, marker claims 99 — credit must be capped at 3.
	raw := "npm warn deprecated a\nnpm warn deprecated b\nnpm warn deprecated c\n"
	clean := "... and 99 more npm warnings\n"
	score, important := ComputeQualityScore(raw, clean)
	if important < 1 {
		t.Fatalf("expected important > 0")
	}
	if score != 100 {
		t.Errorf("expected 100%% (credit capped at dropped), got %.1f", score)
	}
}

func TestComputeQualityScore_NoMarkerNoCredit(t *testing.T) {
	raw := "npm warn deprecated a\nnpm warn deprecated b\n"
	clean := "(no summary)\n"
	score, _ := ComputeQualityScore(raw, clean)
	if score != 0 {
		t.Errorf("expected 0%% (no marker, lines truly lost), got %.1f", score)
	}
}
