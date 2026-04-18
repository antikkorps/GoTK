package mcp

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"
)

// setupGlobTree creates a temp tree:
//
//	root/
//	  main.go
//	  util.go
//	  docs/README.md
//	  docs/guide.md
//	  src/cmd/run.go
//	  src/cmd/run_test.go
//	  node_modules/ignored.js
//	  .git/HEAD
func setupGlobTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	files := map[string]string{
		"main.go":                 "package main\n",
		"util.go":                 "package main\n",
		"docs/README.md":          "# readme\n",
		"docs/guide.md":           "# guide\n",
		"src/cmd/run.go":          "package cmd\n",
		"src/cmd/run_test.go":     "package cmd\n",
		"node_modules/ignored.js": "nope\n",
		".git/HEAD":               "ref: refs/heads/main\n",
	}
	for rel, content := range files {
		full := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
			t.Fatalf("write: %v", err)
		}
	}
	return root
}

func relSlice(root string, paths []string) []string {
	out := make([]string, 0, len(paths))
	for _, p := range paths {
		rel, err := filepath.Rel(root, p)
		if err != nil {
			rel = p
		}
		out = append(out, filepath.ToSlash(rel))
	}
	sort.Strings(out)
	return out
}

func TestRunGlobBasenamePattern(t *testing.T) {
	root := setupGlobTree(t)

	got, err := runGlob("*.go", root, 0)
	if err != nil {
		t.Fatalf("runGlob: %v", err)
	}
	want := []string{"main.go", "src/cmd/run.go", "src/cmd/run_test.go", "util.go"}
	if diff := cmpSlices(relSlice(root, got), want); diff != "" {
		t.Errorf("%s", diff)
	}
}

func TestRunGlobStripsDoubleStarPrefix(t *testing.T) {
	root := setupGlobTree(t)

	got, err := runGlob("**/*.md", root, 0)
	if err != nil {
		t.Fatalf("runGlob: %v", err)
	}
	want := []string{"docs/README.md", "docs/guide.md"}
	if diff := cmpSlices(relSlice(root, got), want); diff != "" {
		t.Errorf("%s", diff)
	}
}

func TestRunGlobExcludesNoiseDirs(t *testing.T) {
	root := setupGlobTree(t)

	got, err := runGlob("*.js", root, 0)
	if err != nil {
		t.Fatalf("runGlob: %v", err)
	}
	// Both node_modules/ignored.js and any file inside .git must be excluded.
	for _, p := range got {
		if strings.Contains(p, "node_modules") || strings.Contains(p, ".git") {
			t.Errorf("glob included excluded directory: %s", p)
		}
	}
}

func TestRunGlobPathPattern(t *testing.T) {
	root := setupGlobTree(t)

	got, err := runGlob("src/cmd/*.go", root, 0)
	if err != nil {
		t.Fatalf("runGlob: %v", err)
	}
	want := []string{"src/cmd/run.go", "src/cmd/run_test.go"}
	if diff := cmpSlices(relSlice(root, got), want); diff != "" {
		t.Errorf("%s", diff)
	}
}

func TestRunGlobMaxResultsCap(t *testing.T) {
	root := setupGlobTree(t)

	got, err := runGlob("*.go", root, 2)
	if err != nil {
		t.Fatalf("runGlob: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("want 2 results (capped), got %d: %v", len(got), got)
	}
}

func TestFormatGlobResultsFlat(t *testing.T) {
	root := setupGlobTree(t)
	matches := []string{
		filepath.Join(root, "main.go"),
		filepath.Join(root, "util.go"),
	}

	got := formatGlobResults(matches, root, nil)

	if strings.Contains(got, "(2)") {
		t.Errorf("flat output should not show a cluster count: %s", got)
	}
	for _, want := range []string{"main.go", "util.go"} {
		if !strings.Contains(got, want) {
			t.Errorf("flat output missing %q: %s", want, got)
		}
	}
}

func TestFormatGlobResultsClusterAutoOnManyMatches(t *testing.T) {
	root := "/tmp/fake"
	matches := make([]string, 0, 40)
	for i := 0; i < 20; i++ {
		matches = append(matches, filepath.Join(root, "src/a", filenameN("f", i, ".go")))
	}
	for i := 0; i < 20; i++ {
		matches = append(matches, filepath.Join(root, "src/b", filenameN("g", i, ".go")))
	}

	got := formatGlobResults(matches, root, nil)

	// Clustered output must contain the directory headers with counts.
	if !strings.Contains(got, "src/a/ (20):") {
		t.Errorf("missing src/a cluster header with count in:\n%s", got)
	}
	if !strings.Contains(got, "src/b/ (20):") {
		t.Errorf("missing src/b cluster header with count in:\n%s", got)
	}
}

func TestFormatGlobResultsClusterForcedOff(t *testing.T) {
	root := "/tmp/fake"
	matches := make([]string, 0, 40)
	for i := 0; i < 40; i++ {
		matches = append(matches, filepath.Join(root, "pkg", filenameN("a", i, ".go")))
	}
	off := false

	got := formatGlobResults(matches, root, &off)

	if strings.Contains(got, "(40):") {
		t.Errorf("cluster=false must keep the flat layout: %s", got)
	}
}

func TestCapGrepMatchesPerFile(t *testing.T) {
	input := ">> src/foo.go\n" +
		"  10:match1\n" +
		"  20:match2\n" +
		"  30:match3\n" +
		"  40:match4\n" +
		"\n" +
		">> src/bar.go\n" +
		"  5:one\n"

	got := capGrepMatchesPerFile(input, 2)

	// foo.go kept only first 2 matches + marker.
	if !strings.Contains(got, "  10:match1") {
		t.Errorf("first match missing in:\n%s", got)
	}
	if !strings.Contains(got, "  20:match2") {
		t.Errorf("second match missing in:\n%s", got)
	}
	if strings.Contains(got, "  30:match3") {
		t.Errorf("third match should have been dropped:\n%s", got)
	}
	if !strings.Contains(got, "… 2 more in this file") {
		t.Errorf("expected overflow marker with count 2, got:\n%s", got)
	}
	// bar.go had only 1 match — no marker.
	if !strings.Contains(got, "  5:one") {
		t.Errorf("second file's single match should be preserved:\n%s", got)
	}
	if strings.Count(got, "more in this file") != 1 {
		t.Errorf("only the overflowed file should get a marker; got:\n%s", got)
	}
}

func TestCapGrepMatchesPerFileZeroMeansUnlimited(t *testing.T) {
	input := ">> f.go\n  1:a\n  2:b\n  3:c\n"
	got := capGrepMatchesPerFile(input, 0)
	if got != input {
		t.Errorf("perFile=0 should be a no-op, got:\n%s", got)
	}
}

// helpers

func cmpSlices(got, want []string) string {
	if len(got) != len(want) {
		return "len mismatch: got=" + strings.Join(got, ",") + " want=" + strings.Join(want, ",")
	}
	for i := range got {
		if got[i] != want[i] {
			return "mismatch at " + got[i] + " vs " + want[i]
		}
	}
	return ""
}

func filenameN(prefix string, i int, ext string) string {
	s := prefix
	// cheap itoa — keeps test self-contained without importing strconv twice
	if i == 0 {
		s += "0"
	} else {
		var rev []byte
		for n := i; n > 0; n /= 10 {
			rev = append([]byte{byte('0' + n%10)}, rev...)
		}
		s += string(rev)
	}
	return s + ext
}
