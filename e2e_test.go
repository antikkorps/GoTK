package gotk_test

import (
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
