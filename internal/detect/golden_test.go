package detect_test

import (
	"path/filepath"
	"runtime"
	"testing"

	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/filter"
	"github.com/antikkorps/GoTK/internal/testutil"
)

func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	return filepath.Join(projectRoot, "testdata", "golden")
}

// applyFiltersFor builds a filter function that applies all command-specific
// filters for the given command type.
func applyFiltersFor(cmdType detect.CmdType) func(string) string {
	return func(input string) string {
		filters := detect.FiltersFor(cmdType)
		chain := filter.NewChain()
		for _, f := range filters {
			chain.Add(f)
		}
		return chain.Apply(input)
	}
}

func TestGolden_Grep(t *testing.T) {
	dir := filepath.Join(testdataDir(), "grep")
	testutil.RunGoldenTest(t, dir, applyFiltersFor(detect.CmdGrep))
}

func TestGolden_GitLogStat(t *testing.T) {
	dir := filepath.Join(testdataDir(), "git")
	inputPath := filepath.Join(dir, "log_stat.input")
	expectedPath := filepath.Join(dir, "log_stat.expected")
	testutil.RunGoldenTestSingle(t, "log_stat", inputPath, expectedPath, applyFiltersFor(detect.CmdGit))
}

func TestGolden_GitDiff(t *testing.T) {
	dir := filepath.Join(testdataDir(), "git")
	inputPath := filepath.Join(dir, "diff.input")
	expectedPath := filepath.Join(dir, "diff.expected")
	testutil.RunGoldenTestSingle(t, "diff", inputPath, expectedPath, applyFiltersFor(detect.CmdGit))
}

func TestGolden_GoTest(t *testing.T) {
	dir := filepath.Join(testdataDir(), "gotest")
	testutil.RunGoldenTest(t, dir, applyFiltersFor(detect.CmdGoTool))
}

func TestGolden_Find(t *testing.T) {
	dir := filepath.Join(testdataDir(), "find")
	testutil.RunGoldenTest(t, dir, applyFiltersFor(detect.CmdFind))
}

func TestGolden_Ls(t *testing.T) {
	dir := filepath.Join(testdataDir(), "ls")
	testutil.RunGoldenTest(t, dir, applyFiltersFor(detect.CmdLs))
}

func TestGolden_QualityErrorsPreserved(t *testing.T) {
	// Quality fixtures are tested with the generic filter chain
	// (no command-specific filters) to verify errors survive all filters.
	pipeline := func(input string) string {
		chain := filter.NewChain()
		chain.Add(filter.StripANSI)
		chain.Add(filter.NormalizeWhitespace)
		chain.Add(filter.Dedup)
		chain.Add(filter.TrimEmpty)
		return chain.Apply(input)
	}
	dir := filepath.Join(testdataDir(), "quality")
	inputPath := filepath.Join(dir, "errors_preserved.input")
	expectedPath := filepath.Join(dir, "errors_preserved.expected")
	testutil.RunGoldenTestSingle(t, "errors_preserved", inputPath, expectedPath, pipeline)
}

func TestGolden_QualityGoTestFail(t *testing.T) {
	// Go test failure output through the go-specific filter chain
	dir := filepath.Join(testdataDir(), "quality")
	inputPath := filepath.Join(dir, "go_test_fail.input")
	expectedPath := filepath.Join(dir, "go_test_fail.expected")
	testutil.RunGoldenTestSingle(t, "go_test_fail", inputPath, expectedPath, applyFiltersFor(detect.CmdGoTool))
}
