package bench

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/paths"
)

// SnapshotFileName is the filename used inside paths.DataDir() for the
// per-filter snapshot consumed by `gotk dashboard`.
const SnapshotFileName = "perfilter.json"

// FilterSnapshot is one row in a PerFilterSnapshot — totals aggregated
// across every fixture for a single filter.
type FilterSnapshot struct {
	FilterName    string  `json:"filter"`
	BytesBefore   int     `json:"bytes_before"`
	BytesAfter    int     `json:"bytes_after"`
	Reduction     float64 `json:"reduction_pct"`
	DurationMicro int64   `json:"duration_us"`
	Invocations   int     `json:"invocations"` // number of fixtures the filter ran on
}

// PerFilterSnapshot is the on-disk shape consumed by the dashboard.
// Aggregating over every fixture gives a stable per-filter number even
// though fixture coverage varies by cmd type.
type PerFilterSnapshot struct {
	GeneratedAt string           `json:"generated_at"`
	Filters     []FilterSnapshot `json:"filters"`
}

// SnapshotPath returns the canonical snapshot path under paths.DataDir().
// Empty string if no data dir is available (e.g. in CI without HOME) — the
// caller should fall back to the cwd in that case.
func SnapshotPath() string {
	if dir, ok := paths.DataDir(); ok {
		return filepath.Join(dir, SnapshotFileName)
	}
	return ""
}

// BuildPerFilterSnapshot runs MeasureFilters over every built-in fixture
// and aggregates the contributions by filter name. The result is ranked by
// total bytes saved (descending), which is what the dashboard renders.
func BuildPerFilterSnapshot(cfg *config.Config) PerFilterSnapshot {
	totals := make(map[string]*FilterSnapshot)
	for _, f := range AllFixtureInputs() {
		for _, c := range MeasureFilters(cfg, f.Input, f.CmdType) {
			t, ok := totals[c.FilterName]
			if !ok {
				t = &FilterSnapshot{FilterName: c.FilterName}
				totals[c.FilterName] = t
			}
			t.BytesBefore += c.BytesBefore
			t.BytesAfter += c.BytesAfter
			t.DurationMicro += c.Duration.Microseconds()
			t.Invocations++
		}
	}

	rows := make([]FilterSnapshot, 0, len(totals))
	for _, t := range totals {
		if t.BytesBefore > 0 {
			t.Reduction = float64(t.BytesBefore-t.BytesAfter) / float64(t.BytesBefore) * 100
		}
		rows = append(rows, *t)
	}
	// Rank by bytes saved (BytesBefore-BytesAfter) descending so the
	// dashboard shows the most impactful filters first.
	sort.Slice(rows, func(i, j int) bool {
		si := rows[i].BytesBefore - rows[i].BytesAfter
		sj := rows[j].BytesBefore - rows[j].BytesAfter
		if si != sj {
			return si > sj
		}
		return rows[i].FilterName < rows[j].FilterName
	})

	return PerFilterSnapshot{
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Filters:     rows,
	}
}

// WritePerFilterSnapshot serializes the snapshot to path atomically
// (temp file + rename) so a concurrent reader never sees a partial file.
func WritePerFilterSnapshot(path string, snap PerFilterSnapshot) error {
	if path == "" {
		return fmt.Errorf("per-filter snapshot: empty path")
	}
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(snap, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(dir, "perfilter-*.json.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return os.Rename(tmpPath, path)
}

// ReadPerFilterSnapshot loads a snapshot from disk. A missing file is
// reported as os.ErrNotExist so callers (the dashboard) can render an
// empty-state hint distinct from a corrupted file.
func ReadPerFilterSnapshot(path string) (PerFilterSnapshot, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return PerFilterSnapshot{}, err
	}
	var snap PerFilterSnapshot
	if err := json.Unmarshal(data, &snap); err != nil {
		return PerFilterSnapshot{}, err
	}
	return snap, nil
}
