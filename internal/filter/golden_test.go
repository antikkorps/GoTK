package filter_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/antikkorps/GoTK/internal/filter"
	"github.com/antikkorps/GoTK/internal/testutil"
)

// testdataDir returns the absolute path to testdata/golden relative to the
// project root.
func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	// internal/filter/golden_test.go -> project root
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	return filepath.Join(projectRoot, "testdata", "golden")
}

func TestGolden_StripANSI(t *testing.T) {
	// ANSI escape bytes cannot be reliably stored in text fixture files,
	// so we construct input in-memory with real ESC (\x1b) bytes.
	esc := "\x1b"

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "colored_grep",
			input: esc + "[35msrc/main.go" + esc + "[0m:" + esc + "[33m12" + esc + "[0m:func " + esc + "[1;31mmain" + esc + "[0m() {\n" +
				esc + "[35msrc/main.go" + esc + "[0m:" + esc + "[33m25" + esc + "[0m:\tfmt.Println(\"" + esc + "[1;31mmain" + esc + "[0m: starting\")\n" +
				esc + "[35msrc/utils.go" + esc + "[0m:" + esc + "[33m8" + esc + "[0m:// Called from " + esc + "[1;31mmain" + esc + "[0m\n" +
				esc + "[35msrc/utils.go" + esc + "[0m:" + esc + "[33m44" + esc + "[0m:\tlog.Printf(\"" + esc + "[1;31mmain" + esc + "[0m loop iteration %d\", i)\n" +
				esc + "[35mlib/core.go" + esc + "[0m:" + esc + "[33m99" + esc + "[0m:\t// See " + esc + "[1;31mmain" + esc + "[0m for usage\n" +
				esc + "[36m--" + esc + "[0m\n" +
				esc + "[1;32mMatches: 5" + esc + "[0m\n",
			want: "src/main.go:12:func main() {\n" +
				"src/main.go:25:\tfmt.Println(\"main: starting\")\n" +
				"src/utils.go:8:// Called from main\n" +
				"src/utils.go:44:\tlog.Printf(\"main loop iteration %d\", i)\n" +
				"lib/core.go:99:\t// See main for usage\n" +
				"--\n" +
				"Matches: 5\n",
		},
		{
			name:  "no_ansi_passthrough",
			input: "plain text output\nwith multiple lines\n",
			want:  "plain text output\nwith multiple lines\n",
		},
		{
			name:  "cursor_movement",
			input: "before" + esc + "[2Jafter\n",
			want:  "beforeafter\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := filter.StripANSI(tt.input)
			if got != tt.want {
				t.Errorf("StripANSI:\n  want: %q\n  got:  %q", tt.want, got)
			}
		})
	}
}

func TestGolden_NormalizeWhitespace(t *testing.T) {
	dir := filepath.Join(testdataDir(), "whitespace")
	testutil.RunGoldenTest(t, dir, filter.NormalizeWhitespace)
}

func TestGolden_Dedup(t *testing.T) {
	dir := filepath.Join(testdataDir(), "dedup")
	testutil.RunGoldenTest(t, dir, filter.Dedup)
}

func TestGolden_FullGenericPipeline(t *testing.T) {
	// Test the full generic filter chain on the whitespace fixture.
	pipeline := func(input string) string {
		chain := filter.NewChain()
		chain.Add(filter.StripANSI)
		chain.Add(filter.NormalizeWhitespace)
		chain.Add(filter.Dedup)
		chain.Add(filter.TrimEmpty)
		return chain.Apply(input)
	}

	dir := filepath.Join(testdataDir(), "whitespace")
	inputPath := filepath.Join(dir, "messy_output.input")
	expectedPath := filepath.Join(dir, "messy_output.expected")
	testutil.RunGoldenTestSingle(t, "whitespace_through_full_pipeline", inputPath, expectedPath, pipeline)
}
