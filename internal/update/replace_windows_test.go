//go:build windows

package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceBinaryWindowsLeavesOldArtifact(t *testing.T) {
	dir := t.TempDir()
	exePath := filepath.Join(dir, "gotk.exe")
	if err := os.WriteFile(exePath, []byte("old-binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	tmpPath := filepath.Join(dir, "gotk-update-tmp")
	if err := os.WriteFile(tmpPath, []byte("new-binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	if err := replaceBinary(tmpPath, exePath); err != nil {
		t.Fatalf("replaceBinary: %v", err)
	}

	got, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "new-binary" {
		t.Errorf("exePath content = %q, want %q", got, "new-binary")
	}

	// Windows leaves the previous binary as .old until the next sweep.
	oldData, err := os.ReadFile(exePath + ".old")
	if err != nil {
		t.Fatalf("expected .old file: %v", err)
	}
	if string(oldData) != "old-binary" {
		t.Errorf(".old content = %q, want %q", oldData, "old-binary")
	}
}

func TestSweepStaleReplacementsRemovesOld(t *testing.T) {
	dir := t.TempDir()
	exePath := filepath.Join(dir, "gotk.exe")
	oldPath := exePath + ".old"
	if err := os.WriteFile(oldPath, []byte("stale"), 0o755); err != nil {
		t.Fatal(err)
	}
	sweepStaleReplacements(exePath)
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Errorf(".old should be removed: err=%v", err)
	}
}
