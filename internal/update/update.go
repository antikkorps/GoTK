// Package update implements the `gotk update` self-upgrade command.
//
// Two modes of operation:
//
//  1. Binary replace — fetch the latest GitHub release, pick the asset for
//     the current GOOS/GOARCH, verify its SHA256 against checksums.txt,
//     extract the gotk binary, and atomically rename it over the running
//     executable. Works on Linux and macOS.
//
//  2. Source fallback — `go install github.com/antikkorps/GoTK/cmd/gotk@latest`.
//     Used when no matching binary is published, when --from-source is set,
//     or as a last resort after a download/verify failure on a platform
//     without a signed asset.
package update

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// DefaultRepo is the canonical GitHub owner/repo used for releases.
const DefaultRepo = "antikkorps/GoTK"

// UserAgent is sent with every GitHub API request to identify the client.
// GitHub's unauthenticated API rejects requests with no User-Agent.
const UserAgent = "gotk-updater"

// devVersionMarker is the baked version string when the binary is built
// without a goreleaser tag (e.g. local `go build`). For these builds the
// update command cannot reason about currency and must say so.
const devVersionMarker = "dev"

// httpTimeout caps every HTTP round-trip. The GitHub Releases API and asset
// downloads are normally fast; a bounded timeout prevents a broken network
// from hanging the CLI indefinitely.
const httpTimeout = 60 * time.Second

// Release describes the subset of the GitHub release payload we consume.
type Release struct {
	TagName string  `json:"tag_name"`
	Name    string  `json:"name"`
	HTMLURL string  `json:"html_url"`
	Assets  []Asset `json:"assets"`
}

// Asset is one downloadable file attached to a Release.
type Asset struct {
	Name        string `json:"name"`
	DownloadURL string `json:"browser_download_url"`
	Size        int64  `json:"size"`
}

// Options controls an update run.
type Options struct {
	Repo        string // defaults to DefaultRepo
	Current     string // the currently running version (e.g. main.Version)
	CheckOnly   bool   // print status and exit without downloading
	FromSource  bool   // skip the binary path, use `go install @latest`
	Force       bool   // re-install even if already at the latest version
	BaseURL     string // override for the GitHub API host (used by tests)
	AssetClient *http.Client
	Out         io.Writer // where to write user-facing messages (default: os.Stdout)
	// goos and goarch allow tests to exercise platform-specific logic
	// without being bound to the host runtime. In production both are empty
	// and the runtime constants are used.
	goos, goarch string
}

