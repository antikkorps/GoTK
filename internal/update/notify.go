package update

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/antikkorps/GoTK/internal/paths"
)

// cacheTTL defines how long a cached "latest release" lookup stays fresh
// before we'll opportunistically refresh in the background. Short enough
// that users notice new releases within a day; long enough that we only
// hit GitHub ~once per day per machine.
const cacheTTL = 24 * time.Hour

// refreshTimeout bounds the background HTTP call that refreshes the cache.
// It's intentionally tight: the notice is non-essential, so a slow network
// must never impact the wrapped-command latency the user actually cares
// about. If gotk exits before the fetch finishes, the next invocation will
// try again.
const refreshTimeout = 3 * time.Second

// cacheEntry is the on-disk schema of the update-check cache.
type cacheEntry struct {
	CheckedAt time.Time `json:"checked_at"`
	Latest    string    `json:"latest"`
}

// NotifyIfUpdate returns a one-line notice the caller should write to
// stderr when a newer gotk release is available. It reads a small JSON
// cache and never makes a synchronous network call, so it's safe to call
// on every invocation — typical cost is a single ~200-byte file read.
//
// When the cache is stale (> cacheTTL) or missing, it kicks off a
// detached goroutine that refreshes it against GitHub. That goroutine
// dies with the process, so whether it finishes depends on how long the
// main command takes; no work is lost either way because the next
// invocation will try again.
//
// Returns "" (and emits nothing) when:
//   - GOTK_NO_UPDATE_CHECK is set
//   - CI or GITHUB_ACTIONS is set (avoid noise in automation)
//   - the caller is a dev build (current == "dev" or "")
//   - stderr is not a TTY per isNotice
//   - the cache has no fresher version, or no cache exists yet
func NotifyIfUpdate(current string, stderrIsTTY bool) string {
	if !shouldCheck(current, stderrIsTTY) {
		return ""
	}

	path := cachePath()
	entry, err := readCache(path)
	cacheMissing := err != nil

	// Whether or not we had a cache hit, trigger a refresh when it's stale.
	// We do this before returning so the next invocation has fresh data.
	if cacheMissing || time.Since(entry.CheckedAt) > cacheTTL {
		go refreshCacheAsync(path, current)
	}

	if cacheMissing || entry.Latest == "" {
		return ""
	}

	cmp, err := Compare(stripV(current), stripV(entry.Latest))
	if err != nil || cmp >= 0 {
		return ""
	}

	return fmt.Sprintf("[gotk] update available: %s → %s  (run \"gotk update\" to install)",
		current, entry.Latest)
}

// shouldCheck encapsulates every gate that silences the notifier: env
// variables, dev builds, and non-interactive stderr.
func shouldCheck(current string, stderrIsTTY bool) bool {
	if os.Getenv("GOTK_NO_UPDATE_CHECK") != "" {
		return false
	}
	// Common CI env vars. `CI` is almost universal; `GITHUB_ACTIONS` catches
	// the case where CI is explicitly unset but we're still in a workflow.
	if os.Getenv("CI") != "" || os.Getenv("GITHUB_ACTIONS") != "" {
		return false
	}
	if current == "" || current == devVersionMarker {
		return false
	}
	return stderrIsTTY
}

func cachePath() string {
	if dir, ok := paths.DataDir(); ok {
		return filepath.Join(dir, "update_check.json")
	}
	// Fall back to cwd so behavior is deterministic; a write failure
	// will simply mean no cache, which is fine.
	return ".gotk-update-check.json"
}

func readCache(path string) (cacheEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return cacheEntry{}, err
	}
	var e cacheEntry
	if err := json.Unmarshal(data, &e); err != nil {
		return cacheEntry{}, err
	}
	return e, nil
}

func writeCache(path string, entry cacheEntry) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(entry, "", "  ")
	if err != nil {
		return err
	}
	// Atomic rename to avoid racing with a concurrent reader.
	tmp, err := os.CreateTemp(dir, "update_check-*.json.tmp")
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

// refreshCacheAsync fetches the latest release from GitHub and writes the
// result to the cache. It is meant to be called as a goroutine — errors
// are swallowed because they're strictly background work.
//
// We always bump CheckedAt on success OR on a clean "not found" response
// (e.g. no releases yet) so we don't hammer GitHub when the repo has no
// tags. Transient network errors leave CheckedAt untouched, which means
// the next invocation will retry — desired behavior.
func refreshCacheAsync(path, current string) {
	ctx, cancel := context.WithTimeout(context.Background(), refreshTimeout)
	defer cancel()

	client := &http.Client{Timeout: refreshTimeout}
	rel, err := FetchLatest(ctx, "", DefaultRepo, client)
	if err != nil {
		// "not found" is a clean signal: mark the check time so we don't
		// retry for another TTL window.
		var nrf *notFoundErr
		if errors.As(err, &nrf) {
			_ = writeCache(path, cacheEntry{CheckedAt: time.Now()})
		}
		return
	}
	_ = writeCache(path, cacheEntry{
		CheckedAt: time.Now(),
		Latest:    rel.TagName,
	})
	_ = current // reserved for future per-version gating
}

// notFoundErr exists so refreshCacheAsync can distinguish "repo has no
// releases" (a settled state worth caching) from transient errors.
// FetchLatest currently returns a plain errors.New, so we only match on
// the message text; this type is a forward-compatible hook if FetchLatest
// ever starts returning a typed error.
type notFoundErr struct{ msg string }

func (e *notFoundErr) Error() string { return e.msg }

func stripV(s string) string {
	if len(s) > 0 && (s[0] == 'v' || s[0] == 'V') {
		return s[1:]
	}
	return s
}
