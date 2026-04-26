//go:build !windows

package update

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReplaceBinaryUnix(t *testing.T) {
	dir := t.TempDir()
	exePath := filepath.Join(dir, "gotk")
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

	// The Unix path leaves no .old or .new artifacts behind.
	if _, err := os.Stat(exePath + ".old"); !os.IsNotExist(err) {
		t.Errorf(".old should not exist on Unix: err=%v", err)
	}
	if _, err := os.Stat(exePath + ".new"); !os.IsNotExist(err) {
		t.Errorf(".new should not exist after success: err=%v", err)
	}
}