// Run executes an update with the given options. It returns a user-facing
// error on failure; callers should print the error and set a non-zero exit.
func Run(ctx context.Context, opts Options) error {
	if opts.Repo == "" {
		opts.Repo = DefaultRepo
	}
	if opts.Out == nil {
		opts.Out = os.Stdout
	}
	if opts.AssetClient == nil {
		opts.AssetClient = &http.Client{Timeout: httpTimeout}
	}
	goos := opts.goos
	if goos == "" {
		goos = runtime.GOOS
	}
	goarch := opts.goarch
	if goarch == "" {
		goarch = runtime.GOARCH
	}

	if opts.FromSource {
		fmt.Fprintln(opts.Out, "[gotk update] using go install fallback (--from-source)") //nolint:errcheck
		return goInstallLatest(ctx, opts.Out)
	}

	rel, err := FetchLatest(ctx, opts.BaseURL, opts.Repo, opts.AssetClient)
	if err != nil {
		return fmt.Errorf("fetch latest release: %w", err)
	}

	latest := strings.TrimPrefix(rel.TagName, "v")
	current := strings.TrimPrefix(opts.Current, "v")

	if opts.Current == devVersionMarker || opts.Current == "" {
		fmt.Fprintf(opts.Out, "[gotk update] current version is a development build; latest release is %s\n", rel.TagName) //nolint:errcheck
		if opts.CheckOnly {
			return nil
		}
		// Proceed with install — a dev build asking for an update clearly wants it.
	} else {
		cmp, cerr := Compare(current, latest)
		if cerr != nil {
			fmt.Fprintf(opts.Out, "[gotk update] WARN: cannot parse version %q (%v); assuming update is needed\n", opts.Current, cerr) //nolint:errcheck
			cmp = -1
		}
		switch {
		case cmp >= 0 && !opts.Force:
			fmt.Fprintf(opts.Out, "[gotk update] already up to date (current %s, latest %s)\n", opts.Current, rel.TagName) //nolint:errcheck
			return nil
		case cmp < 0:
			fmt.Fprintf(opts.Out, "[gotk update] new version available: %s → %s\n", opts.Current, rel.TagName) //nolint:errcheck
			fmt.Fprintf(opts.Out, "  release notes: %s\n", rel.HTMLURL)                                        //nolint:errcheck
		}
	}

	if opts.CheckOnly {
		return nil
	}

	asset, ok := PickAsset(rel.Assets, latest, goos, goarch)
	if !ok {
		fmt.Fprintf(opts.Out, "[gotk update] no pre-built binary for %s/%s in %s — falling back to `go install`\n", //nolint:errcheck
			goos, goarch, rel.TagName)
		return goInstallLatest(ctx, opts.Out)
	}

	checksum, cerr := fetchChecksum(ctx, rel.Assets, asset.Name, opts.AssetClient)
	if cerr != nil {
		fmt.Fprintf(opts.Out, "[gotk update] WARN: %v — falling back to `go install`\n", cerr) //nolint:errcheck
		return goInstallLatest(ctx, opts.Out)
	}

	fmt.Fprintf(opts.Out, "[gotk update] downloading %s (%s)\n", asset.Name, humanBytes(asset.Size)) //nolint:errcheck
	tmpBin, err := downloadAndExtract(ctx, asset, checksum, opts.AssetClient)
	if err != nil {
		return fmt.Errorf("download %s: %w", asset.Name, err)
	}
	defer os.Remove(tmpBin) //nolint:errcheck

	exePath, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve current executable: %w", err)
	}
	exePath, err = filepath.EvalSymlinks(exePath)
	if err != nil {
		return fmt.Errorf("resolve executable symlinks: %w", err)
	}

	if err := replaceBinary(tmpBin, exePath); err != nil {
		return fmt.Errorf("replace %s: %w", exePath, err)
	}

	fmt.Fprintf(opts.Out, "[gotk update] installed %s to %s\n", rel.TagName, exePath) //nolint:errcheck
	return nil
}

// FetchLatest calls the GitHub Releases API and returns the latest release.
// baseURL is used only by tests — pass "" in production.
func FetchLatest(ctx context.Context, baseURL, repo string, client *http.Client) (*Release, error) {
	if baseURL == "" {
		baseURL = "https://api.github.com"
	}
	url := fmt.Sprintf("%s/repos/%s/releases/latest", strings.TrimRight(baseURL, "/"), repo)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close() //nolint:errcheck

	if resp.StatusCode == http.StatusNotFound {
		return nil, errors.New("no release published yet for this repository")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github API returned HTTP %d", resp.StatusCode)
	}

	var rel Release
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, fmt.Errorf("decode release JSON: %w", err)
	}
	if rel.TagName == "" {
		return nil, errors.New("release payload has no tag_name")
	}
	return &rel, nil
}

// PickAsset returns the asset matching the given version, OS, and arch.
// Asset names follow goreleaser's default: gotk_<version>_<os>_<arch>.tar.gz.
func PickAsset(assets []Asset, version, goos, goarch string) (Asset, bool) {
	wanted := fmt.Sprintf("gotk_%s_%s_%s.tar.gz", version, goos, goarch)
	for _, a := range assets {
		if a.Name == wanted {
			return a, true
		}
	}
	return Asset{}, false
}

