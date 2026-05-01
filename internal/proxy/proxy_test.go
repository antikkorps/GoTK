package proxy

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/detect"
)

func TestBuildChain_DefaultConfig(t *testing.T) {
	cfg := config.Default()
	chain := BuildChain(cfg, detect.CmdGeneric, cfg.General.MaxLines)
	if chain == nil {
		t.Fatal("BuildChain returned nil")
	}
}

func TestBuildChain_AllFiltersDisabled(t *testing.T) {
	cfg := config.Default()
	cfg.Filters.StripANSI = false
	cfg.Filters.NormalizeWhitespace = false
	cfg.Filters.Dedup = false
	cfg.Filters.CompressPaths = false
	cfg.Filters.TrimDecorative = false
	cfg.Filters.Truncate = false
	cfg.Security.RedactSecrets = false

	chain := BuildChain(cfg, detect.CmdGeneric, 50)
	if chain == nil {
		t.Fatal("BuildChain returned nil")
	}

	// With most filters disabled, input should pass through mostly unchanged
	// (only CompressStackTraces is always added)
	input := "hello world\n"
	got := chain.Apply(input)
	if got != input {
		t.Errorf("expected input to pass through with minimal filters, got %q", got)
	}
}

func TestBuildChain_StripANSIToggle(t *testing.T) {
	ansiInput := "\033[31mred text\033[0m\n"

	// With StripANSI enabled
	cfg := config.Default()
	cfg.Filters.NormalizeWhitespace = false
	cfg.Filters.Dedup = false
	cfg.Filters.TrimDecorative = false
	cfg.Filters.Truncate = false
	cfg.Security.RedactSecrets = false
	chain := BuildChain(cfg, detect.CmdGeneric, 50)
	got := chain.Apply(ansiInput)
	if got == ansiInput {
		t.Error("StripANSI enabled but ANSI codes were not removed")
	}

	// With StripANSI disabled
	cfg.Filters.StripANSI = false
	chain = BuildChain(cfg, detect.CmdGeneric, 50)
	got = chain.Apply(ansiInput)
	if got != ansiInput {
		t.Errorf("StripANSI disabled but output changed: %q", got)
	}
}

func TestBuildChain_SecretRedaction(t *testing.T) {
	secretInput := "API_KEY=sk-fake00test00not00real00value00fake00test00not0\n"

	// With RedactSecrets enabled
	cfg := config.Default()
	cfg.Filters.StripANSI = false
	cfg.Filters.NormalizeWhitespace = false
	cfg.Filters.Dedup = false
	cfg.Filters.TrimDecorative = false
	cfg.Filters.Truncate = false
	chain := BuildChain(cfg, detect.CmdGeneric, 50)
	got := chain.Apply(secretInput)
	if got == secretInput {
		t.Error("RedactSecrets enabled but secret was not redacted")
	}

	// With RedactSecrets disabled
	cfg.Security.RedactSecrets = false
	chain = BuildChain(cfg, detect.CmdGeneric, 50)
	got = chain.Apply(secretInput)
	if got != secretInput {
		t.Errorf("RedactSecrets disabled but output changed: got %q", got)
	}
}

func TestBuildChain_DifferentCmdTypes(t *testing.T) {
	cmdTypes := []detect.CmdType{
		detect.CmdGeneric,
		detect.CmdGrep,
		detect.CmdFind,
		detect.CmdGit,
		detect.CmdGoTool,
		detect.CmdLs,
	}

	cfg := config.Default()
	for _, ct := range cmdTypes {
		chain := BuildChain(cfg, ct, 50)
		if chain == nil {
			t.Errorf("BuildChain returned nil for CmdType %d", ct)
		}
	}
}

func TestExitCode_NilError(t *testing.T) {
	if got := exitCode(nil); got != 0 {
		t.Errorf("exitCode(nil) = %d, want 0", got)
	}
}

func TestExitCode_GenericError(t *testing.T) {
	err := os.ErrNotExist
	if got := exitCode(err); got != 1 {
		t.Errorf("exitCode(generic error) = %d, want 1", got)
	}
}

