package proxy

import (
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

func TestFindShell_RespectsGOTK_SHELL(t *testing.T) {
	// Save and restore env
	origGOTK := os.Getenv("GOTK_SHELL")
	origSHELL := os.Getenv("SHELL")
	defer func() {
		os.Setenv("GOTK_SHELL", origGOTK) //nolint:errcheck
		os.Setenv("SHELL", origSHELL)     //nolint:errcheck
	}()

	os.Setenv("GOTK_SHELL", "/usr/local/bin/custom-shell") //nolint:errcheck
	got := findShell()
	if got != "/usr/local/bin/custom-shell" {
		t.Errorf("findShell() = %q, want /usr/local/bin/custom-shell", got)
	}
}

func TestFindShell_AvoidsGotkRecursion(t *testing.T) {
	origGOTK := os.Getenv("GOTK_SHELL")
	origSHELL := os.Getenv("SHELL")
	defer func() {
		os.Setenv("GOTK_SHELL", origGOTK) //nolint:errcheck
		os.Setenv("SHELL", origSHELL)     //nolint:errcheck
	}()

	os.Unsetenv("GOTK_SHELL")                 //nolint:errcheck
	os.Setenv("SHELL", "/usr/local/bin/gotk") //nolint:errcheck

	got := findShell()
	if got == "/usr/local/bin/gotk" {
		t.Error("findShell() returned gotk, should avoid recursion")
	}
	// Should fall back to /bin/bash, /bin/sh, or "sh"
	if got == "" {
		t.Error("findShell() returned empty string")
	}
}

func TestFindShell_UsesSHELL(t *testing.T) {
	origGOTK := os.Getenv("GOTK_SHELL")
	origSHELL := os.Getenv("SHELL")
	defer func() {
		os.Setenv("GOTK_SHELL", origGOTK) //nolint:errcheck
		os.Setenv("SHELL", origSHELL)     //nolint:errcheck
	}()

	os.Unsetenv("GOTK_SHELL")       //nolint:errcheck
	os.Setenv("SHELL", "/bin/bash") //nolint:errcheck

	got := findShell()
	if got != "/bin/bash" {
		t.Errorf("findShell() = %q, want /bin/bash", got)
	}
}

func TestFindShell_ReturnsValidPath(t *testing.T) {
	got := findShell()
	if got == "" {
		t.Fatal("findShell() returned empty string")
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
