package measure

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoggerWriteAndRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "sub", "test.jsonl")

	l, err := NewLogger(path, "test-session")
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}

	e := Entry{
		Command:      "grep -rn foo .",
		CommandType:  "grep",
		RawBytes:     1000,
		CleanBytes:   200,
		RawTokens:    250,
		CleanTokens:  50,
		TokensSaved:  200,
		ReductionPct: 80.0,
		LinesRaw:     50,
		LinesClean:   10,
		QualityScore: 100.0,
		Mode:         "balanced",
		Source:       "cli",
	}

	if err := l.Log(e); err != nil {
		t.Fatalf("Log: %v", err)
	}

	e2 := Entry{
		Command:     "ls -la",
		CommandType: "ls",
		RawBytes:    500,
		CleanBytes:  100,
		Source:      "cli",
	}
	if err := l.Log(e2); err != nil {
		t.Fatalf("Log: %v", err)
	}

	if err := l.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	entries, err := ReadEntries(path)
	if err != nil {
		t.Fatalf("ReadEntries: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	if entries[0].Command != "grep -rn foo ." {
		t.Errorf("entry[0].Command = %q, want %q", entries[0].Command, "grep -rn foo .")
	}
	if entries[0].SessionID != "test-session" {
		t.Errorf("entry[0].SessionID = %q, want %q", entries[0].SessionID, "test-session")
	}
	if entries[0].Timestamp == "" {
		t.Error("entry[0].Timestamp is empty")
	}
}

func TestReadEntriesSince(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.jsonl")

	l, err := NewLogger(path, "s1")
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}

	old := Entry{
		Timestamp: "2025-01-01T00:00:00Z",
		Command:   "old",
		Source:    "cli",
	}
	recent := Entry{
		Timestamp: "2025-06-15T12:00:00Z",
		Command:   "recent",
		Source:    "cli",
	}

	_ = l.Log(old)
	_ = l.Log(recent)
	_ = l.Close()

	since, _ := time.Parse(time.RFC3339, "2025-06-01T00:00:00Z")
	entries, err := ReadEntriesSince(path, since)
	if err != nil {
		t.Fatalf("ReadEntriesSince: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
	if entries[0].Command != "recent" {
		t.Errorf("got command %q, want %q", entries[0].Command, "recent")
	}
}

func TestSessionID(t *testing.T) {
	id := SessionID()
	if id == "" {
		t.Error("SessionID returned empty string")
	}
}

func TestReadEntriesMissingFile(t *testing.T) {
	_, err := ReadEntries("/nonexistent/path.jsonl")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestDefaultLogPath(t *testing.T) {
	p := DefaultLogPath()
	if p == "" {
		t.Error("DefaultLogPath returned empty")
	}
	// Should end with measure.jsonl
	if filepath.Base(p) != "measure.jsonl" {
		t.Errorf("DefaultLogPath = %q, want basename measure.jsonl", p)
	}
}

func TestLoggerRotation(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "rotate.jsonl")

	// Use a tiny max size to trigger rotation quickly
	l, err := NewLogger(path, "s1", 500)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}

	// Write enough entries to exceed 500 bytes
	for i := 0; i < 20; i++ {
		_ = l.Log(Entry{
			Command:     fmt.Sprintf("cmd-%d", i),
			CommandType: "generic",
			RawBytes:    1000,
			CleanBytes:  200,
			Source:      "cli",
		})
	}
	_ = l.Close()

	// File should be smaller than if all 20 entries were kept
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("Stat: %v", err)
	}

	entries, err := ReadEntries(path)
	if err != nil {
		t.Fatalf("ReadEntries: %v", err)
	}

	// Rotation should have kept some entries but not all 20
	if len(entries) >= 20 {
		t.Errorf("expected rotation to discard old entries, got %d", len(entries))
	}
	if len(entries) == 0 {
		t.Error("rotation discarded all entries")
	}

	// The most recent entry should be the last one written
	last := entries[len(entries)-1]
	if last.Command != "cmd-19" {
		t.Errorf("last entry = %q, want cmd-19", last.Command)
	}

	t.Logf("rotation: 20 entries written, %d kept, file size %d bytes", len(entries), info.Size())
}

func TestLoggerNoRotationWhenUnlimited(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "norotate.jsonl")

	// maxSize=0 means unlimited
	l, err := NewLogger(path, "s1", 0)
	if err != nil {
		t.Fatalf("NewLogger: %v", err)
	}

	for i := 0; i < 10; i++ {
		_ = l.Log(Entry{
			Command:     fmt.Sprintf("cmd-%d", i),
			CommandType: "generic",
			Source:      "cli",
		})
	}
	_ = l.Close()

	entries, err := ReadEntries(path)
	if err != nil {
		t.Fatalf("ReadEntries: %v", err)
	}
	if len(entries) != 10 {
		t.Errorf("got %d entries, want 10 (no rotation)", len(entries))
	}
}

func TestLoggerCreatesParentDirs(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a", "b", "c", "test.jsonl")

	l, err := NewLogger(path, "s1")
	if err != nil {
		t.Fatalf("NewLogger with nested dirs: %v", err)
	}
	_ = l.Close()

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("expected file to be created")
	}
}
