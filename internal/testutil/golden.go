package testutil

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// normalizeLineEndings strips carriage returns so Windows checkouts that
// converted LF to CRLF (via Git's autocrlf default) compare cleanly against
// filter output, which is always LF after Chain.Apply's normalization. The
// .gitattributes file pins golden fixtures to LF; this is the second line
// of defense.
func normalizeLineEndings(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

// RunGoldenTest reads .input files from dir, applies filterFn, and compares
// the result with the corresponding .expected file.
// If the UPDATE_GOLDEN=1 environment variable is set, it overwrites the
// .expected files with the actual output instead of comparing.
func RunGoldenTest(t *testing.T, dir string, filterFn func(string) string) {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("failed to read golden directory %s: %v", dir, err)
	}

	var inputs []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".input") {
			inputs = append(inputs, e.Name())
		}
	}

	if len(inputs) == 0 {
		t.Fatalf("no .input files found in %s", dir)
	}

	for _, inputFile := range inputs {
		baseName := strings.TrimSuffix(inputFile, ".input")
		t.Run(baseName, func(t *testing.T) {
			inputPath := filepath.Join(dir, inputFile)
			expectedPath := filepath.Join(dir, baseName+".expected")

			inputData, err := os.ReadFile(inputPath)
			if err != nil {
				t.Fatalf("failed to read input file %s: %v", inputPath, err)
			}

			got := filterFn(string(inputData))

			if os.Getenv("UPDATE_GOLDEN") == "1" {
				if err := os.WriteFile(expectedPath, []byte(got), 0644); err != nil {
					t.Fatalf("failed to update golden file %s: %v", expectedPath, err)
				}
				t.Logf("updated golden file: %s", expectedPath)
				return
			}

			expectedData, err := os.ReadFile(expectedPath)
			if err != nil {
				t.Fatalf("failed to read expected file %s (run with UPDATE_GOLDEN=1 to create): %v", expectedPath, err)
			}

			want := normalizeLineEndings(string(expectedData))

			if got != want {
				t.Errorf("golden mismatch for %s\n--- want ---\n%s\n--- got ---\n%s\n--- diff ---\n%s",
					baseName, want, got, lineDiff(want, got))
			}
		})
	}
}

// RunGoldenTestSingle runs a single golden test with an explicit input/expected
// pair. Useful when the filter function varies per test case.
func RunGoldenTestSingle(t *testing.T, name, inputPath, expectedPath string, filterFn func(string) string) {
	t.Helper()

	t.Run(name, func(t *testing.T) {
		inputData, err := os.ReadFile(inputPath)
		if err != nil {
			t.Fatalf("failed to read input file %s: %v", inputPath, err)
		}

		got := filterFn(string(inputData))

		if os.Getenv("UPDATE_GOLDEN") == "1" {
			if err := os.WriteFile(expectedPath, []byte(got), 0644); err != nil {
				t.Fatalf("failed to update golden file %s: %v", expectedPath, err)
			}
			t.Logf("updated golden file: %s", expectedPath)
			return
		}

		expectedData, err := os.ReadFile(expectedPath)
		if err != nil {
			t.Fatalf("failed to read expected file %s (run with UPDATE_GOLDEN=1 to create): %v", expectedPath, err)
		}

		want := normalizeLineEndings(string(expectedData))

		if got != want {
			t.Errorf("golden mismatch for %s\n--- want ---\n%s\n--- got ---\n%s\n--- diff ---\n%s",
				name, want, got, lineDiff(want, got))
		}
	})
}

// lineDiff produces a simple line-by-line diff between two strings for
// diagnostic output.
func lineDiff(want, got string) string {
	wantLines := strings.Split(want, "\n")
	gotLines := strings.Split(got, "\n")

	var diff []string
	maxLen := len(wantLines)
	if len(gotLines) > maxLen {
		maxLen = len(gotLines)
	}

	for i := 0; i < maxLen; i++ {
		var w, g string
		if i < len(wantLines) {
			w = wantLines[i]
		}
		if i < len(gotLines) {
			g = gotLines[i]
		}

		if w != g {
			diff = append(diff, "  line "+strconv.Itoa(i+1)+":")
			diff = append(diff, "    want: "+repr(w))
			diff = append(diff, "    got:  "+repr(g))
		}
	}

	if len(diff) == 0 {
		return "(no visible diff)"
	}
	return strings.Join(diff, "\n")
}

// repr returns a quoted representation showing whitespace.
func repr(s string) string {
	s = strings.ReplaceAll(s, "\t", "\\t")
	s = strings.ReplaceAll(s, " ", "\u00b7")
	return "\"" + s + "\""
}
