package update

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// withFakeHome redirects HOME for one test, so cachePath() resolves
// under a temp directory we can inspect and clean up.
func withFakeHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("GOTK_NO_UPDATE_CHECK", "")
	t.Setenv("CI", "")
	t.Setenv("GITHUB_ACTIONS", "")
	return home
}

func writeCacheFor(t *testing.T, latest string, age time.Duration) {
	t.Helper()
	entry := cacheEntry{
		CheckedAt: time.Now().Add(-age),
		Latest:    latest,
	}
	data, _ := json.Marshal(entry)
	path := cachePath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
}

func TestNotifyIfUpdate_NewerAvailable(t *testing.T) {
	withFakeHome(t)
	writeCacheFor(t, "v1.6.0", 1*time.Hour)

	notice := NotifyIfUpdate("v1.5.0", true)
	if notice == "" {
		t.Fatalf("expected a notice when cache shows a newer version")
	}
	for _, want := range []string{"v1.5.0", "v1.6.0", "gotk update"} {
		if !strings.Contains(notice, want) {
			t.Errorf("notice missing %q: %s", want, notice)
		}
	}
}

func TestNotifyIfUpdate_SameVersion(t *testing.T) {
	withFakeHome(t)
	writeCacheFor(t, "v1.5.0", 1*time.Hour)

	if notice := NotifyIfUpdate("v1.5.0", true); notice != "" {
		t.Errorf("no notice expected at same version, got: %s", notice)
	}
}

func TestNotifyIfUpdate_OlderCachedVersion(t *testing.T) {
	// Cache can only go stale downward in practice, but the comparison
	// must handle it: a locally newer build should never show a notice.
	withFakeHome(t)
	writeCacheFor(t, "v1.4.0", 1*time.Hour)

	if notice := NotifyIfUpdate("v1.5.0", true); notice != "" {
		t.Errorf("no notice expected when local > cached latest, got: %s", notice)
	}
}

func TestNotifyIfUpdate_NoCacheReturnsEmpty(t *testing.T) {
	withFakeHome(t)
	// No cache file written.
	if notice := NotifyIfUpdate("v1.5.0", true); notice != "" {
		t.Errorf("no notice expected without cache, got: %s", notice)
	}
}

func TestNotifyIfUpdate_DevBuildSilent(t *testing.T) {
	withFakeHome(t)
	writeCacheFor(t, "v1.6.0", 1*time.Hour)

	if notice := NotifyIfUpdate("dev", true); notice != "" {
		t.Errorf("dev build should never show a notice, got: %s", notice)
	}
	if notice := NotifyIfUpdate("", true); notice != "" {
		t.Errorf("empty version should never show a notice, got: %s", notice)
	}
}

func TestNotifyIfUpdate_NotTTYSilent(t *testing.T) {
	withFakeHome(t)
	writeCacheFor(t, "v1.6.0", 1*time.Hour)

	if notice := NotifyIfUpdate("v1.5.0", false); notice != "" {
		t.Errorf("non-TTY stderr should suppress notice, got: %s", notice)
	}
}

func TestNotifyIfUpdate_CIEnvSilences(t *testing.T) {
	withFakeHome(t)
	writeCacheFor(t, "v1.6.0", 1*time.Hour)

	t.Setenv("CI", "true")
	if notice := NotifyIfUpdate("v1.5.0", true); notice != "" {
		t.Errorf("CI=true should silence notice, got: %s", notice)
	}
}

func TestNotifyIfUpdate_GitHubActionsSilences(t *testing.T) {
	withFakeHome(t)
	writeCacheFor(t, "v1.6.0", 1*time.Hour)

	t.Setenv("GITHUB_ACTIONS", "true")
	if notice := NotifyIfUpdate("v1.5.0", true); notice != "" {
		t.Errorf("GITHUB_ACTIONS=true should silence notice, got: %s", notice)
	}
}

func TestNotifyIfUpdate_OptOutEnvVar(t *testing.T) {
	withFakeHome(t)
	writeCacheFor(t, "v1.6.0", 1*time.Hour)

	t.Setenv("GOTK_NO_UPDATE_CHECK", "1")
	if notice := NotifyIfUpdate("v1.5.0", true); notice != "" {
		t.Errorf("GOTK_NO_UPDATE_CHECK=1 should silence notice, got: %s", notice)
	}
}

func TestNotifyIfUpdate_StripVPrefix(t *testing.T) {
	// Either side of the comparison may or may not have the `v` prefix;
	// both shapes must produce consistent results.
	withFakeHome(t)
	writeCacheFor(t, "1.6.0", 1*time.Hour) // cache entry without 'v'

	notice := NotifyIfUpdate("v1.5.0", true)
	if notice == "" {
		t.Errorf("expected notice regardless of v-prefix normalization")
	}
}

func TestNotifyIfUpdate_StaleCacheStillUsed(t *testing.T) {
	// A stale cache (>24h) should still show the notice; refresh happens
	// asynchronously and shouldn't gate the notice.
	withFakeHome(t)
	writeCacheFor(t, "v1.6.0", 48*time.Hour)

	if notice := NotifyIfUpdate("v1.5.0", true); notice == "" {
		t.Errorf("stale cache should still produce a notice")
	}
}

func TestWriteCache_AtomicRenameLeavesNoTempFile(t *testing.T) {
	home := withFakeHome(t)
	path := cachePath()
	entry := cacheEntry{CheckedAt: time.Now(), Latest: "v1.5.0"}

	if err := writeCache(path, entry); err != nil {
		t.Fatal(err)
	}

	// Read back
	got, err := readCache(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Latest != entry.Latest {
		t.Errorf("round-trip Latest: got %q want %q", got.Latest, entry.Latest)
	}

	// No leftover *.tmp files in the cache directory.
	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".tmp") {
			t.Errorf("temp file leaked: %s/%s", home, e.Name())
		}
	}
}

func TestStripV(t *testing.T) {
	cases := map[string]string{
		"v1.2.3": "1.2.3",
		"V1.2.3": "1.2.3",
		"1.2.3":  "1.2.3",
		"":       "",
	}
	for in, want := range cases {
		if got := stripV(in); got != want {
			t.Errorf("stripV(%q) = %q want %q", in, got, want)
		}
	}
}
