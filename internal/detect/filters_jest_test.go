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

func TestCompressNodeOutput_CollapsesGenericWorkerWarnings(t *testing.T) {
	input := strings.Join([]string{
		"(node:43294) Warning: --localstorage-file was provided without a valid path",
		"(Use `node --trace-warnings ...` to show where the warning was created)",
		"(node:43295) Warning: --localstorage-file was provided without a valid path",
		"(Use `node --trace-warnings ...` to show where the warning was created)",
		"(node:43296) Warning: --localstorage-file was provided without a valid path",
		"(Use `node --trace-warnings ...` to show where the warning was created)",
		"Test Suites: 1 passed",
	}, "\n")

	got := compressNodeOutput(input)

	// First occurrence (with original PID) must remain visible.
	if !strings.Contains(got, "(node:43294) Warning:") {
		t.Errorf("first warning line should be preserved, got:\n%s", got)
	}
	// Remaining PIDs must be collapsed away.
	for _, pid := range []string{"43295", "43296"} {
		if strings.Contains(got, "(node:"+pid+")") {
			t.Errorf("PID %s should have been collapsed, got:\n%s", pid, got)
		}
	}
	// The collapse marker must report the right count (2 additional workers).
	if !strings.Contains(got, "2 identical warnings") {
		t.Errorf("collapse marker should count remaining workers, got:\n%s", got)
	}
	// The `(Use \`node --trace-warnings\`...)` hint is emitted once alongside the first.
	if strings.Count(got, "trace-warnings") != 1 {
		t.Errorf("hint should appear exactly once, got:\n%s", got)
	}
	// Summary must be preserved.
	if !strings.Contains(got, "Test Suites: 1 passed") {
		t.Errorf("summary must survive, got:\n%s", got)
	}
}

func TestCompressNodeOutput_SingleGenericWarningUnchanged(t *testing.T) {
	// A single (non-repeated) generic warning must be kept intact without
	// any collapse marker.
	input := strings.Join([]string{
		"(node:100) Warning: something happened",
		"(Use `node --trace-warnings ...` to show where the warning was created)",
		"done",
	}, "\n")

	got := compressNodeOutput(input)
	if !strings.Contains(got, "(node:100) Warning: something happened") {
		t.Errorf("single warning should be preserved, got:\n%s", got)
	}
	if !strings.Contains(got, "trace-warnings") {
		t.Errorf("hint should be preserved, got:\n%s", got)
	}
	if strings.Contains(got, "identical warnings") {
		t.Errorf("collapse marker should not appear for single occurrence, got:\n%s", got)
	}
}
