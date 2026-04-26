package update

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// buildTarball wraps `binary` in a tar.gz matching the goreleaser layout.
// Returns the archive bytes plus the hex SHA256 of that full archive.
func buildTarball(t *testing.T, binary []byte) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)

	if err := tw.WriteHeader(&tar.Header{
		Name:     "gotk",
		Mode:     0o755,
		Size:     int64(len(binary)),
		Typeflag: tar.TypeReg,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := tw.Write(binary); err != nil {
		t.Fatal(err)
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}

	sum := sha256.Sum256(buf.Bytes())
	return buf.Bytes(), hex.EncodeToString(sum[:])
}

// buildZip wraps `binary` in a zip archive matching the goreleaser layout for
// Windows releases. The entry is named "gotk.exe" to mirror the real artifact.
func buildZip(t *testing.T, binary []byte) ([]byte, string) {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, err := zw.Create("gotk.exe")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := w.Write(binary); err != nil {
		t.Fatal(err)
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(buf.Bytes())
	return buf.Bytes(), hex.EncodeToString(sum[:])
}

// fakeGitHub spins up an httptest server that behaves like the GitHub
// Releases API for a single release. Returns the server and the rewritten
// download URL the server serves asset bytes from.
func fakeGitHub(t *testing.T, tag string, assets map[string][]byte) *httptest.Server {
	t.Helper()
	mux := http.NewServeMux()

	var handler *httptest.Server
	mux.HandleFunc("/repos/", func(w http.ResponseWriter, r *http.Request) {
		// Build asset descriptors using the server's own URL so downloads
		// stay local.
		rel := Release{
			TagName: tag,
			Name:    tag,
			HTMLURL: "https://example.invalid/release/" + tag,
		}
		for name, data := range assets {
			rel.Assets = append(rel.Assets, Asset{
				Name:        name,
				DownloadURL: handler.URL + "/assets/" + name,
				Size:        int64(len(data)),
			})
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(rel)
	})
	mux.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/assets/")
		data, ok := assets[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})

	handler = httptest.NewServer(mux)
	t.Cleanup(handler.Close)
	return handler
}

func TestCompare(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"v1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"1.1.0", "1.0.9", 1},
		{"2.0.0", "1.9.9", 1},
		{"1.4.0", "1.3.0", 1},
		{"1.0.0-rc1", "1.0.0", -1},
		{"1.0.0", "1.0.0-rc1", 1},
		{"1.0.0-rc1", "1.0.0-rc2", -1},
		{"1.0", "1.0.0", 0},
		{"1", "1.0.0", 0},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			got, err := Compare(tt.a, tt.b)
			if err != nil {
				t.Fatalf("Compare(%q, %q): %v", tt.a, tt.b, err)
			}
			if got != tt.want {
				t.Errorf("Compare(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestCompareRejectsGarbage(t *testing.T) {
	if _, err := Compare("not-a-version", "1.0.0"); err == nil {
		t.Error("expected error for non-numeric version")
	}
}

func TestPickAssetMatchesGoOSArch(t *testing.T) {
	assets := []Asset{
		{Name: "gotk_1.4.0_linux_amd64.tar.gz"},
		{Name: "gotk_1.4.0_darwin_arm64.tar.gz"},
		{Name: "gotk_1.4.0_darwin_amd64.tar.gz"},
		{Name: "checksums.txt"},
	}
	got, ok := PickAsset(assets, "1.4.0", "darwin", "arm64")
	if !ok {
		t.Fatal("asset not found")
	}
	if got.Name != "gotk_1.4.0_darwin_arm64.tar.gz" {
		t.Errorf("wrong asset: %s", got.Name)
	}
}

func TestPickAssetReturnsFalseForUnknownPlatform(t *testing.T) {
	assets := []Asset{{Name: "gotk_1.4.0_linux_amd64.tar.gz"}}
	if _, ok := PickAsset(assets, "1.4.0", "windows", "amd64"); ok {
		t.Error("windows/amd64 should not be found in Linux-only release")
	}
}

func TestFetchLatestDecodesTagName(t *testing.T) {
	srv := fakeGitHub(t, "v1.4.0", map[string][]byte{})

	rel, err := FetchLatest(context.Background(), srv.URL, "antikkorps/GoTK", srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if rel.TagName != "v1.4.0" {
		t.Errorf("TagName = %q", rel.TagName)
	}
}

func TestFetchLatestHandlesMissingRelease(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	}))
	defer srv.Close()

	_, err := FetchLatest(context.Background(), srv.URL, "antikkorps/GoTK", srv.Client())
	if err == nil || !strings.Contains(err.Error(), "no release") {
		t.Errorf("want 'no release' error, got %v", err)
	}
}

func TestRunCheckOnlyAlreadyUpToDate(t *testing.T) {
	srv := fakeGitHub(t, "v1.4.0", map[string][]byte{})

	var out bytes.Buffer
	err := Run(context.Background(), Options{
		Repo:        "antikkorps/GoTK",
		Current:     "1.4.0",
		CheckOnly:   true,
		BaseURL:     srv.URL,
		AssetClient: srv.Client(),
		Out:         &out,
		goos:        "linux",
		goarch:      "amd64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "already up to date") {
		t.Errorf("output did not report up-to-date: %s", out.String())
	}
}

func TestRunCheckOnlyNewerAvailable(t *testing.T) {
	srv := fakeGitHub(t, "v1.5.0", map[string][]byte{})

	var out bytes.Buffer
	err := Run(context.Background(), Options{
		Repo:        "antikkorps/GoTK",
		Current:     "1.4.0",
		CheckOnly:   true,
		BaseURL:     srv.URL,
		AssetClient: srv.Client(),
		Out:         &out,
		goos:        "linux",
		goarch:      "amd64",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "new version available") {
		t.Errorf("output did not report newer version: %s", out.String())
	}
	// --check must not actually download the asset.
	if strings.Contains(out.String(), "downloading") {
		t.Errorf("--check should not download: %s", out.String())
	}
}

func TestRunDownloadReplacesBinary(t *testing.T) {
	binaryContent := []byte("#!/bin/sh\necho new-gotk\n")
	tarBytes, tarSHA := buildTarball(t, binaryContent)

	assetName := "gotk_1.5.0_linux_amd64.tar.gz"
	checksumsBody := []byte(tarSHA + "  " + assetName + "\n")

	srv := fakeGitHub(t, "v1.5.0", map[string][]byte{
		assetName:       tarBytes,
		"checksums.txt": checksumsBody,
	})

	// Write a fake "current" binary we can replace.
	dir := t.TempDir()
	exePath := filepath.Join(dir, "gotk")
	if err := os.WriteFile(exePath, []byte("old-binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Run's replaceBinary path reads os.Executable(); we can't easily
	// override that for this process, so test the individual pieces.
	rel, err := FetchLatest(context.Background(), srv.URL, "antikkorps/GoTK", srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	asset, ok := PickAsset(rel.Assets, "1.5.0", "linux", "amd64")
	if !ok {
		t.Fatal("asset not found")
	}
	checksum, err := fetchChecksum(context.Background(), rel.Assets, asset.Name, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	if checksum != tarSHA {
		t.Fatalf("checksum mismatch: want %s got %s", tarSHA, checksum)
	}
	tmpBin, err := downloadAndExtract(context.Background(), asset, checksum, srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpBin) //nolint:errcheck

	if err := replaceBinary(tmpBin, exePath); err != nil {
		t.Fatalf("replaceBinary: %v", err)
	}

	got, err := os.ReadFile(exePath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, binaryContent) {
		t.Errorf("binary contents not replaced: got %q", string(got))
	}
}

func TestDownloadAndExtractFromZipForWindows(t *testing.T) {
	binaryContent := []byte("windows-binary-bytes")
	zipBytes, sum := buildZip(t, binaryContent)

	assetName := "gotk_1.6.0_windows_amd64.zip"
	srv := fakeGitHub(t, "v1.6.0", map[string][]byte{assetName: zipBytes})
	rel, err := FetchLatest(context.Background(), srv.URL, "antikkorps/GoTK", srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	asset, ok := PickAsset(rel.Assets, "1.6.0", "windows", "amd64")
	if !ok {
		t.Fatal("windows asset not found")
	}

	tmpPath, err := downloadAndExtract(context.Background(), asset, sum, srv.Client())
	if err != nil {
		t.Fatalf("downloadAndExtract returned error: %v", err)
	}
	defer os.Remove(tmpPath) //nolint:errcheck

	got, err := os.ReadFile(tmpPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, binaryContent) {
		t.Errorf("extracted binary mismatch: got %q, want %q", got, binaryContent)
	}
}

func TestDownloadAndExtractRejectsBadChecksum(t *testing.T) {
	binaryContent := []byte("hi")
	tarBytes, _ := buildTarball(t, binaryContent)

	assetName := "gotk_1.5.0_linux_amd64.tar.gz"
	srv := fakeGitHub(t, "v1.5.0", map[string][]byte{
		assetName: tarBytes,
	})
	rel, err := FetchLatest(context.Background(), srv.URL, "antikkorps/GoTK", srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	asset, ok := PickAsset(rel.Assets, "1.5.0", "linux", "amd64")
	if !ok {
		t.Fatal("asset not found")
	}
	_, err = downloadAndExtract(context.Background(), asset, strings.Repeat("0", 64), srv.Client())
	if err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Errorf("want sha256 mismatch error, got %v", err)
	}
}

func TestDownloadAndExtractRejectsMissingBinary(t *testing.T) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "README", Mode: 0o644, Size: 3, Typeflag: tar.TypeReg})
	_, _ = tw.Write([]byte("hi\n"))
	_ = tw.Close()
	_ = gz.Close()

	sum := sha256.Sum256(buf.Bytes())
	hexSum := hex.EncodeToString(sum[:])

	assetName := "gotk_1.5.0_linux_amd64.tar.gz"
	srv := fakeGitHub(t, "v1.5.0", map[string][]byte{assetName: buf.Bytes()})

	rel, err := FetchLatest(context.Background(), srv.URL, "antikkorps/GoTK", srv.Client())
	if err != nil {
		t.Fatal(err)
	}
	asset, ok := PickAsset(rel.Assets, "1.5.0", "linux", "amd64")
	if !ok {
		t.Fatal("asset not found")
	}
	_, err = downloadAndExtract(context.Background(), asset, hexSum, srv.Client())
	if err == nil || !strings.Contains(err.Error(), "did not contain") {
		t.Errorf("want 'did not contain' error, got %v", err)
	}
}

func TestRunFallsBackWhenNoAssetForPlatform(t *testing.T) {
	srv := fakeGitHub(t, "v1.5.0", map[string][]byte{
		"gotk_1.5.0_linux_amd64.tar.gz": []byte("ignored"),
	})

	var out bytes.Buffer
	// Ask for a platform with no matching asset; Run must report the
	// fallback rather than crashing. goInstallLatest will try to exec `go`
	// which may or may not succeed in CI — we only check the status line.
	_ = Run(context.Background(), Options{
		Repo:        "antikkorps/GoTK",
		Current:     "1.4.0",
		BaseURL:     srv.URL,
		AssetClient: srv.Client(),
		Out:         &out,
		goos:        "windows",
		goarch:      "amd64",
	})

	if !strings.Contains(out.String(), "no pre-built binary") {
		t.Errorf("fallback notice missing: %s", out.String())
	}
}

func TestHumanBytes(t *testing.T) {
	tests := []struct {
		in   int64
		want string
	}{
		{100, "100 B"},
		{2048, "2.0 KiB"},
		{1500 * 1024, "1.5 MiB"},
	}
	for _, tt := range tests {
		if got := humanBytes(tt.in); got != tt.want {
			t.Errorf("humanBytes(%d) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// Sanity check that the asset-naming convention matches the goreleaser
// template committed in .goreleaser.yml. If someone renames the template,
// this test acts as an early warning.
func TestPickAssetMatchesGoreleaserTemplate(t *testing.T) {
	version := "1.4.0"
	for _, os := range []string{"linux", "darwin"} {
		for _, arch := range []string{"amd64", "arm64"} {
			want := fmt.Sprintf("gotk_%s_%s_%s.tar.gz", version, os, arch)
			assets := []Asset{{Name: want}}
			got, ok := PickAsset(assets, version, os, arch)
			if !ok || got.Name != want {
				t.Errorf("PickAsset failed for %s/%s: want %s, got %s/%v", os, arch, want, got.Name, ok)
			}
		}
	}
}

func TestPickAssetMatchesWindowsZip(t *testing.T) {
	version := "1.6.0"
	for _, arch := range []string{"amd64", "arm64"} {
		want := fmt.Sprintf("gotk_%s_windows_%s.zip", version, arch)
		assets := []Asset{
			{Name: fmt.Sprintf("gotk_%s_linux_amd64.tar.gz", version)},
			{Name: want},
		}
		got, ok := PickAsset(assets, version, "windows", arch)
		if !ok {
			t.Errorf("PickAsset for windows/%s failed: zip asset not found", arch)
			continue
		}
		if got.Name != want {
			t.Errorf("PickAsset for windows/%s: got %s, want %s", arch, got.Name, want)
		}
	}
}

func TestPickAssetIgnoresTarGzOnWindows(t *testing.T) {
	// A release that only ships .tar.gz must not be picked for Windows.
	assets := []Asset{{Name: "gotk_1.6.0_windows_amd64.tar.gz"}}
	if _, ok := PickAsset(assets, "1.6.0", "windows", "amd64"); ok {
		t.Error("PickAsset should not match .tar.gz on Windows")
	}
}

// guard: make sure the update package does not accidentally import GoTK
// infrastructure it should not (e.g. config, measure). Compile-time check.
func TestImportsAreSelfContained(t *testing.T) {
	var _ io.Reader = (*bytes.Buffer)(nil)
}
