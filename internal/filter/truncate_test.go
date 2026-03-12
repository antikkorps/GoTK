package filter

import (
	"fmt"
	"strings"
	"testing"
)

func TestTruncateWithLimit(t *testing.T) {
	// Helper to generate N lines
	makeLines := func(n int) string {
		var lines []string
		for i := 1; i <= n; i++ {
			lines = append(lines, fmt.Sprintf("line %d", i))
		}
		return strings.Join(lines, "\n") + "\n"
	}

	t.Run("under limit no-op", func(t *testing.T) {
		truncate := TruncateWithLimit(50)
		input := makeLines(10)
		got := truncate(input)
		if got != input {
			t.Errorf("expected no truncation for 10 lines with maxLines=50")
		}
	})

	t.Run("exactly at limit no-op", func(t *testing.T) {
		truncate := TruncateWithLimit(10)
		input := makeLines(10)
		got := truncate(input)
		if got != input {
			t.Errorf("expected no truncation for 10 lines with maxLines=10")
		}
	})

	t.Run("over limit with head tail split", func(t *testing.T) {
		truncate := TruncateWithLimit(10)
		input := makeLines(20)
		got := truncate(input)

		// Should contain head lines
		if !strings.Contains(got, "line 1") {
			t.Error("expected head to contain line 1")
		}

		// Should contain tail lines
		if !strings.Contains(got, "line 20") {
			t.Error("expected tail to contain line 20")
		}

		// Should contain omitted marker
		if !strings.Contains(got, "lines omitted") {
			t.Error("expected omitted lines marker")
		}
	})

	t.Run("omitted count accuracy", func(t *testing.T) {
		truncate := TruncateWithLimit(10)
		input := makeLines(100)
		got := truncate(input)

		headCount := int(float64(10) * HeadTailRatio) // 7
		tailCount := 10 - headCount                   // 3
		omitted := 100 - headCount - tailCount        // 90

		marker := fmt.Sprintf("[... %d lines omitted ...]", omitted)
		if !strings.Contains(got, marker) {
			t.Errorf("expected marker %q in output, got:\n%s", marker, got)
		}
	})

	t.Run("maxLines 0 disables truncation", func(t *testing.T) {
		truncate := TruncateWithLimit(0)
		input := makeLines(1000)
		got := truncate(input)
		if got != input {
			t.Error("expected no truncation when maxLines=0")
		}
	})

	t.Run("negative maxLines disables truncation", func(t *testing.T) {
		truncate := TruncateWithLimit(-1)
		input := makeLines(100)
		got := truncate(input)
		if got != input {
			t.Error("expected no truncation when maxLines=-1")
		}
	})
}