func TestRunCommand_SimpleEcho(t *testing.T) {
	cfg := config.Default()
	// Disable all filters for predictable output
	cfg.Filters.StripANSI = false
	cfg.Filters.NormalizeWhitespace = false
	cfg.Filters.Dedup = false
	cfg.Filters.CompressPaths = false
	cfg.Filters.TrimDecorative = false
	cfg.Filters.Truncate = false
	cfg.Security.RedactSecrets = false

	code := RunCommand(cfg, "echo hello", 50)
	if code != 0 {
		t.Errorf("RunCommand('echo hello') returned exit code %d, want 0", code)
	}
}

func TestRunCommand_NonZeroExit(t *testing.T) {
	cfg := config.Default()
	code := RunCommand(cfg, "exit 42", 50)
	if code != 42 {
		t.Errorf("RunCommand('exit 42') returned exit code %d, want 42", code)
	}
}

func TestPassthrough(t *testing.T) {
	orig := os.Getenv("GOTK_PASSTHROUGH")
	defer os.Setenv("GOTK_PASSTHROUGH", orig) //nolint:errcheck

	os.Setenv("GOTK_PASSTHROUGH", "1") //nolint:errcheck
	if !passthrough() {
		t.Error("passthrough() should return true when GOTK_PASSTHROUGH=1")
	}

	os.Setenv("GOTK_PASSTHROUGH", "0") //nolint:errcheck
	if passthrough() {
		t.Error("passthrough() should return false when GOTK_PASSTHROUGH=0")
	}

	os.Unsetenv("GOTK_PASSTHROUGH") //nolint:errcheck
	if passthrough() {
		t.Error("passthrough() should return false when GOTK_PASSTHROUGH is unset")
	}
}

func TestBuildChain_TruncateToggle(t *testing.T) {
	cfg := config.Default()
	cfg.Filters.StripANSI = false
	cfg.Filters.NormalizeWhitespace = false
	cfg.Filters.Dedup = false
	cfg.Filters.TrimDecorative = false
	cfg.Security.RedactSecrets = false

	// Generate input longer than 5 lines
	longInput := ""
	for i := 0; i < 100; i++ {
		longInput += "line content\n"
	}

	// With truncation enabled and low max lines
	cfg.Filters.Truncate = true
	chain := BuildChain(cfg, detect.CmdGeneric, 5)
	got := chain.Apply(longInput)
	if len(got) >= len(longInput) {
		t.Error("Truncate enabled but output was not shorter than input")
	}

	// With truncation disabled — output may have summary header prepended
	// but must contain all original content lines (not truncated)
	cfg.Filters.Truncate = false
	chain = BuildChain(cfg, detect.CmdGeneric, 5)
	got = chain.Apply(longInput)
	if !strings.Contains(got, "line content") {
		t.Error("Truncate disabled but original content is missing")
	}
	// Count that all 100 content lines survived (not truncated)
	contentLines := 0
	for _, line := range strings.Split(got, "\n") {
		if line == "line content" {
			contentLines++
		}
	}
	if contentLines != 100 {
		t.Errorf("Expected 100 content lines, got %d (truncation happened despite being disabled)", contentLines)
	}
}

func TestBuildStderrChain_RedactsSecrets(t *testing.T) {
	cfg := config.Default()
	chain := BuildStderrChain(cfg)

	stderr := "ERROR: connection refused with token=ghp_FAKE00TEST00VALUE00NOT00REAL00TOKEN00xxx\n"
	got := chain.Apply(stderr)

	if strings.Contains(got, "ghp_FAKE00TEST00VALUE00NOT00REAL00TOKEN00xxx") {
		t.Errorf("stderr chain leaked a GitHub token: %q", got)
	}
	if !strings.Contains(got, "connection refused") {
		t.Errorf("stderr chain should preserve the diagnostic text, got: %q", got)
	}
}

func TestBuildStderrChain_CollapsesNodeWorkerWarnings(t *testing.T) {
	cfg := config.Default()
	chain := BuildStderrChain(cfg)

	stderr := strings.Repeat(
		"(node:1) Warning: --localstorage-file was provided without a valid path\n"+
			"(Use `node --trace-warnings ...` to show where the warning was created)\n",
		1,
	) + "(node:2) Warning: --localstorage-file was provided without a valid path\n" +
		"(Use `node --trace-warnings ...` to show where the warning was created)\n" +
		"(node:3) Warning: --localstorage-file was provided without a valid path\n" +
		"(Use `node --trace-warnings ...` to show where the warning was created)\n"

	got := chain.Apply(stderr)

	if !strings.Contains(got, "(node:1) Warning:") {
		t.Errorf("first warning must be preserved, got:\n%s", got)
	}
	if strings.Contains(got, "(node:2)") || strings.Contains(got, "(node:3)") {
		t.Errorf("duplicate worker PIDs must be collapsed, got:\n%s", got)
	}
	if !strings.Contains(got, "identical warnings from other workers") {
		t.Errorf("collapse marker missing, got:\n%s", got)
	}
}

