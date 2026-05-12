package bench

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/antikkorps/GoTK/internal/config"
)

func TestBuildPerFilterSnapshot_RanksByBytesSaved(t *testing.T) {
	cfg := config.Default()
	snap := BuildPerFilterSnapshot(cfg)

	if len(snap.Filters) == 0 {
		t.Fatal("expected at least one filter in snapshot")
	}
	if snap.GeneratedAt == "" {
		t.Error("GeneratedAt must be set")
	}
	for i := 1; i < len(snap.Filters); i++ {
		prev := snap.Filters[i-1].BytesBefore - snap.Filters[i-1].BytesAfter
		cur := snap.Filters[i].BytesBefore - snap.Filters[i].BytesAfter
		if prev < cur {
			t.Errorf("rows not sorted: %s (%d saved) before %s (%d saved)",
				snap.Filters[i-1].FilterName, prev, snap.Filters[i].FilterName, cur)
		}
	}
	// Some filters can legitimately grow the output (Summarize adds a
	// summary block; CompressGrepOutput regroups headers). Don't assert
	// BytesAfter <= BytesBefore globally — only invocation count must be sane.
	for _, f := range snap.Filters {
		if f.Invocations <= 0 {
			t.Errorf("filter %q: Invocations should be positive, got %d", f.FilterName, f.Invocations)
		}
	}
}

func TestWriteReadPerFilterSnapshot_Roundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "snap.json")

	want := PerFilterSnapshot{
		GeneratedAt: "2026-05-12T20:00:00Z",
		Filters: []FilterSnapshot{
			{FilterName: "StripANSI", BytesBefore: 1000, BytesAfter: 800, Reduction: 20, DurationMicro: 50, Invocations: 21},
			{FilterName: "Dedup", BytesBefore: 800, BytesAfter: 700, Reduction: 12.5, DurationMicro: 30, Invocations: 21},
		},
	}
	if err := WritePerFilterSnapshot(path, want); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := ReadPerFilterSnapshot(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if got.GeneratedAt != want.GeneratedAt {
		t.Errorf("GeneratedAt: want %q, got %q", want.GeneratedAt, got.GeneratedAt)
	}
	if len(got.Filters) != len(want.Filters) {
		t.Fatalf("len mismatch: want %d, got %d", len(want.Filters), len(got.Filters))
	}
	for i := range want.Filters {
		if got.Filters[i] != want.Filters[i] {
			t.Errorf("row %d: want %+v, got %+v", i, want.Filters[i], got.Filters[i])
		}
	}
}

func TestReadPerFilterSnapshot_MissingFileIsNotExist(t *testing.T) {
	_, err := ReadPerFilterSnapshot(filepath.Join(t.TempDir(), "nope.json"))
	if err == nil {
		t.Fatal("expected error reading missing file")
	}
	if !os.IsNotExist(err) {
		t.Errorf("expected os.IsNotExist, got %v", err)
	}
}

func TestWritePerFilterSnapshot_EmptyPath(t *testing.T) {
	if err := WritePerFilterSnapshot("", PerFilterSnapshot{}); err == nil {
		t.Error("expected error on empty path")
	}
}