// fetchChecksum downloads checksums.txt from the release assets and returns
// the hex-encoded SHA256 for the requested filename.
func fetchChecksum(ctx context.Context, assets []Asset, filename string, client *http.Client) (string, error) {
	var checksums Asset
	for _, a := range assets {
		if a.Name == "checksums.txt" {
			checksums = a
			break
		}
	}
	if checksums.DownloadURL == "" {
		return "", errors.New("release has no checksums.txt asset")
	}

	body, err := httpGet(ctx, checksums.DownloadURL, client)
	if err != nil {
		return "", err
	}
	defer body.Close() //nolint:errcheck

	data, err := io.ReadAll(body)
	if err != nil {
		return "", err
	}

	// Format: "<sha256>  <filename>" per line.
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}
		if fields[1] == filename {
			return fields[0], nil
		}
	}
	return "", fmt.Errorf("checksum for %q not found in checksums.txt", filename)
}

// downloadAndExtract streams the release tarball, verifies its SHA256 against
// the supplied hex digest, extracts the `gotk` entry to a temp file, and
// returns the temp file path. The caller is responsible for removing it.
func downloadAndExtract(ctx context.Context, asset Asset, wantSHA256 string, client *http.Client) (string, error) {
	body, err := httpGet(ctx, asset.DownloadURL, client)
	if err != nil {
		return "", err
	}
	defer body.Close() //nolint:errcheck

	hasher := sha256.New()
	gz, err := gzip.NewReader(io.TeeReader(body, hasher))
	if err != nil {
		return "", fmt.Errorf("gzip reader: %w", err)
	}
	defer gz.Close() //nolint:errcheck

	tr := tar.NewReader(gz)
	tmp, err := os.CreateTemp("", "gotk-update-*.bin")
	if err != nil {
		return "", err
	}
	tmpPath := tmp.Name()
	cleanup := func() {
		tmp.Close()        //nolint:errcheck
		os.Remove(tmpPath) //nolint:errcheck
	}

	found := false
	for {
		hdr, err := tr.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			cleanup()
			return "", fmt.Errorf("tar read: %w", err)
		}
		if hdr.Typeflag != tar.TypeReg {
			continue
		}
		// The goreleaser archive places the binary at the top level as
		// simply "gotk" — match the basename to be defensive against
		// future layout changes.
		if filepath.Base(hdr.Name) != "gotk" {
			continue
		}
		if _, err := io.Copy(tmp, tr); err != nil {
			cleanup()
			return "", fmt.Errorf("write temp binary: %w", err)
		}
		found = true
		break
	}
	if !found {
		cleanup()
		return "", errors.New("archive did not contain a `gotk` binary")
	}

	// Drain the rest of the stream so the hash covers the full tarball.
	if _, err := io.Copy(io.Discard, tr); err != nil {
		cleanup()
		return "", fmt.Errorf("drain tar: %w", err)
	}
	// Pull any remaining gzip framing bytes through the hasher.
	if _, err := io.Copy(io.Discard, gz); err != nil {
		cleanup()
		return "", fmt.Errorf("drain gzip: %w", err)
	}

	gotSHA := hex.EncodeToString(hasher.Sum(nil))
	if !strings.EqualFold(gotSHA, wantSHA256) {
		cleanup()
		return "", fmt.Errorf("sha256 mismatch: want %s got %s", wantSHA256, gotSHA)
	}

	if err := tmp.Chmod(0o755); err != nil {
		cleanup()
		return "", err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpPath) //nolint:errcheck
		return "", err
	}
	return tmpPath, nil
}

// replaceBinary atomically places the binary at tmpPath at exePath, preserving
// the original mode. On Unix os.Rename replaces the inode while the running
// process keeps executing from the old in-memory image — safe to call from
// within the running gotk.
func replaceBinary(tmpPath, exePath string) error {
	// Try to preserve the current mode (e.g. user-only exec if the binary
	// lives in a restricted directory).
	if info, err := os.Stat(exePath); err == nil {
		_ = os.Chmod(tmpPath, info.Mode().Perm())
	}

	// Move to a sibling path first so the final rename is a same-filesystem
	// operation (os.Rename across filesystems fails with EXDEV).
	stagePath := exePath + ".new"
	if err := moveFile(tmpPath, stagePath); err != nil {
		return err
	}
	if err := os.Rename(stagePath, exePath); err != nil {
		os.Remove(stagePath) //nolint:errcheck
		return err
	}
	return nil
}

