package ctx

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestDir creates a temporary directory with test files for searching.
func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create Go file
	goContent := `package main

import "fmt"

func main() {
	fmt.Println("hello")
}

type Config struct {
	Name string
}

func (c *Config) Validate() error {
	return nil
}
`
	writeTestFile(t, dir, "main.go", goContent)

	// Create Python file
	pyContent := `import os

class Handler:
    def process(self, data):
        return data

def main():
    h = Handler()
    h.process("hello")
`
	writeTestFile(t, dir, "handler.py", pyContent)

	// Create nested file
	subdir := filepath.Join(dir, "sub")
	os.MkdirAll(subdir, 0755)
	writeTestFile(t, subdir, "util.go", `package sub

func Helper() string {
	return "helper"
}
`)

	// Create a node_modules dir that should be excluded
	nm := filepath.Join(dir, "node_modules")
	os.MkdirAll(nm, 0755)
	writeTestFile(t, nm, "junk.js", `function junk() { return "hello"; }`)

	return dir
}

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0644)
	if err != nil {
		t.Fatal(err)
	}
}

func TestParseFlags(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		want    Options
		wantErr bool
	}{
		{
			name: "basic pattern",
			args: []string{"BuildChain"},
			want: Options{Pattern: "BuildChain", Dir: ".", Mode: ModeScan, Context: 3, MaxLine: 120},
		},
		{
			name: "detail mode with context",
			args: []string{"-d", "5", "pattern"},
			want: Options{Pattern: "pattern", Dir: ".", Mode: ModeDetail, Context: 5, MaxLine: 120},
		},
		{
			name: "def mode",
			args: []string{"--def", "Handler"},
			want: Options{Pattern: "Handler", Dir: ".", Mode: ModeDef, Context: 3, MaxLine: 120},
		},
		{
			name: "tree mode",
			args: []string{"--tree", "main"},
			want: Options{Pattern: "main", Dir: ".", Mode: ModeTree, Context: 3, MaxLine: 120},
		},
		{
			name: "summary mode",
			args: []string{"--summary", "func"},
			want: Options{Pattern: "func", Dir: ".", Mode: ModeSummary, Context: 3, MaxLine: 120},
		},
		{
			name: "type filter",
			args: []string{"-t", "go", "main"},
			want: Options{Pattern: "main", Dir: ".", Mode: ModeScan, Context: 3, MaxLine: 120, FileTypes: []string{"go"}},
		},
		{
			name: "max results",
			args: []string{"-m", "5", "pattern"},
			want: Options{Pattern: "pattern", Dir: ".", Mode: ModeScan, Context: 3, MaxLine: 120, MaxResults: 5},
		},
		{
			name: "unknown flag",
			args: []string{"--unknown"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseFlags(tt.args)
			if (err != nil) != tt.wantErr {
				t.Errorf("ParseFlags() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantErr {
				return
			}
			if got.Pattern != tt.want.Pattern {
				t.Errorf("Pattern = %q, want %q", got.Pattern, tt.want.Pattern)
			}
			if got.Dir != tt.want.Dir {
				t.Errorf("Dir = %q, want %q", got.Dir, tt.want.Dir)
			}
			if got.Mode != tt.want.Mode {
				t.Errorf("Mode = %v, want %v", got.Mode, tt.want.Mode)
			}
			if got.Context != tt.want.Context {
				t.Errorf("Context = %d, want %d", got.Context, tt.want.Context)
			}
			if got.MaxResults != tt.want.MaxResults {
				t.Errorf("MaxResults = %d, want %d", got.MaxResults, tt.want.MaxResults)
			}
		})
	}
}

func TestWalkFiles(t *testing.T) {
	dir := setupTestDir(t)

	t.Run("walks all files, excludes node_modules", func(t *testing.T) {
		files, err := WalkFiles(Options{Dir: dir})
		if err != nil {
			t.Fatal(err)
		}
		// Should find main.go, handler.py, sub/util.go but NOT node_modules/junk.js
		for _, f := range files {
			if strings.Contains(f, "node_modules") {
				t.Errorf("should not include node_modules, got %s", f)
			}
		}
		if len(files) != 3 {
			t.Errorf("expected 3 files, got %d: %v", len(files), files)
		}
	})

	t.Run("type filter", func(t *testing.T) {
		files, err := WalkFiles(Options{Dir: dir, FileTypes: []string{"go"}})
		if err != nil {
			t.Fatal(err)
		}
		for _, f := range files {
			if !strings.HasSuffix(f, ".go") {
				t.Errorf("expected only .go files, got %s", f)
			}
		}
		if len(files) != 2 {
			t.Errorf("expected 2 go files, got %d", len(files))
		}
	})
}