// TestBuildStderrChain_RealisticJestStderr_RegressionGuard mirrors a
// real multi-worker jest run where stderr carries the bulk of the output:
// node deprecation warnings from each worker, an ANSI-colored verbose
// trace, an accidental token leak, and the test-runner totals emitted on
// stderr. Acts as a regression fixture for the stderr policy chain — if
// future stderr work silently stops redacting, stops collapsing duplicate
// worker warnings, or starts mangling the totals line, this test fails.
func TestBuildStderrChain_RealisticJestStderr_RegressionGuard(t *testing.T) {
	cfg := config.Default()
	chain := BuildStderrChain(cfg)

	// Build a realistic 4-worker stderr stream.
	const warnBlock = "(node:%d) Warning: --experimental-vm-modules was provided\n" +
		"(Use `node --trace-warnings ...` to show where the warning was created)\n"
	stderr := ""
	for _, pid := range []int{1234, 1235, 1236, 1237} {
		stderr += fmt.Sprintf(warnBlock, pid)
	}
	// ANSI-colored failure trace + accidental token leak.
	stderr += "\x1b[31mFAIL\x1b[0m  src/auth/token.test.js\n"
	stderr += "  at fetchToken (src/auth/token.ts:42:11)\n"
	stderr += "  GITHUB_TOKEN=ghp_FAKE00TEST00VALUE00NOT00REAL00TOKEN00xxx\n"
	// Some runners emit totals on stderr too.
	stderr += "Tests:       1 failed, 23 passed, 24 total\n"
	stderr += "Test Suites: 1 failed, 5 passed, 6 total\n"

	got := chain.Apply(stderr)

	// Signal that MUST survive.
	for _, must := range []string{
		"FAIL",                                  // verdict
		"src/auth/token.test.js",                // failing file
		"src/auth/token.ts:42:11",               // stack frame with line:col
		"Tests:       1 failed, 23 passed",      // totals
		"Test Suites: 1 failed, 5 passed",       // suite totals
		"(node:1234)",                           // first worker warning preserved
		"identical warnings from other workers", // collapse marker
	} {
		if !strings.Contains(got, must) {
			t.Errorf("stderr chain dropped required signal %q\nGot:\n%s", must, got)
		}
	}

	// Noise / dangerous content that MUST NOT survive.
	for _, mustNot := range []string{
		"\x1b[31m", // ANSI escape
		"ghp_FAKE00TEST00VALUE00NOT00REAL00TOKEN00xxx", // token leak
		"(node:1235)", // collapsed worker
		"(node:1236)", // collapsed worker
		"(node:1237)", // collapsed worker
	} {
		if strings.Contains(got, mustNot) {
			t.Errorf("stderr chain leaked content that should be filtered: %q\nGot:\n%s", mustNot, got)
		}
	}

	// Reduction ratio: must stay within a known band so future refactors
	// don't silently regress (low) or over-aggressively trim signal (high).
	rawBytes := len(stderr)
	cleanBytes := len(got)
	reduction := float64(rawBytes-cleanBytes) / float64(rawBytes) * 100
	const minReduction, maxReduction = 30.0, 70.0
	if reduction < minReduction || reduction > maxReduction {
		t.Errorf("stderr reduction %.1f%% outside expected band [%.0f%%, %.0f%%] — raw=%d clean=%d",
			reduction, minReduction, maxReduction, rawBytes, cleanBytes)
	}
	t.Logf("stderr fixture: %d → %d bytes (%.1f%% reduction)", rawBytes, cleanBytes, reduction)
}

func TestBuildStderrChain_RespectsRedactSecretsToggle(t *testing.T) {
	cfg := config.Default()
	cfg.Security.RedactSecrets = false
	chain := BuildStderrChain(cfg)

	stderr := "API_KEY=sk-fake00test00000000000000000000000000\n"
	got := chain.Apply(stderr)

	if !strings.Contains(got, "sk-fake00test00000000000000000000000000") {
		t.Errorf("redaction was disabled, value should pass through, got: %q", got)
	}
}
