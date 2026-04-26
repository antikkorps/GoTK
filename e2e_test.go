//go:build !windows

// The e2e suite drives the compiled gotk binary through scenarios that rely
// on a POSIX shell (`sh -c "..."`, `#!/bin/sh` scripts, signal forwarding to
// child processes). Porting this to Windows is its own effort — for now the
// build tag keeps the suite POSIX-only. Cross-platform code is exercised by
// per-package unit tests; the Linux + macOS CI jobs run e2e here.

package gotk_test

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// binary returns the path to the compiled gotk binary, building it if needed.
func binary(t *testing.T) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "gotk")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/gotk/")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to build gotk: %v", err)
	}
	return bin
}

// run executes the gotk binary with the given args and piped stdin.
// Returns stdout, stderr, and exit code.
func run(t *testing.T, bin string, stdin string, args ...string) (string, string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("failed to run gotk: %v", err)
	}
	return stdout.String(), stderr.String(), code
}

func TestE2E_PipeMode(t *testing.T) {
	bin := binary(t)
	input := "line1\nline2\nline3\n"
	stdout, _, code := run(t, bin, input)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "line1") {
		t.Errorf("output should contain 'line1', got: %q", stdout)
	}
}

func TestE2E_PipeMode_Stats(t *testing.T) {
	bin := binary(t)
	input := "hello\nhello\nhello\nhello\nhello\n"
	stdout, stderr, _ := run(t, bin, input, "--stats")
	if !strings.Contains(stderr, "[gotk]") {
		t.Errorf("--stats should produce stats on stderr, got: %q", stderr)
	}
	// Dedup should compress the 5 identical lines
	if strings.Count(stdout, "hello") >= 5 {
		t.Errorf("dedup should reduce duplicate lines, got: %q", stdout)
	}
}

func TestE2E_PipeMode_Dedup(t *testing.T) {
	bin := binary(t)
	lines := strings.Repeat("duplicate line\n", 20)
	stdout, _, _ := run(t, bin, lines)
	if strings.Count(stdout, "duplicate line") > 5 {
		t.Errorf("dedup should compress 20 identical lines, got: %q", stdout)
	}
	if !strings.Contains(stdout, "duplicate") {
		t.Errorf("output should still contain 'duplicate', got: %q", stdout)
	}
}

func TestE2E_PipeMode_ANSIStrip(t *testing.T) {
	bin := binary(t)
	input := "\x1b[31mred text\x1b[0m\n\x1b[1mbold\x1b[0m\n"
	stdout, _, _ := run(t, bin, input)
	if strings.Contains(stdout, "\x1b[") {
		t.Errorf("ANSI codes should be stripped, got: %q", stdout)
	}
	if !strings.Contains(stdout, "red text") {
		t.Errorf("text content should be preserved, got: %q", stdout)
	}
}

func TestE2E_PipeMode_SecretRedaction(t *testing.T) {
	bin := binary(t)
	input := "API_KEY=sk-fake00test00not00real00value00fake00test\nother line\n"
	stdout, _, _ := run(t, bin, input)
	if strings.Contains(stdout, "sk-fake00test") {
		t.Errorf("secret should be redacted, got: %q", stdout)
	}
	if !strings.Contains(stdout, "[REDACTED]") {
		t.Errorf("output should contain [REDACTED], got: %q", stdout)
	}
}

func TestE2E_PipeMode_Truncation(t *testing.T) {
	bin := binary(t)
	// Generate 200 unique lines
	var lines []string
	for i := range 200 {
		lines = append(lines, "unique line number "+strings.Repeat("x", i%50+1))
	}
	input := strings.Join(lines, "\n") + "\n"
	stdout, _, _ := run(t, bin, input, "--max-lines", "20")
	if !strings.Contains(stdout, "omitted") {
		t.Errorf("truncation should show omission marker, got: %q", stdout)
	}
}

func TestE2E_PipeMode_NoTruncate(t *testing.T) {
	bin := binary(t)
	var lines []string
	for range 100 {
		lines = append(lines, "unique line content here")
	}
	input := strings.Join(lines, "\n") + "\n"
	stdout, _, _ := run(t, bin, input, "--no-truncate")
	if strings.Contains(stdout, "omitted") {
		t.Errorf("--no-truncate should not truncate, got omission marker")
	}
}

