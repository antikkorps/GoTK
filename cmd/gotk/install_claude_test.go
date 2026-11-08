package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/antikkorps/GoTK/internal/install"
)

// The notes serve every project, so dropping a project-level hook must
// not take them away from the others.
func TestClaudeUninstall_LocalKeepsInstructions(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Chdir(t.TempDir())
	if err := install.InstallInstructions(io.Discard); err != nil {
		t.Fatal(err)
	}
	if err := claudeUninstall(install.ScopeLocal); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".claude", "GOTK.md")); err != nil {
		t.Errorf("GOTK.md removed by a local uninstall: %v", err)
	}
}
