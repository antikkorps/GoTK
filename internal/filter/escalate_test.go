package filter

import (
	"fmt"
	"strings"
	"testing"
)

// buildWithFailure generates `total` lines of filler with a multi-line
// failure block inserted at position `failAt`.
func buildWithFailure(total, failAt int) string {
	lines := make([]string, 0, total+6)
	for i := 0; i < failAt; i++ {
		lines = append(lines, fmt.Sprintf("info line %d", i))
	}
	lines = append(lines,
		"  ● UserService › should create user",
		"    TypeError: Cannot read property 'id' of undefined",
		"      at Object.<anonymous> (src/user.test.js:42:5)",
		"      at Module._compile (internal/modules/cjs/loader.js:999:30)",
	)
	for i := failAt; i < total; i++ {
		lines = append(lines, fmt.Sprintf("info line %d", i))
	}
	return strings.Join(lines, "\n") + "\n"
}

func TestTruncateWithEscalationNoFailure(t *testing.T) {
	// No anchor → behaves exactly like TruncateWithLimit head+tail.
	input := genLines(500, "info line")
	window := TruncateWithEscalation(50, EscalateWindow, 10)(input)
	legacy := TruncateWithLimit(50)(input)
	if window != legacy {
		t.Errorf("no-failure escalation should match legacy truncate output")
	}
}

func TestTruncateWithEscalationOffAlwaysLegacy(t *testing.T) {
	// Off mode must match TruncateWithLimit even when failures are present.
	input := buildWithFailure(500, 250)
	off := TruncateWithEscalation(50, EscalateOff, 10)(input)
	legacy := TruncateWithLimit(50)(input)
	if off != legacy {
		t.Errorf("EscalateOff should match TruncateWithLimit exactly")
	}
}

func TestTruncateWithEscalationWindowPreservesFailure(t *testing.T) {
	// Failure at line 250 of 500 with maxLines=50 would normally be dropped
	// into the omitted middle. window mode must preserve it.
	input := buildWithFailure(500, 250)

	got := TruncateWithEscalation(50, EscalateWindow, 10)(input)

	wantLines := []string{
		"● UserService › should create user",
		"TypeError: Cannot read property 'id' of undefined",
		"at Object.<anonymous> (src/user.test.js:42:5)",
	}
	for _, want := range wantLines {
		if !strings.Contains(got, want) {
			t.Errorf("window escalation dropped failure context: missing %q", want)
		}
	}

	// Must still produce omission markers for the dropped regions.
	if !strings.Contains(got, "lines omitted") {
		t.Errorf("window escalation should still omit non-failure middle content")
	}
}

func TestTruncateWithEscalationWindowKeepsHeadAndTail(t *testing.T) {
	input := buildWithFailure(500, 250)
	got := TruncateWithEscalation(50, EscalateWindow, 5)(input)

	// Head and tail must still be present.
	if !strings.Contains(got, "info line 0") {
		t.Errorf("head lost in window mode")
	}
	if !strings.Contains(got, "info line 499") {
		t.Errorf("tail lost in window mode")
	}
}

func TestTruncateWithEscalationHintAddsFooter(t *testing.T) {
	input := buildWithFailure(500, 250)
	got := TruncateWithEscalation(50, EscalateHint, 10)(input)

	if !strings.Contains(got, "failure signals detected") {
		t.Errorf("hint mode should append a failure-signals footer")
	}
	// Body is the legacy head+tail — confirm the failure detail is NOT included
	// (hint mode explicitly trades keeping failure context for a smaller output).
	if strings.Contains(got, "TypeError: Cannot read property") {
		t.Errorf("hint mode should keep legacy head+tail body, not inject failure window")
	}
}

func TestTruncateWithEscalationHintSkipsFooterWhenNoFailure(t *testing.T) {
	input := genLines(500, "info line")
	got := TruncateWithEscalation(50, EscalateHint, 10)(input)

	if strings.Contains(got, "failure signals detected") {
		t.Errorf("hint mode footer must only appear when failure anchors are present")
	}
}

func TestTruncateWithEscalationConservativeKeepsFullOutput(t *testing.T) {
	input := buildWithFailure(500, 250)
	got := TruncateWithEscalation(50, EscalateConservative, 10)(input)

	if got != input {
		t.Errorf("conservative mode should keep the full input when failures are detected")
	}
}

func TestTruncateWithEscalationShortInputUnchanged(t *testing.T) {
	// Below maxLines threshold → pass-through regardless of mode.
	input := genLines(20, "info line")
	for _, mode := range []AutoEscalateMode{EscalateOff, EscalateHint, EscalateWindow, EscalateConservative} {
		got := TruncateWithEscalation(50, mode, 10)(input)
		if got != input {
			t.Errorf("mode %q: short input should be unchanged", mode)
		}
	}
}