func TestE2E_DirectMode_Echo(t *testing.T) {
	bin := binary(t)
	stdout, _, code := run(t, bin, "", "echo", "hello world")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "hello world") {
		t.Errorf("output should contain 'hello world', got: %q", stdout)
	}
}

func TestE2E_DirectMode_ExitCode(t *testing.T) {
	bin := binary(t)
	_, _, code := run(t, bin, "", "sh", "-c", "exit 42")
	if code != 42 {
		t.Errorf("exit code = %d, want 42", code)
	}
}

func TestE2E_ShellCmd(t *testing.T) {
	bin := binary(t)
	stdout, _, code := run(t, bin, "", "-c", "echo from shell cmd")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "from shell cmd") {
		t.Errorf("output should contain 'from shell cmd', got: %q", stdout)
	}
}

func TestE2E_Bench(t *testing.T) {
	bin := binary(t)
	stdout, _, code := run(t, bin, "", "bench")
	if code != 0 {
		t.Errorf("bench exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "GoTK Benchmark Report") {
		t.Errorf("bench should output report, got: %q", stdout)
	}
}

func TestE2E_BenchJSON(t *testing.T) {
	bin := binary(t)
	stdout, _, code := run(t, bin, "", "bench", "--json")
	if code != 0 {
		t.Errorf("bench --json exit code = %d, want 0", code)
	}
	if !strings.HasPrefix(strings.TrimSpace(stdout), "{") {
		t.Errorf("bench --json should output JSON, got: %q", stdout[:min(len(stdout), 50)])
	}
}

func TestE2E_BenchQuality(t *testing.T) {
	bin := binary(t)
	stdout, _, code := run(t, bin, "", "bench", "--quality")
	if code != 0 {
		t.Errorf("bench --quality exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Quality Score Report") {
		t.Errorf("bench --quality should output quality report, got: %q", stdout)
	}
}

func TestE2E_Help(t *testing.T) {
	bin := binary(t)
	stdout, _, code := run(t, bin, "", "--help")
	if code != 0 {
		t.Errorf("--help exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "GoTK") {
		t.Errorf("--help should mention GoTK, got: %q", stdout)
	}
	if !strings.Contains(stdout, "--aggressive") {
		t.Errorf("--help should mention --aggressive flag")
	}
	if !strings.Contains(stdout, ".gotk.toml") {
		t.Errorf("--help should mention .gotk.toml project config")
	}
}

func TestE2E_Version(t *testing.T) {
	bin := binary(t)
	stdout, _, code := run(t, bin, "", "--version")
	if code != 0 {
		t.Errorf("--version exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "gotk") {
		t.Errorf("--version should print version, got: %q", stdout)
	}
}

func TestE2E_AggressiveMode(t *testing.T) {
	bin := binary(t)
	// Generate enough UNIQUE lines to see truncation difference
	var lines []string
	for i := range 100 {
		lines = append(lines, "unique output line number "+strings.Repeat("x", i))
	}
	input := strings.Join(lines, "\n") + "\n"

	aggressiveOut, _, _ := run(t, bin, input, "--aggressive")
	conservativeOut, _, _ := run(t, bin, input, "--conservative")

	// Aggressive should produce less output than conservative
	if len(aggressiveOut) >= len(conservativeOut) {
		t.Errorf("aggressive (%d bytes) should produce less output than conservative (%d bytes)",
			len(aggressiveOut), len(conservativeOut))
	}
}

func TestE2E_Passthrough(t *testing.T) {
	bin := binary(t)
	// Test passthrough via -c mode (proxy path) where GOTK_PASSTHROUGH is checked
	cmd := exec.Command(bin, "-c", "echo hello from passthrough")
	cmd.Env = append(os.Environ(), "GOTK_PASSTHROUGH=1")
	var stdout strings.Builder
	cmd.Stdout = &stdout
	if err := cmd.Run(); err != nil {
		t.Fatalf("passthrough run failed: %v", err)
	}

	if !strings.Contains(stdout.String(), "hello from passthrough") {
		t.Errorf("passthrough should preserve output, got: %q", stdout.String())
	}
}

func TestE2E_Ctx_Scan(t *testing.T) {
	bin := binary(t)
	stdout, _, code := run(t, bin, "", "ctx", "BuildChain", "-t", "go")
	if code != 0 {
		t.Errorf("ctx scan exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "BuildChain") {
		t.Errorf("ctx scan should contain match, got: %q", stdout)
	}
	// Scan mode shows "Nx file.go" format
	if !strings.Contains(stdout, "x ") {
		t.Errorf("ctx scan should show match count format 'Nx file', got: %q", stdout)
	}
}

func TestE2E_Ctx_Detail(t *testing.T) {
	bin := binary(t)
	stdout, _, code := run(t, bin, "", "ctx", "BuildChain", "-d", "3", "-t", "go")
	if code != 0 {
		t.Errorf("ctx detail exit code = %d, want 0", code)
	}
	// Detail mode has "--- file:line ---" headers
	if !strings.Contains(stdout, "---") {
		t.Errorf("ctx detail should contain '---' headers, got: %q", stdout)
	}
	// Detail mode marks matching lines with ">"
	if !strings.Contains(stdout, ">") {
		t.Errorf("ctx detail should contain '>' match markers, got: %q", stdout)
	}
}

func TestE2E_Ctx_Def(t *testing.T) {
	bin := binary(t)
	stdout, _, code := run(t, bin, "", "ctx", "BuildChain", "--def", "-t", "go")
	if code != 0 {
		t.Errorf("ctx def exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "BuildChain") {
		t.Errorf("ctx def should contain match, got: %q", stdout)
	}
}

func TestE2E_Ctx_Tree(t *testing.T) {
	bin := binary(t)
	stdout, _, code := run(t, bin, "", "ctx", "BuildChain", "--tree", "-t", "go")
	if code != 0 {
		t.Errorf("ctx tree exit code = %d, want 0", code)
	}
	// Tree mode has "== file ==" headers
	if !strings.Contains(stdout, "==") {
		t.Errorf("ctx tree should contain '==' file headers, got: %q", stdout)
	}
	// Should show structural elements like package/import/func
	if !strings.Contains(stdout, "package") {
		t.Errorf("ctx tree should show 'package' in skeleton, got: %q", stdout)
	}
}

func TestE2E_Ctx_Summary(t *testing.T) {
	bin := binary(t)
	stdout, _, code := run(t, bin, "", "ctx", "BuildChain", "--summary", "-t", "go")
	if code != 0 {
		t.Errorf("ctx summary exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Pattern: BuildChain") {
		t.Errorf("ctx summary should show pattern, got: %q", stdout)
	}
	if !strings.Contains(stdout, "Total:") {
		t.Errorf("ctx summary should show totals, got: %q", stdout)
	}
	if !strings.Contains(stdout, "Directory") {
		t.Errorf("ctx summary should show directory header, got: %q", stdout)
	}
}

func TestE2E_Ctx_Stats(t *testing.T) {
	bin := binary(t)
	stdout, _, code := run(t, bin, "", "--stats", "ctx", "BuildChain", "-t", "go")
	if code != 0 {
		t.Errorf("ctx stats exit code = %d, want 0", code)
	}
	// Stats are embedded in the output by gotkctx.Run
	if !strings.Contains(stdout, "[gotk]") {
		t.Errorf("--stats should produce stats in output, got: %q", stdout)
	}
}

func TestE2E_Ctx_NoMatch(t *testing.T) {
	bin := binary(t)
	// Search in a temp dir with no matching files to guarantee no matches
	dir := t.TempDir()
	os.WriteFile(dir+"/test.go", []byte("package main\nfunc main() {}\n"), 0644) //nolint:errcheck
	stdout, _, code := run(t, bin, "", "ctx", "xyzNeverMatchThis123", "-p", dir)
	if code != 0 {
		t.Errorf("ctx no-match exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "no matches") {
		t.Errorf("ctx no-match should say 'no matches', got: %q", stdout)
	}
}

func TestE2E_Ctx_Help(t *testing.T) {
	bin := binary(t)
	stdout, _, code := run(t, bin, "", "help", "ctx")
	if code != 0 {
		t.Errorf("help ctx exit code = %d, want 0", code)
	}
	if !strings.Contains(stdout, "Context Search") {
		t.Errorf("help ctx should contain 'Context Search', got: %q", stdout)
	}
	if !strings.Contains(stdout, "--def") {
		t.Errorf("help ctx should mention --def flag, got: %q", stdout)
	}
}

// ---------------------------------------------------------------------------
// Complex filter interactions
// ---------------------------------------------------------------------------

func TestE2E_FilterInteraction_GrepOutput(t *testing.T) {
	bin := binary(t)
	// Simulate grep -rn output with ANSI colors, duplicate lines, and error messages
	input := strings.Join([]string{
		"\x1b[35msrc/main.go\x1b[0m:\x1b[32m10\x1b[0m:func main() {",
		"\x1b[35msrc/main.go\x1b[0m:\x1b[32m11\x1b[0m:    fmt.Println(\"hello\")",
		"\x1b[35msrc/main.go\x1b[0m:\x1b[32m12\x1b[0m:    fmt.Println(\"hello\")",
		"\x1b[35msrc/utils.go\x1b[0m:\x1b[32m5\x1b[0m:func helper() error {",
		"\x1b[35msrc/utils.go\x1b[0m:\x1b[32m6\x1b[0m:    return fmt.Errorf(\"connection refused\")",
		"grep: /tmp/binary.dat: binary file matches",
		"\x1b[35msrc/handler.go\x1b[0m:\x1b[32m20\x1b[0m:    log.Fatal(\"critical error: out of memory\")",
	}, "\n") + "\n"

	stdout, _, code := run(t, bin, input)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// ANSI codes should be stripped
	if strings.Contains(stdout, "\x1b[") {
		t.Error("ANSI codes should be stripped from grep output")
	}
	// Error/warning lines must be preserved
	if !strings.Contains(stdout, "connection refused") {
		t.Error("error message 'connection refused' should be preserved")
	}
	if !strings.Contains(stdout, "critical error") {
		t.Error("error message 'critical error' should be preserved")
	}
	// Functional content should be preserved
	if !strings.Contains(stdout, "func main()") {
		t.Error("function definition should be preserved")
	}
}

func TestE2E_FilterInteraction_GoTestOutput(t *testing.T) {
	bin := binary(t)
	// Simulate go test output with mix of PASS, FAIL, and error lines
	input := strings.Join([]string{
		"=== RUN   TestAdd",
		"--- PASS: TestAdd (0.00s)",
		"=== RUN   TestSubtract",
		"--- PASS: TestSubtract (0.00s)",
		"=== RUN   TestDivide",
		"    divide_test.go:15: expected 5, got 0",
		"    divide_test.go:16: division by zero not handled",
		"--- FAIL: TestDivide (0.00s)",
		"=== RUN   TestMultiply",
		"--- PASS: TestMultiply (0.00s)",
		"FAIL",
		"exit status 1",
		"FAIL\texample.com/calc\t0.003s",
	}, "\n") + "\n"

	stdout, _, code := run(t, bin, input)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// FAIL lines must be preserved (most important for LLM debugging)
	if !strings.Contains(stdout, "FAIL") {
		t.Error("FAIL lines should be preserved in go test output")
	}
	// Error details must be preserved
	if !strings.Contains(stdout, "division by zero") {
		t.Error("error detail 'division by zero' should be preserved")
	}
	if !strings.Contains(stdout, "expected 5, got 0") {
		t.Error("assertion detail should be preserved")
	}
	// Summary line should be preserved
	if !strings.Contains(stdout, "exit status 1") {
		t.Error("exit status line should be preserved")
	}
}

func TestE2E_FilterInteraction_GitDiffOutput(t *testing.T) {
	bin := binary(t)
	// Simulate git diff output
	input := strings.Join([]string{
		"diff --git a/main.go b/main.go",
		"index abc1234..def5678 100644",
		"--- a/main.go",
		"+++ b/main.go",
		"@@ -10,6 +10,8 @@ func main() {",
		"     fmt.Println(\"hello\")",
		"-    os.Exit(0)",
		"+    if err != nil {",
		"+        log.Fatal(err)",
		"+    }",
		"     fmt.Println(\"done\")",
		"diff --git a/go.mod b/go.mod",
		"index 111aaaa..222bbbb 100644",
		"--- a/go.mod",
		"+++ b/go.mod",
		"@@ -1,3 +1,4 @@",
		" module example.com/app",
		"+require golang.org/x/text v0.3.0",
	}, "\n") + "\n"

	stdout, _, code := run(t, bin, input)
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// Diff markers should be preserved
	if !strings.Contains(stdout, "diff --git") {
		t.Error("diff header should be preserved")
	}
	// Added/removed lines should be preserved
	if !strings.Contains(stdout, "+    if err != nil") {
		t.Error("added lines should be preserved in diff output")
	}
	if !strings.Contains(stdout, "-    os.Exit(0)") {
		t.Error("removed lines should be preserved in diff output")
	}
	// Hunk headers should be preserved
	if !strings.Contains(stdout, "@@") {
		t.Error("hunk headers (@@) should be preserved")
	}
}

// ---------------------------------------------------------------------------
// Large output handling
// ---------------------------------------------------------------------------

func TestE2E_LargeOutput_Truncation(t *testing.T) {
	bin := binary(t)
	// Generate 10,000+ unique lines
	var lines []string
	for i := 0; i < 10500; i++ {
		lines = append(lines, fmt.Sprintf("unique output line %d with data %x", i, i*17))
	}
	input := strings.Join(lines, "\n") + "\n"

	stdout, _, code := run(t, bin, input, "--max-lines", "50")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// Should contain the omission marker
	if !strings.Contains(stdout, "lines omitted") {
		t.Errorf("large output should be truncated with omission marker, got %d bytes", len(stdout))
	}
	// Output should be significantly smaller than input
	if len(stdout) > len(input)/2 {
		t.Errorf("truncated output (%d bytes) should be much smaller than input (%d bytes)", len(stdout), len(input))
	}
	// Head lines should be present (first lines)
	if !strings.Contains(stdout, "unique output line 0") {
		t.Error("head of output should be preserved after truncation")
	}
	// Tail lines should be present (last lines)
	if !strings.Contains(stdout, "unique output line 10499") {
		t.Error("tail of output should be preserved after truncation")
	}
}

func TestE2E_LargeOutput_NoTruncate(t *testing.T) {
	bin := binary(t)
	// Generate 10,000+ unique lines
	var lines []string
	for i := 0; i < 10500; i++ {
		lines = append(lines, fmt.Sprintf("line-%d-data", i))
	}
	input := strings.Join(lines, "\n") + "\n"

	stdout, _, code := run(t, bin, input, "--no-truncate")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// Should NOT contain omission marker
	if strings.Contains(stdout, "lines omitted") {
		t.Error("--no-truncate should not truncate large output")
	}
	// First and last lines should be present
	if !strings.Contains(stdout, "line-0-data") {
		t.Error("first line should be present with --no-truncate")
	}
	if !strings.Contains(stdout, "line-10499-data") {
		t.Error("last line should be present with --no-truncate")
	}
}

func TestE2E_LargeOutput_ErrorsPreservedAfterTruncation(t *testing.T) {
	bin := binary(t)
	// Generate large output with error lines scattered throughout, including at the end
	var lines []string
	for i := 0; i < 500; i++ {
		lines = append(lines, fmt.Sprintf("normal output line %d", i))
		if i == 5 {
			lines = append(lines, "ERROR: connection timeout at startup")
		}
	}
	// Add errors near the end (these should be in the tail section)
	lines = append(lines, "FATAL: process crashed with signal SIGSEGV")
	lines = append(lines, "exit status 1")
	input := strings.Join(lines, "\n") + "\n"

	stdout, _, code := run(t, bin, input, "--max-lines", "30")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// Truncation should have occurred
	if !strings.Contains(stdout, "lines omitted") {
		t.Error("output should be truncated")
	}
	// Errors at the tail should be preserved (truncation keeps head + tail)
	if !strings.Contains(stdout, "FATAL: process crashed") {
		t.Error("tail error 'FATAL: process crashed' should be preserved after truncation")
	}
	if !strings.Contains(stdout, "exit status 1") {
		t.Error("tail 'exit status 1' should be preserved after truncation")
	}
}

// ---------------------------------------------------------------------------
// Signal handling and edge cases
// ---------------------------------------------------------------------------

func TestE2E_ExecNonZeroExitWithOutput(t *testing.T) {
	bin := binary(t)
	// Command that produces output and exits non-zero
	stdout, _, code := run(t, bin, "", "sh", "-c", "echo 'some output before failure'; echo 'error: something went wrong' >&2; exit 7")
	if code != 7 {
		t.Errorf("exit code = %d, want 7", code)
	}
	if !strings.Contains(stdout, "some output before failure") {
		t.Errorf("stdout should contain output before failure, got: %q", stdout)
	}
}

func TestE2E_MaxLines_Zero(t *testing.T) {
	bin := binary(t)
	// --max-lines 0 should disable truncation (same as --no-truncate)
	var lines []string
	for i := 0; i < 200; i++ {
		lines = append(lines, fmt.Sprintf("line-zero-test-%d", i))
	}
	input := strings.Join(lines, "\n") + "\n"

	stdout, _, code := run(t, bin, input, "--max-lines", "0")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// maxLines=0 disables truncation
	if strings.Contains(stdout, "lines omitted") {
		t.Error("--max-lines 0 should disable truncation")
	}
}

func TestE2E_MaxLines_One(t *testing.T) {
	bin := binary(t)
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, fmt.Sprintf("edge-case-line-%d", i))
	}
	input := strings.Join(lines, "\n") + "\n"

	stdout, _, code := run(t, bin, input, "--max-lines", "1")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	// With max-lines=1, output should be very small
	outputLines := strings.Split(strings.TrimSpace(stdout), "\n")
	// Should have very few content lines (1 kept + possibly omission marker)
	if len(outputLines) > 5 {
		t.Errorf("--max-lines 1 should produce very few lines, got %d lines", len(outputLines))
	}
}

// runMerged executes gotk and returns a single buffer that concatenates
// child stdout and stderr in the order they were written (os/exec preserves
// the order when both descriptors target the same writer). This reproduces
// the user's `2>&1` pipeline at the shell level so we can assert the
// relative ordering of lines across both streams.
func runMerged(t *testing.T, bin string, args ...string) (string, int) {
	t.Helper()
	cmd := exec.Command(bin, args...)
	var buf strings.Builder
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	code := 0
	if exitErr, ok := err.(*exec.ExitError); ok {
		code = exitErr.ExitCode()
	} else if err != nil {
		t.Fatalf("failed to run gotk: %v", err)
	}
	return buf.String(), code
}

// Regression for issue #27: when stdout and stderr are merged (e.g. `2>&1`
// in the user's shell), the `[gotk] X -> Y bytes` stats line must land at
// the very end of the stream — after any passthrough stderr emitted by the
// wrapped command. Earlier builds emitted stats before stderr passthrough,
// causing the marker to appear in the middle of the merged output.
func TestE2E_StatsLandsAtEndOfMergedStream(t *testing.T) {
	bin := binary(t)
	// The child writes to stdout, then to stderr, then exits. Under a merged
	// stream the observed order should be: stdout, stderr, [gotk] stats.
	merged, code := runMerged(t, bin, "-s", "sh", "-c",
		"echo child-stdout; echo child-stderr-final >&2")
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; output:\n%s", code, merged)
	}
	stdoutIdx := strings.Index(merged, "child-stdout")
	stderrIdx := strings.Index(merged, "child-stderr-final")
	statsIdx := strings.Index(merged, "[gotk]")
	if stdoutIdx < 0 || stderrIdx < 0 || statsIdx < 0 {
		t.Fatalf("expected all three markers in merged output, got:\n%s", merged)
	}
	if stdoutIdx >= stderrIdx || stderrIdx >= statsIdx {
		t.Errorf("stats must land after stderr passthrough (stdout=%d, stderr=%d, stats=%d):\n%s",
			stdoutIdx, stderrIdx, statsIdx, merged)
	}
}

// Regression for issue #28: Jest wraps every test-side `console.log` call
// with a header line, the message, and an `at <path>:<line>:<col>` trailer.
// The filter must strip the header and the `at` trailer, keeping only the
// message content. Exercised through pipe mode to match the most common
// integration pattern (`gotk < jest-output.txt`).
func TestE2E_JestConsoleBlocksStripped(t *testing.T) {
	bin := binary(t)
	// Force CmdNpm by executing through a fake `npm` shim; pipe mode alone
	// goes through auto-detect which may not recognize Jest output.
	shim := filepath.Join(t.TempDir(), "npm")
	body := `#!/bin/sh
cat <<'RAW'
  console.log
    Session created for user_12345
      at src/auth/session.ts:88:7

Test Suites: 1 passed
RAW
`
	if err := os.WriteFile(shim, []byte(body), 0755); err != nil {
		t.Fatalf("write shim: %v", err)
	}
	stdout, _, code := run(t, bin, "", "exec", "--", shim)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout:\n%s", code, stdout)
	}
	if strings.Contains(stdout, "console.log") {
		t.Errorf("console.log header should be stripped, got:\n%s", stdout)
	}
	if strings.Contains(stdout, "at src/auth/session.ts:88:7") {
		t.Errorf("`at path:line:col` trailer should be stripped, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Session created for user_12345") {
		t.Errorf("message content must be preserved, got:\n%s", stdout)
	}
	if !strings.Contains(stdout, "Test Suites: 1 passed") {
		t.Errorf("summary must pass through, got:\n%s", stdout)
	}
}

// Regression for issue #32: repeated `(node:PID) Warning: …` blocks from
// multi-worker Jest runs differ only in the PID. The filter collapses them
// into one original-line occurrence plus a count marker for the rest.
func TestE2E_MultiWorkerNodeWarningsCollapsed(t *testing.T) {
	bin := binary(t)
	shim := filepath.Join(t.TempDir(), "npm")
	body := `#!/bin/sh
cat <<'RAW'
(node:1001) Warning: something happened in a worker
(Use ` + "`" + `node --trace-warnings ...` + "`" + ` to show where the warning was created)
(node:1002) Warning: something happened in a worker
(Use ` + "`" + `node --trace-warnings ...` + "`" + ` to show where the warning was created)
(node:1003) Warning: something happened in a worker
(Use ` + "`" + `node --trace-warnings ...` + "`" + ` to show where the warning was created)
Test Suites: 3 passed
RAW
`
	if err := os.WriteFile(shim, []byte(body), 0755); err != nil {
		t.Fatalf("write shim: %v", err)
	}
	stdout, _, code := run(t, bin, "", "exec", "--", shim)
	if code != 0 {
		t.Fatalf("exit code = %d, want 0; stdout:\n%s", code, stdout)
	}
	if !strings.Contains(stdout, "(node:1001) Warning:") {
		t.Errorf("first warning should be kept verbatim, got:\n%s", stdout)
	}
	for _, pid := range []string{"1002", "1003"} {
		if strings.Contains(stdout, "(node:"+pid+")") {
			t.Errorf("PID %s should have been collapsed, got:\n%s", pid, stdout)
		}
	}
	if !strings.Contains(stdout, "2 identical warnings") {
		t.Errorf("collapse marker should count remaining workers, got:\n%s", stdout)
	}
	if strings.Count(stdout, "trace-warnings") != 1 {
		t.Errorf("hint should appear exactly once, got:\n%s", stdout)
	}
}

// TestE2E_UpdateNoticeSilentOnPipe verifies the passive "update
// available" notice doesn't leak into captured output — i.e. it's
// silenced when stderr isn't a TTY. This is what keeps scripts, pipes,
// and CI runs quiet even when a newer gotk release is cached.
func TestE2E_UpdateNoticeSilentOnPipe(t *testing.T) {
	bin := binary(t)
	home := t.TempDir()
	cacheDir := filepath.Join(home, ".local", "share", "gotk")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		t.Fatal(err)
	}
	cache := `{"checked_at":"2026-04-20T00:00:00Z","latest":"v99.0.0"}`
	if err := os.WriteFile(filepath.Join(cacheDir, "update_check.json"), []byte(cache), 0600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "--version")
	cmd.Env = append(os.Environ(), "HOME="+home, "CI=", "GITHUB_ACTIONS=", "GOTK_NO_UPDATE_CHECK=")
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("gotk --version failed: %v (stderr=%s)", err, stderr.String())
	}
	if strings.Contains(stderr.String(), "update available") {
		t.Errorf("update notice must not appear when stderr is captured (not a TTY), got:\n%s", stderr.String())
	}
}

// TestE2E_UninstallClaudeSymmetric covers `gotk uninstall claude` as a
// symmetric alias for `gotk install claude --uninstall`. The user can now
// write uninstall/install as parallel verbs instead of a flag toggle.
func TestE2E_UninstallClaudeSymmetric(t *testing.T) {
	bin := binary(t)
	project := t.TempDir()
	oldCwd, _ := os.Getwd()
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldCwd) //nolint:errcheck

	// install → settings.local.json gets a GoTK hook
	_, _, code := run(t, bin, "", "install", "claude", "--local")
	if code != 0 {
		t.Fatalf("install exit = %d", code)
	}
	settingsPath := filepath.Join(project, ".claude", "settings.local.json")
	data, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings file not created: %v", err)
	}
	if !strings.Contains(string(data), "gotk hook") {
		t.Fatalf("hook not installed, got: %s", data)
	}

	// uninstall claude → hook removed, file remains (possibly empty JSON)
	_, _, code = run(t, bin, "", "uninstall", "claude", "--local")
	if code != 0 {
		t.Fatalf("uninstall exit = %d", code)
	}
	data, err = os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("settings file removed by uninstall: %v", err)
	}
	if strings.Contains(string(data), "gotk hook") {
		t.Errorf("hook still present after uninstall: %s", data)
	}
}