// moveFile renames src → dst, falling back to copy+remove when src and dst
// live on different filesystems (typical for /tmp on Linux).
func moveFile(src, dst string) error {
	if err := os.Rename(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close() //nolint:errcheck

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o755)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()    //nolint:errcheck
		os.Remove(dst) //nolint:errcheck
		return err
	}
	if err := out.Close(); err != nil {
		os.Remove(dst) //nolint:errcheck
		return err
	}
	os.Remove(src) //nolint:errcheck
	return nil
}

// goInstallLatest runs `go install github.com/antikkorps/GoTK/cmd/gotk@latest`
// in the caller's environment, streaming stdout/stderr to out for visibility.
func goInstallLatest(ctx context.Context, out io.Writer) error {
	goBin, err := exec.LookPath("go")
	if err != nil {
		return errors.New("go toolchain not found in PATH — install Go or download a release binary manually")
	}
	cmd := exec.CommandContext(ctx, goBin, "install", "github.com/antikkorps/GoTK/cmd/gotk@latest")
	cmd.Stdout = out
	cmd.Stderr = out
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("go install failed: %w", err)
	}
	fmt.Fprintln(out, "[gotk update] installed via go install (binary now lives in $GOBIN or $GOPATH/bin)") //nolint:errcheck
	return nil
}

// httpGet issues a GET request with the module's UserAgent header and returns
// the response body on a 2xx status. The caller closes the body.
func httpGet(ctx context.Context, url string, client *http.Client) (io.ReadCloser, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close() //nolint:errcheck
		return nil, fmt.Errorf("%s returned HTTP %d", url, resp.StatusCode)
	}
	return resp.Body, nil
}

// humanBytes renders a byte count for human-readable status lines.
func humanBytes(n int64) string {
	const kib = 1024
	switch {
	case n >= kib*kib:
		return strconv.FormatFloat(float64(n)/(kib*kib), 'f', 1, 64) + " MiB"
	case n >= kib:
		return strconv.FormatFloat(float64(n)/kib, 'f', 1, 64) + " KiB"
	default:
		return strconv.FormatInt(n, 10) + " B"
	}
}

// Compare returns -1, 0, or +1 when semver a is less than, equal to, or
// greater than b. Pre-release/metadata suffixes after `-` are treated as
// "older than the same base version" in the conventional semver sense, but
// fine-grained ordering within pre-releases is not attempted — this is the
// minimum behaviour needed to decide "should I update?".
func Compare(a, b string) (int, error) {
	aBase, aPre := splitPre(strings.TrimPrefix(a, "v"))
	bBase, bPre := splitPre(strings.TrimPrefix(b, "v"))

	aParts, err := parseParts(aBase)
	if err != nil {
		return 0, fmt.Errorf("parse %q: %w", a, err)
	}
	bParts, err := parseParts(bBase)
	if err != nil {
		return 0, fmt.Errorf("parse %q: %w", b, err)
	}

	for i := 0; i < 3; i++ {
		if aParts[i] != bParts[i] {
			if aParts[i] < bParts[i] {
				return -1, nil
			}
			return 1, nil
		}
	}

	// Base versions equal — a pre-release suffix means "older".
	switch {
	case aPre == "" && bPre == "":
		return 0, nil
	case aPre == "":
		return 1, nil
	case bPre == "":
		return -1, nil
	}
	return strings.Compare(aPre, bPre), nil
}

func splitPre(v string) (string, string) {
	if i := strings.IndexByte(v, '-'); i >= 0 {
		return v[:i], v[i+1:]
	}
	if i := strings.IndexByte(v, '+'); i >= 0 {
		return v[:i], v[i+1:]
	}
	return v, ""
}

func parseParts(base string) ([3]int, error) {
	var parts [3]int
	segs := strings.Split(base, ".")
	if len(segs) < 1 || len(segs) > 3 {
		return parts, fmt.Errorf("need 1-3 dotted segments, got %d in %q", len(segs), base)
	}
	for i, s := range segs {
		n, err := strconv.Atoi(s)
		if err != nil {
			return parts, fmt.Errorf("segment %d (%q): %w", i, s, err)
		}
		parts[i] = n
	}
	return parts, nil
}
