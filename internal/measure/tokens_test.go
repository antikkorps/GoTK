package measure

import "testing"

func TestEstimateTokens_Empty(t *testing.T) {
	if got := EstimateTokens(""); got != 0 {
		t.Errorf("EstimateTokens(\"\") = %d, want 0", got)
	}
}

func TestEstimateTokens_Short(t *testing.T) {
	got := EstimateTokens("hello world")
	if got < 2 {
		t.Errorf("EstimateTokens(\"hello world\") = %d, want >= 2", got)
	}
}

func TestEstimateTokens_Code(t *testing.T) {
	code := "func main() {\n\tfmt.Println(\"hello\")\n}"
	got := EstimateTokens(code)
	if got < 5 {
		t.Errorf("EstimateTokens(code) = %d, want >= 5", got)
	}
}

func TestEstimateTokens_FloorApplied(t *testing.T) {
	// A long string with few whitespace-separated words should hit the floor
	input := "abcdefghijklmnopqrstuvwxyz"
	got := EstimateTokens(input)
	floor := len(input) / 4
	if got < floor {
		t.Errorf("EstimateTokens = %d, want >= floor %d", got, floor)
	}
}

func TestEstimateTokens_LargeOutput(t *testing.T) {
	// Simulate terminal output
	var lines string
	for i := 0; i < 100; i++ {
		lines += "src/internal/filter/ansi.go:42: func StripANSI(s string) string {\n"
	}
	got := EstimateTokens(lines)
	if got < 500 {
		t.Errorf("EstimateTokens(100 lines) = %d, want >= 500", got)
	}
}