func TestSearch(t *testing.T) {
	dir := setupTestDir(t)

	files, _ := WalkFiles(Options{Dir: dir})
	results, err := Search(files, Options{Pattern: "func"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) == 0 {
		t.Fatal("expected at least one result")
	}

	// Check that matches contain "func"
	for _, fr := range results {
		for _, m := range fr.Matches {
			if !strings.Contains(m.Line, "func") {
				t.Errorf("match line %q does not contain 'func'", m.Line)
			}
		}
	}
}

func TestSearchMaxResults(t *testing.T) {
	dir := setupTestDir(t)

	files, _ := WalkFiles(Options{Dir: dir})
	results, err := Search(files, Options{Pattern: "func", MaxResults: 1})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result with MaxResults=1, got %d", len(results))
	}
}

func TestFormatScan(t *testing.T) {
	results := []FileResult{
		{
			Path:    "main.go",
			Matches: []Match{{LineNum: 5, Line: "func main() {"}},
		},
	}
	out := FormatScan(results, DefaultOptions())
	if !strings.Contains(out, "1x main.go") {
		t.Errorf("expected '1x main.go' in output, got:\n%s", out)
	}
	if !strings.Contains(out, "5: func main()") {
		t.Errorf("expected line number and match in output, got:\n%s", out)
	}
}

func TestFormatScanEmpty(t *testing.T) {
	out := FormatScan(nil, DefaultOptions())
	if !strings.Contains(out, "no matches") {
		t.Errorf("expected 'no matches' message, got:\n%s", out)
	}
}

func TestFormatDetail(t *testing.T) {
	dir := setupTestDir(t)
	files, _ := WalkFiles(Options{Dir: dir, FileTypes: []string{"go"}})
	results, _ := Search(files, Options{Pattern: "main"})

	out := FormatDetail(results, Options{Context: 2, MaxLine: 120})
	if !strings.Contains(out, "---") {
		t.Errorf("expected header separator in detail output, got:\n%s", out)
	}
	if !strings.Contains(out, ">") {
		t.Errorf("expected match marker '>' in detail output, got:\n%s", out)
	}
}

func TestFormatDef(t *testing.T) {
	results := []FileResult{
		{
			Path: "main.go",
			Matches: []Match{
				{LineNum: 5, Line: "func main() {"},
				{LineNum: 7, Line: `	fmt.Println("hello")`},
				{LineNum: 10, Line: "type Config struct {"},
			},
		},
	}
	out := FormatDef(results, DefaultOptions())
	// Should include func and type definitions but not fmt.Println
	if !strings.Contains(out, "func main") {
		t.Errorf("expected func main in def output, got:\n%s", out)
	}
	if !strings.Contains(out, "type Config") {
		t.Errorf("expected type Config in def output, got:\n%s", out)
	}
}

func TestFormatTree(t *testing.T) {
	dir := setupTestDir(t)
	files, _ := WalkFiles(Options{Dir: dir, FileTypes: []string{"go"}})
	results, _ := Search(files, Options{Pattern: "."})

	out := FormatTree(results, DefaultOptions())
	if !strings.Contains(out, "==") {
		t.Errorf("expected file header in tree output, got:\n%s", out)
	}
	if !strings.Contains(out, "package") {
		t.Errorf("expected 'package' in tree skeleton, got:\n%s", out)
	}
}

func TestFormatSummary(t *testing.T) {
	dir := setupTestDir(t)
	files, _ := WalkFiles(Options{Dir: dir})
	results, _ := Search(files, Options{Pattern: "func"})

	out := FormatSummary(results, Options{Pattern: "func"})
	if !strings.Contains(out, "Pattern: func") {
		t.Errorf("expected 'Pattern: func' in summary, got:\n%s", out)
	}
	if !strings.Contains(out, "Total:") {
		t.Errorf("expected 'Total:' in summary, got:\n%s", out)
	}
	if !strings.Contains(out, "Directory") {
		t.Errorf("expected 'Directory' header in summary, got:\n%s", out)
	}
}

func TestFormatDispatcher(t *testing.T) {
	results := []FileResult{
		{Path: "a.go", Matches: []Match{{LineNum: 1, Line: "func a() {"}}, TotalLines: 5},
	}

	modes := []Mode{ModeScan, ModeDetail, ModeDef, ModeTree, ModeSummary}
	for _, m := range modes {
		opts := DefaultOptions()
		opts.Mode = m
		out := Format(results, opts)
		if out == "" {
			t.Errorf("Format() returned empty for mode %s", m)
		}
	}
}

func TestMergeWindows(t *testing.T) {
	matches := []Match{
		{LineNum: 5},
		{LineNum: 7},  // overlaps with 5 at context=3
		{LineNum: 20}, // separate window
	}
	windows := mergeWindows(matches, 3, 30)
	if len(windows) != 2 {
		t.Errorf("expected 2 merged windows, got %d", len(windows))
	}
}

func TestModeString(t *testing.T) {
	tests := []struct {
		mode Mode
		want string
	}{
		{ModeScan, "scan"},
		{ModeDetail, "detail"},
		{ModeDef, "def"},
		{ModeTree, "tree"},
		{ModeSummary, "summary"},
	}
	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.want {
			t.Errorf("Mode(%d).String() = %q, want %q", tt.mode, got, tt.want)
		}
	}
}
