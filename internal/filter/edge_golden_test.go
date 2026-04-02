package filter_test

import (
	"path/filepath"
	"testing"

	"github.com/antikkorps/GoTK/internal/filter"
	"github.com/antikkorps/GoTK/internal/testutil"
)

// genericPipeline returns the standard generic filter chain:
// StripANSI -> NormalizeWhitespace -> Dedup -> TrimEmpty.
func genericPipeline(input string) string {
	chain := filter.NewChain()
	chain.Add(filter.StripANSI)
	chain.Add(filter.NormalizeWhitespace)
	chain.Add(filter.Dedup)
	chain.Add(filter.TrimEmpty)
	return chain.Apply(input)
}

func TestGoldenEdge_Empty(t *testing.T) {
	dir := filepath.Join(testdataDir(), "edge_empty")
	testutil.RunGoldenTest(t, dir, genericPipeline)
}

func TestGoldenEdge_Binary(t *testing.T) {
	dir := filepath.Join(testdataDir(), "edge_binary")
	testutil.RunGoldenTest(t, dir, genericPipeline)
}

func TestGoldenEdge_Conflicts(t *testing.T) {
	dir := filepath.Join(testdataDir(), "edge_conflicts")
	testutil.RunGoldenTest(t, dir, genericPipeline)
}

func TestGoldenEdge_Rules(t *testing.T) {
	dir := filepath.Join(testdataDir(), "edge_rules")
	testutil.RunGoldenTest(t, dir, genericPipeline)
}
