package learn

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStoreWriteRead(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, StoreFileName)

	observations := []Observation{
		{Timestamp: "2024-01-01T00:00:00Z", Command: "go test", Normalized: "ok pkg <DURATION>", Level: 0, SessionID: "s1"},
		{Timestamp: "2024-01-01T00:00:00Z", Command: "go test", Normalized: "FAIL pkg", Level: 4, SessionID: "s1"},
	}

	if err := StoreWrite(path, observations); err != nil {
		t.Fatalf("StoreWrite: %v", err)
	}

	got, err := StoreRead(path)
	if err != nil {
		t.Fatalf("StoreRead: %v", err)
	}

	if len(got) != 2 {
		t.Fatalf("StoreRead: got %d observations, want 2", len(got))
	}

	if got[0].Command != "go test" {
		t.Errorf("got[0].Command = %q, want %q", got[0].Command, "go test")
	}
	if got[1].Level != 4 {
		t.Errorf("got[1].Level = %d, want 4", got[1].Level)
	}
}

func TestStoreReadNonExistent(t *testing.T) {
	got, err := StoreRead("/nonexistent/path.jsonl")
	if err != nil {
		t.Fatalf("StoreRead on non-existent file should return nil error, got: %v", err)
	}
	if got != nil {
		t.Fatalf("StoreRead on non-existent file should return nil, got %d entries", len(got))
	}
}

func TestStoreAppend(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, StoreFileName)

	batch1 := []Observation{
		{Timestamp: "2024-01-01T00:00:00Z", Command: "cmd1", Normalized: "line1", SessionID: "s1"},
	}
	batch2 := []Observation{
		{Timestamp: "2024-01-02T00:00:00Z", Command: "cmd2", Normalized: "line2", SessionID: "s2"},
	}

	if err := StoreWrite(path, batch1); err != nil {
		t.Fatalf("StoreWrite batch1: %v", err)
	}
	if err := StoreWrite(path, batch2); err != nil {
		t.Fatalf("StoreWrite batch2: %v", err)
	}

	got, err := StoreRead(path)
	if err != nil {
		t.Fatalf("StoreRead: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %d observations after 2 appends, want 2", len(got))
	}
}

func TestStoreStat(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, StoreFileName)

	// Empty store
	stats, err := StoreStat(path)
	if err != nil {
		t.Fatalf("StoreStat on empty: %v", err)
	}
	if stats.TotalObservations != 0 {
		t.Errorf("empty store: TotalObservations = %d, want 0", stats.TotalObservations)
	}

	// With data
	observations := []Observation{
		{Timestamp: "2024-01-01T00:00:00Z", Command: "cmd1", Normalized: "a", SessionID: "s1"},
		{Timestamp: "2024-01-02T00:00:00Z", Command: "cmd2", Normalized: "b", SessionID: "s2"},
		{Timestamp: "2024-01-03T00:00:00Z", Command: "cmd1", Normalized: "c", SessionID: "s1"},
	}
	if err := StoreWrite(path, observations); err != nil {
		t.Fatalf("StoreWrite: %v", err)
	}

	stats, err = StoreStat(path)
	if err != nil {
		t.Fatalf("StoreStat: %v", err)
	}
	if stats.TotalObservations != 3 {
		t.Errorf("TotalObservations = %d, want 3", stats.TotalObservations)
	}
	if stats.Sessions != 2 {
		t.Errorf("Sessions = %d, want 2", stats.Sessions)
	}
	if stats.Commands != 2 {
		t.Errorf("Commands = %d, want 2", stats.Commands)
	}
}

func TestStoreClear(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, StoreFileName)

	observations := []Observation{
		{Timestamp: "t", Command: "c", Normalized: "n", SessionID: "s"},
	}
	if err := StoreWrite(path, observations); err != nil {
		t.Fatalf("StoreWrite: %v", err)
	}

	if err := StoreClear(path); err != nil {
		t.Fatalf("StoreClear: %v", err)
	}

	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Errorf("store file should be deleted after clear")
	}

	// Clear on non-existent file should not error
	if err := StoreClear(path); err != nil {
		t.Errorf("StoreClear on non-existent file should not error: %v", err)
	}
}
