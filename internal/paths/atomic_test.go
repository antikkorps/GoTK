package paths

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestWriteFileAtomic_WritesWithPerm(t *testing.T) {
	path := filepath.Join(t.TempDir(), "sub", "f.txt")
	if err := WriteFileAtomic(path, []byte("hello"), 0o600); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil || string(data) != "hello" {
		t.Fatalf("content %q err %v", data, err)
	}
	if runtime.GOOS != "windows" {
		if info, _ := os.Stat(path); info.Mode().Perm() != 0o600 {
			t.Errorf("perm = %v, want 0600", info.Mode().Perm())
		}
	}
}

// Dotfile managers symlink ~/.claude/CLAUDE.md into a repo; a plain rename
// would replace the link with a regular file and fork the user's config.
func TestWriteFileAtomic_FollowsSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlinks need privileges on Windows")
	}
	dir := t.TempDir()
	target := filepath.Join(dir, "real.md")
	link := filepath.Join(dir, "link.md")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(link, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	if info, _ := os.Lstat(link); info.Mode()&os.ModeSymlink == 0 {
		t.Error("symlink replaced by a regular file")
	}
	if data, _ := os.ReadFile(target); string(data) != "new" {
		t.Errorf("target = %q, want new", data)
	}
}