// TestE2E_UninstallDryRun runs `gotk uninstall --dry-run` and verifies
// (a) it prints a plan and (b) it makes no filesystem changes.
func TestE2E_UninstallDryRun(t *testing.T) {
	bin := binary(t)
	project := t.TempDir()
	oldCwd, _ := os.Getwd()
	if err := os.Chdir(project); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldCwd) //nolint:errcheck

	// Fake home so we don't touch the user's real config.
	home := t.TempDir()
	cfg := filepath.Join(home, ".config", "gotk", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfg), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("# test\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "uninstall", "--dry-run")
	cmd.Env = append(os.Environ(), "HOME="+home)
	var stdout, stderr strings.Builder
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("dry-run errored: %v (stderr=%s)", err, stderr.String())
	}

	combined := stdout.String() + stderr.String()
	for _, want := range []string{"This will remove GoTK", "Binary", "dry-run"} {
		if !strings.Contains(combined, want) {
			t.Errorf("dry-run output missing %q, got:\n%s", want, combined)
		}
	}
	// config file must still exist
	if _, err := os.Stat(cfg); err != nil {
		t.Errorf("dry-run must not touch files, stat err: %v", err)
	}
}

// TestE2E_UninstallYesRemovesConfig runs `gotk uninstall --yes` against a
// fake home populated with a config file and verifies the file (and its
// now-empty parent directory) are removed without prompting.
func TestE2E_UninstallYesRemovesConfig(t *testing.T) {
	bin := binary(t)
	home := t.TempDir()
	cfg := filepath.Join(home, ".config", "gotk", "config.toml")
	if err := os.MkdirAll(filepath.Dir(cfg), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfg, []byte("# test\n"), 0600); err != nil {
		t.Fatal(err)
	}

	cmd := exec.Command(bin, "uninstall", "--yes")
	cmd.Env = append(os.Environ(), "HOME="+home)
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("uninstall --yes errored: %v (stderr=%s)", err, stderr.String())
	}
	if _, err := os.Stat(cfg); !os.IsNotExist(err) {
		t.Errorf("config.toml should be removed, stat err: %v", err)
	}
	if !strings.Contains(stderr.String(), "rm ") {
		t.Errorf("expected an `rm <binary>` hint in output, got:\n%s", stderr.String())
	}
}

func TestE2E_MaxLines_VeryLarge(t *testing.T) {
	bin := binary(t)
	// With a very large --max-lines value, no truncation should occur for small input
	var lines []string
	for i := 0; i < 100; i++ {
		lines = append(lines, fmt.Sprintf("large-limit-line-%d", i))
	}
	input := strings.Join(lines, "\n") + "\n"

	stdout, _, code := run(t, bin, input, "--max-lines", "999999")
	if code != 0 {
		t.Errorf("exit code = %d, want 0", code)
	}
	if strings.Contains(stdout, "lines omitted") {
		t.Error("very large --max-lines should not truncate small input")
	}
	// Content should be preserved
	if !strings.Contains(stdout, "large-limit-line-0") {
		t.Error("first line should be present")
	}
	if !strings.Contains(stdout, "large-limit-line-99") {
		t.Error("last line should be present")
	}
}
