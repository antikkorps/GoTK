package watch

import (
	"strings"
	"testing"

	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/exec"
)

// Watch must apply the same stderr policy as exec mode: secrets redacted and
// the per-command noise allowlist applied (issue #96 review).
func TestFilterOutput_AppliesStderrChain(t *testing.T) {
	cfg := Config{Command: []string{"cargo", "build"}, MaxLines: 50, GoTKConfig: config.Default()}
	res := &exec.Result{Stderr: "   Compiling a v1.0.0\n   Compiling b v1.0.0\n" +
		"error: token=ghp_FAKE00TEST00VALUE00NOT00REAL00TOKEN00xxx\n"}

	_, errOut := filterOutput(cfg, res)

	if strings.Contains(errOut, "ghp_FAKE00TEST00VALUE00NOT00REAL00TOKEN00xxx") {
		t.Errorf("secret leaked on watch stderr: %q", errOut)
	}
	if !strings.Contains(errOut, "... 2 cargo progress lines (Compiling 2)") {
		t.Errorf("cargo progress not collapsed: %q", errOut)
	}
}
