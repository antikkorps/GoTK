package detect

import (
	"strings"
	"testing"
)

func TestStripJestConsoleBlocks_BasicBlock(t *testing.T) {
	input := strings.Join([]string{
		"PASS src/auth.test.ts",
		"  console.log",
		"    Email sent: OK",
		"      at src/utils/nodemailer.ts:262:13",
		"",
		"Test Suites: 1 passed",
	}, "\n")

	got := stripJestConsoleBlocks(input)

	if strings.Contains(got, "console.log") {
		t.Errorf("console.log header should be stripped, got:\n%s", got)
	}
	if strings.Contains(got, "at src/utils/nodemailer.ts:262:13") {
		t.Errorf("at-trailer should be stripped, got:\n%s", got)
	}
	if !strings.Contains(got, "Email sent: OK") {
		t.Errorf("message content must be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "Test Suites: 1 passed") {
		t.Errorf("summary must be preserved, got:\n%s", got)
	}
}

func TestStripJestConsoleBlocks_MultipleBlocks(t *testing.T) {
	input := strings.Join([]string{
		"  console.log",
		"    first message",
		"      at a/b.ts:1:1",
		"",
		"  console.warn",
		"    second message",
		"      at c/d.ts:2:2",
		"",
		"Tests: 2 passed",
	}, "\n")

	got := stripJestConsoleBlocks(input)
	if strings.Count(got, "console.") != 0 {
		t.Errorf("all console headers should be stripped, got:\n%s", got)
	}
	for _, msg := range []string{"first message", "second message", "Tests: 2 passed"} {
		if !strings.Contains(got, msg) {
			t.Errorf("missing %q in:\n%s", msg, got)
		}
	}
}

func TestStripJestConsoleBlocks_MultilineMessage(t *testing.T) {
	input := strings.Join([]string{
		"  console.log",
		"    line one of the message",
		"    line two of the message",
		"    line three",
		"      at src/foo.ts:10:5",
		"",
		"next block",
	}, "\n")

	got := stripJestConsoleBlocks(input)
	for _, msg := range []string{"line one", "line two", "line three", "next block"} {
		if !strings.Contains(got, msg) {
			t.Errorf("missing %q after strip, got:\n%s", msg, got)
		}
	}
	if strings.Contains(got, "at src/foo.ts") {
		t.Errorf("trailer should be stripped")
	}
}

func TestStripJestConsoleBlocks_PreservesRealStackTrace(t *testing.T) {
	// Real error stack traces use `at <funcName> (<path>:N:N)` — parens distinguish
	// them from Jest's bare `at <path>:N:N` trailer. They must pass through intact.
	input := strings.Join([]string{
		"  console.log",
		"    throwing...",
		"      at Object.<anonymous> (src/foo.ts:10:5)",
		"      at Module._compile (node:internal/modules/cjs/loader:1256:14)",
		"",
		"Error: something broke",
	}, "\n")

	got := stripJestConsoleBlocks(input)
	// Header stays because we never found a bare `at <path>:N:N` trailer.
	if !strings.Contains(got, "console.log") {
		t.Errorf("should leave block alone when trailer doesn't match strict pattern, got:\n%s", got)
	}
	if !strings.Contains(got, "at Object.<anonymous>") {
		t.Errorf("real stack frames must pass through, got:\n%s", got)
	}
}

func TestStripJestConsoleBlocks_NoMatchingTrailer(t *testing.T) {
	// A `console.log` header with no matching `at` trailer within the look-ahead
	// window must leave the content unchanged.
	input := strings.Join([]string{
		"  console.log",
		"    some message",
		"",
		"next line",
	}, "\n")

	got := stripJestConsoleBlocks(input)
	if !strings.Contains(got, "console.log") {
		t.Errorf("should preserve header when no trailer follows, got:\n%s", got)
	}
	if !strings.Contains(got, "some message") {
		t.Errorf("message should be preserved, got:\n%s", got)
	}
}

func TestStripJestConsoleBlocks_NonIndentedBreak(t *testing.T) {
	// Unindented content between header and would-be trailer aborts the match.
	input := strings.Join([]string{
		"  console.log",
		"    first",
		"UNINDENTED INTERRUPTION",
		"      at a/b.ts:1:1",
	}, "\n")

	got := stripJestConsoleBlocks(input)
	if !strings.Contains(got, "console.log") {
		t.Errorf("non-indented break should abort strip, got:\n%s", got)
	}
	if !strings.Contains(got, "UNINDENTED INTERRUPTION") {
		t.Errorf("interrupting line must be preserved, got:\n%s", got)
	}
}

func TestStripJestConsoleBlocks_AllMethods(t *testing.T) {
	for _, method := range []string{"log", "warn", "error", "info", "debug"} {
		input := "  console." + method + "\n    msg-for-" + method + "\n      at x.ts:1:1\n"
		got := stripJestConsoleBlocks(input)
		if strings.Contains(got, "console."+method) {
			t.Errorf("console.%s not stripped, got:\n%s", method, got)
		}
		if !strings.Contains(got, "msg-for-"+method) {
			t.Errorf("message for console.%s lost, got:\n%s", method, got)
		}
	}
}

// Generic `(node:PID) Warning:` collapse is owned by filter.CollapseNodeWarnings;
// see internal/filter/nodewarn_test.go for coverage of the consecutive-block
// case, single-warning passthrough, and signature mismatch behavior.

// TestStripJestConsoleBlocks_NamedTrailers covers issue #79: Jest names the
// calling frame for almost every application log, and the parenthesised form
// was not recognised as a trailer — so every triplet survived the filter.
func TestStripJestConsoleBlocks_NamedTrailers(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "named parenthesised trailer",
			input: "  console.log\n    JWT AUTH ok for testuser\n      at log (middlewares/jwt.auth.js:85:13)\n",
			want:  "    JWT AUTH ok for testuser",
		},
		{
			name:  "Object-qualified trailer",
			input: "  console.log\n    Jest setup loaded\n      at Object.log (tests/setup.js:188:9)\n",
			want:  "    Jest setup loaded",
		},
		{
			name:  "anonymous trailer still works",
			input: "  console.log\n    hello\n    at src/utils/foo.ts:42:13\n",
			want:  "    hello",
		},
		{
			name:  "console.warn header",
			input: "  console.warn\n    [gen-forms] skipped: access denied\n      at warn (scripts/gen-forms.js:12:9)\n",
			want:  "    [gen-forms] skipped: access denied",
		},
		{
			// The critical guard: a real multi-frame stack must survive whole.
			name: "real stack trace is untouched",
			input: "  console.error\n" +
				"    TypeError: Cannot read property 'foo' of undefined\n" +
				"      at Object.<anonymous> (src/bar.test.js:12:5)\n" +
				"      at Module._compile (internal/modules/cjs/loader.js:999:30)\n" +
				"      at runTest (jest/runner.js:42:11)\n",
			want: "  console.error\n" +
				"    TypeError: Cannot read property 'foo' of undefined\n" +
				"      at Object.<anonymous> (src/bar.test.js:12:5)\n" +
				"      at Module._compile (internal/modules/cjs/loader.js:999:30)\n" +
				"      at runTest (jest/runner.js:42:11)\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := stripJestConsoleBlocks(tt.input); got != tt.want {
				t.Errorf("stripJestConsoleBlocks(%q)\n got: %q\nwant: %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestStripJestConsoleBlocks_CollapsesRepeats covers the other half of #79:
// a parallel runner emits the same setup banner once per worker.
func TestStripJestConsoleBlocks_CollapsesRepeats(t *testing.T) {
	block := "  console.log\n" +
		"    ✅ Jest setup loaded - Test environment configured\n" +
		"      at Object.log (tests/setup.js:188:9)\n" +
		"\n"

	// No runner totals line, so the verdict is unknown and the blocks are
	// kept — only the repeats collapse.
	input := strings.Repeat(block, 6) + "Running remaining suites...\n"
	want := "    ✅ Jest setup loaded - Test environment configured\n" +
		"    ... and 5 identical console blocks\n" +
		"Running remaining suites...\n"

	if got := stripJestConsoleBlocks(input); got != want {
		t.Errorf("stripJestConsoleBlocks(6 identical blocks)\n got: %q\nwant: %q", got, want)
	}
}

// TestStripJestConsoleBlocks_DropsOnPass covers the other half of #79: when the
// runner reports its own run as green, application logs are dropped outright
// and replaced by a single marker, so the truncation window goes to real output.
func TestStripJestConsoleBlocks_DropsOnPass(t *testing.T) {
	input := "  console.log\n    setup loaded\n      at Object.log (tests/setup.js:188:9)\n\n" +
		"  console.log\n    JWT AUTH ok\n      at log (middlewares/jwt.auth.js:85:13)\n\n" +
		"Tests:       1779 passed, 1779 total\n"

	want := "  [gotk: 2 console.* log blocks dropped — run passed]\n" +
		"Tests:       1779 passed, 1779 total\n"

	if got := stripJestConsoleBlocks(input); got != want {
		t.Errorf("stripJestConsoleBlocks(passing run)\n got: %q\nwant: %q", got, want)
	}
}

// TestStripJestConsoleBlocks_KeepsOnFail is the counterpart: on a red run the
// logs are the context needed to understand the failure, so they stay.
func TestStripJestConsoleBlocks_KeepsOnFail(t *testing.T) {
	input := "  console.log\n    JWT AUTH ok\n      at log (middlewares/jwt.auth.js:85:13)\n\n" +
		"Tests:       2 failed, 1777 passed, 1779 total\n"

	want := "    JWT AUTH ok\n" +
		"Tests:       2 failed, 1777 passed, 1779 total\n"

	if got := stripJestConsoleBlocks(input); got != want {
		t.Errorf("stripJestConsoleBlocks(failing run)\n got: %q\nwant: %q", got, want)
	}
}

// TestStripJestConsoleBlocks_DistinctMessagesNotCollapsed guards against the
// collapse swallowing genuinely different logs.
func TestStripJestConsoleBlocks_DistinctMessagesNotCollapsed(t *testing.T) {
	input := "  console.log\n    first message\n      at log (a.js:1:1)\n\n" +
		"  console.log\n    second message\n      at log (b.js:2:2)\n\n" +
		"  console.log\n    first message\n      at log (a.js:1:1)\n"

	want := "    first message\n    second message\n    first message"

	if got := stripJestConsoleBlocks(input); got != want {
		t.Errorf("stripJestConsoleBlocks(distinct messages)\n got: %q\nwant: %q", got, want)
	}
}