func TestTruncateWithEscalationManyAnchorsRespectsCap(t *testing.T) {
	// 500 lines with 50 failure anchors spread across the middle. With a ±10
	// window and maxLines=30, the naive kept-set would balloon well past the
	// 2×maxLines cap; the fallback must shrink to head + tail + first + last.
	var b strings.Builder
	for i := 0; i < 500; i++ {
		if i > 50 && i < 450 && i%8 == 0 {
			b.WriteString("  ● Test failure marker\n")
		} else {
			fmt.Fprintf(&b, "info line %d\n", i)
		}
	}
	got := TruncateWithEscalation(30, EscalateWindow, 10)(b.String())

	// Without a cap, ±10 windows around 50 anchors would keep ~400 lines of the
	// 500-line input. The cap fallback must keep that well under control.
	kept := strings.Count(got, "\n")
	if kept > 100 {
		t.Errorf("window mode exceeded cap: kept %d lines, expected ≤ 100", kept)
	}
	if !strings.Contains(got, "lines omitted") {
		t.Errorf("expected omission markers with many anchors and cap fallback")
	}
}

// Regression for #40 via the escalate path: a vitest run with failures that
// also prints a summary footer (Test Files / Tests / Duration) must keep both
// the failure window AND the summary footer lines, even when the footer lands
// in the omitted middle.
func TestTruncateWithEscalationWindowPinsSummary(t *testing.T) {
	var b strings.Builder
	for i := 0; i < 60; i++ {
		fmt.Fprintf(&b, "setup line %d\n", i)
	}
	b.WriteString(" Test Files  2 failed | 19 passed (21)\n")
	b.WriteString("      Tests  2 failed | 282 passed (284)\n")
	b.WriteString("   Duration  181.65s\n")
	for i := 0; i < 30; i++ {
		fmt.Fprintf(&b, "more stuff %d\n", i)
	}
	b.WriteString(" ❯ FAIL src/foo.test.ts > scenario\n")
	b.WriteString("   TypeError: boom\n")
	b.WriteString("     at run (src/foo.ts:10:5)\n")
	for i := 0; i < 20; i++ {
		fmt.Fprintf(&b, "trailer %d\n", i)
	}

	got := TruncateWithEscalation(50, EscalateWindow, 5)(b.String())
	for _, want := range []string{
		"Test Files  2 failed | 19 passed (21)",
		"Tests  2 failed | 282 passed (284)",
		"TypeError: boom", // failure window must also survive
	} {
		if !strings.Contains(got, want) {
			t.Errorf("expected %q in escalate-window output", want)
		}
	}
}

func TestParseAutoEscalate(t *testing.T) {
	tests := []struct {
		in   string
		want AutoEscalateMode
	}{
		{"off", EscalateOff},
		{"none", EscalateOff},
		{"hint", EscalateHint},
		{"window", EscalateWindow},
		{"conservative", EscalateConservative},
		{"full", EscalateConservative},
		{"", EscalateWindow},
		{"garbage", EscalateWindow},
		{"WINDOW", EscalateWindow},
	}
	for _, tt := range tests {
		if got := ParseAutoEscalate(tt.in); got != tt.want {
			t.Errorf("ParseAutoEscalate(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestFindFailureAnchors(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		isAnchor bool
	}{
		{"jest FAIL suite prefix", "FAIL tests/foo.test.js", true},
		{"go test --- FAIL", "--- FAIL: TestFoo", true},
		{"jest bullet marker", "  ● UserService › should create user", true},
		{"N failed count > 0", "Tests:       3 failed, 1617 passed, 1620 total", true},
		{"0 failed is not an anchor", "Tests:       0 failed, 1620 passed, 1620 total", false},
		{"Error: banner", "Error: connection refused", true},
		{"TypeError exception", "TypeError: Cannot read property 'x' of undefined", true},
		{"go panic", "panic: runtime error: index out of range", true},
		{"fatal prefix", "fatal: not a git repository", true},
		{"python traceback", "Traceback (most recent call last):", true},
		{"typescript error", "src/foo.ts(42,5): error TS2322: Type 'string' is not assignable", true},
		{"rust error code", "error[E0308]: mismatched types", true},
		{"BUILD FAILED", "BUILD FAILED", true},
		{"github actions error", "##[error]Process completed with exit code 1", true},
		{"info line not anchor", "info line 42", false},
		{"word 'fail' in text not anchor", "should not fail when the input is empty", false},
		{"word 'FAIL' mid-line not anchor", "description: should FAIL on invalid input", false},
		{"jest console trailer not anchor", "      at log (middlewares/jwt.auth.js:71:13)", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isFailureAnchor(tt.line); got != tt.isAnchor {
				t.Errorf("isFailureAnchor(%q) = %v, want %v", tt.line, got, tt.isAnchor)
			}
		})
	}
}
