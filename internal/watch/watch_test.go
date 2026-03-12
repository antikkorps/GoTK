package watch

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/antikkorps/GoTK/internal/config"
)

func TestTakeSnapshot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tests use Unix paths")
	}

	dir := t.TempDir()

	// Create test files with different extensions.
	writeFile(t, filepath.Join(dir, "main.go"), "package main")
	writeFile(t, filepath.Join(dir, "util.go"), "package util")
	writeFile(t, filepath.Join(dir, "readme.txt"), "hello")
	writeFile(t, filepath.Join(dir, "data.json"), "{}")

	t.Run("snapshot captures all files when no extension filter", func(t *testing.T) {
		snap := takeSnapshot([]string{dir}, nil)
		if len(snap) != 4 {
			t.Errorf("expected 4 files, got %d", len(snap))
		}
	})

	t.Run("snapshot filters by extension", func(t *testing.T) {
		snap := takeSnapshot([]string{dir}, []string{".go"})
		if len(snap) != 2 {
			t.Errorf("expected 2 .go files, got %d", len(snap))
		}
		for path := range snap {
			if filepath.Ext(path) != ".go" {
				t.Errorf("unexpected file in snapshot: %s", path)
			}
		}
	})
}

func TestSnapshotChanged(t *testing.T) {
	dir := t.TempDir()

	file1 := filepath.Join(dir, "a.go")
	writeFile(t, file1, "v1")
	time.Sleep(10 * time.Millisecond)

	snap1 := takeSnapshot([]string{dir}, []string{".go"})

	t.Run("no change detected when files unchanged", func(t *testing.T) {
		snap2 := takeSnapshot([]string{dir}, []string{".go"})
		if snapshotChanged(snap1, snap2) {
			t.Error("expected no change, but change was detected")
		}
	})

	t.Run("change detected when file modified", func(t *testing.T) {
		// Ensure mod time changes (some filesystems have 1s granularity).
		time.Sleep(50 * time.Millisecond)
		writeFile(t, file1, "v2")

		snap2 := takeSnapshot([]string{dir}, []string{".go"})
		if !snapshotChanged(snap1, snap2) {
			t.Error("expected change after file modification, but none detected")
		}
	})

	t.Run("change detected when file added", func(t *testing.T) {
		writeFile(t, filepath.Join(dir, "b.go"), "new file")
		snap2 := takeSnapshot([]string{dir}, []string{".go"})
		if !snapshotChanged(snap1, snap2) {
			t.Error("expected change after file addition, but none detected")
		}
	})

	t.Run("change detected when file removed", func(t *testing.T) {
		extra := filepath.Join(dir, "c.go")
		writeFile(t, extra, "temp")
		snapWithExtra := takeSnapshot([]string{dir}, []string{".go"})

		os.Remove(extra)
		snapWithout := takeSnapshot([]string{dir}, []string{".go"})
		if !snapshotChanged(snapWithExtra, snapWithout) {
			t.Error("expected change after file removal, but none detected")
		}
	})
}

func TestExtensionFiltering(t *testing.T) {
	dir := t.TempDir()

	writeFile(t, filepath.Join(dir, "main.go"), "package main")
	writeFile(t, filepath.Join(dir, "notes.txt"), "notes")
	writeFile(t, filepath.Join(dir, "style.css"), "body{}")

	t.Run("only go files trigger change", func(t *testing.T) {
		snap1 := takeSnapshot([]string{dir}, []string{".go"})

		// Modify a .txt file — should NOT appear as a change.
		time.Sleep(50 * time.Millisecond)
		writeFile(t, filepath.Join(dir, "notes.txt"), "updated notes")

		snap2 := takeSnapshot([]string{dir}, []string{".go"})
		if snapshotChanged(snap1, snap2) {
			t.Error("change in .txt file should not be detected with .go filter")
		}

		// Modify a .go file — should trigger.
		time.Sleep(50 * time.Millisecond)
		writeFile(t, filepath.Join(dir, "main.go"), "package main // updated")

		snap3 := takeSnapshot([]string{dir}, []string{".go"})
		if !snapshotChanged(snap1, snap3) {
			t.Error("change in .go file should be detected")
		}
	})
}

func TestIgnoredDirectories(t *testing.T) {
	dir := t.TempDir()

	// Create files in ignored directories.
	gitDir := filepath.Join(dir, ".git")
	os.MkdirAll(gitDir, 0o755)
	writeFile(t, filepath.Join(gitDir, "config"), "git config")

	nodeDir := filepath.Join(dir, "node_modules")
	os.MkdirAll(nodeDir, 0o755)
	writeFile(t, filepath.Join(nodeDir, "pkg.js"), "module.exports = {}")

	pycacheDir := filepath.Join(dir, "__pycache__")
	os.MkdirAll(pycacheDir, 0o755)
	writeFile(t, filepath.Join(pycacheDir, "mod.pyc"), "bytecode")

	// Create a normal file.
	writeFile(t, filepath.Join(dir, "main.go"), "package main")

	snap := takeSnapshot([]string{dir}, nil)

	if len(snap) != 1 {
		t.Errorf("expected 1 file (main.go only), got %d files:", len(snap))
		for p := range snap {
			t.Errorf("  %s", p)
		}
	}
}

func TestDebounce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tests use Unix commands")
	}

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "main.go"), "package main")

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	cfg := Config{
		Command:    []string{"echo", "hello"},
		Interval:   100 * time.Millisecond,
		Debounce:   200 * time.Millisecond,
		Paths:      []string{dir},
		Extensions: []string{".go"},
		MaxLines:   50,
		GoTKConfig: config.Default(),
	}

	// Start watch in background.
	done := make(chan error, 1)
	go func() {
		done <- Run(ctx, cfg)
	}()

	// Wait for initial run.
	time.Sleep(150 * time.Millisecond)

	// Rapid batch of changes — should debounce into one re-run.
	for i := 0; i < 5; i++ {
		writeFile(t, filepath.Join(dir, "main.go"), fmt.Sprintf("package main // v%d", i))
		time.Sleep(20 * time.Millisecond)
	}

	// Wait for context to expire.
	err := <-done
	if err != nil && err != context.DeadlineExceeded {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRunCommandExecution(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("tests use Unix commands")
	}

	// Test that Run actually executes the command and returns when ctx is cancelled.
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "test.go"), "package test")

	cfg := Config{
		Command:    []string{"echo", "watch-test-output"},
		Interval:   100 * time.Millisecond,
		Debounce:   50 * time.Millisecond,
		Paths:      []string{dir},
		Extensions: []string{".go"},
		MaxLines:   50,
		GoTKConfig: config.Default(),
	}

	err := Run(ctx, cfg)
	if err != nil && err != context.DeadlineExceeded {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestIsIgnoredDir(t *testing.T) {
	cases := []struct {
		name     string
		expected bool
	}{
		{".git", true},
		{".hg", true},
		{".svn", true},
		{"node_modules", true},
		{"vendor", true},
		{"__pycache__", true},
		{".hidden", true},
		{"src", false},
		{"internal", false},
		{"cmd", false},
		{".", false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := isIgnoredDir(tc.name)
			if got != tc.expected {
				t.Errorf("isIgnoredDir(%q) = %v, want %v", tc.name, got, tc.expected)
			}
		})
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("failed to write file %s: %v", path, err)
	}
}
