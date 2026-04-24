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

	// Regression for #40: when a test-runner summary footer lands in the
	// omitted middle, it must be pinned so the LLM keeps authoritative counts.
	t.Run("vitest summary pinned when otherwise dropped", func(t *testing.T) {
		var b strings.Builder
		for i := 1; i <= 100; i++ {
			fmt.Fprintf(&b, "setup line %d\n", i)
		}
		b.WriteString(" Test Files  2 failed | 19 passed (21)\n")
		b.WriteString("      Tests  2 failed | 282 passed (284)\n")
		b.WriteString("   Duration  181.65s\n")
		for i := 1; i <= 100; i++ {
			fmt.Fprintf(&b, "failure detail line %d\n", i)
		}
		truncate := TruncateWithLimit(50)
		got := truncate(b.String())
		for _, want := range []string{
			"Test Files  2 failed | 19 passed (21)",
			"Tests  2 failed | 282 passed (284)",
			"Duration  181.65s",
			"summary pinned below",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("expected %q in truncated output", want)
			}
		}
	})

	t.Run("jest totals pinned when otherwise dropped", func(t *testing.T) {
		var b strings.Builder
		for i := 1; i <= 80; i++ {
			fmt.Fprintf(&b, "running test %d\n", i)
		}
		b.WriteString("Test Suites: 4 passed, 4 total\n")
		b.WriteString("Tests:       44 passed, 44 total\n")
		b.WriteString("Snapshots:   0 total\n")
		b.WriteString("Time:        3.5 s\n")
		for i := 1; i <= 80; i++ {
			fmt.Fprintf(&b, "tail noise %d\n", i)
		}
		got := TruncateWithLimit(50)(b.String())
		for _, want := range []string{
			"Test Suites: 4 passed, 4 total",
			"Tests:       44 passed, 44 total",
		} {
			if !strings.Contains(got, want) {
				t.Errorf("expected %q in truncated output", want)
			}
		}
	})

	t.Run("no pinning when summary already in head or tail", func(t *testing.T) {
		// Summary near the end lands naturally in the tail window — no pinning
		// marker should appear.
		var b strings.Builder
		for i := 1; i <= 100; i++ {
			fmt.Fprintf(&b, "line %d\n", i)
		}
		b.WriteString("Tests:       10 passed, 10 total\n")
		got := TruncateWithLimit(50)(b.String())
		if strings.Contains(got, "summary pinned below") {
			t.Error("unexpected pinning marker when summary is already in tail window")
		}
		if !strings.Contains(got, "Tests:       10 passed, 10 total") {
			t.Error("tail-window summary line was lost")
		}
	})
}
