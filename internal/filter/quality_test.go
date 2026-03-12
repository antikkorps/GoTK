package filter_test

import (
	"strings"
	"testing"

	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/filter"
)

// fullPipeline applies StripANSI -> NormalizeWhitespace -> Dedup -> TrimEmpty.
func fullPipeline(input string) string {
	chain := filter.NewChain()
	chain.Add(filter.StripANSI)
	chain.Add(filter.NormalizeWhitespace)
	chain.Add(filter.Dedup)
	chain.Add(filter.TrimEmpty)
	return chain.Apply(input)
}

// fullPipelineWithCmd applies generic filters plus command-specific filters.
func fullPipelineWithCmd(cmdType detect.CmdType) func(string) string {
	return func(input string) string {
		chain := filter.NewChain()
		chain.Add(filter.StripANSI)
		chain.Add(filter.NormalizeWhitespace)
		chain.Add(filter.Dedup)
		for _, f := range detect.FiltersFor(cmdType) {
			chain.Add(f)
		}
		chain.Add(filter.TrimEmpty)
		return chain.Apply(input)
	}
}

// assertLinesPreserved checks that every line matching any of the given
// substrings appears in the output. The check is case-sensitive.
func assertLinesPreserved(t *testing.T, label, input, output string, mustContain []string) {
	t.Helper()

	inputLines := strings.Split(input, "\n")
	for _, line := range inputLines {
		for _, substr := range mustContain {
			if strings.Contains(line, substr) {
				if !strings.Contains(output, strings.TrimSpace(line)) {
					t.Errorf("[%s] critical line removed by filter:\n  line:    %q\n  matched: %q", label, line, substr)
				}
				break
			}
		}
	}
}

func TestQuality_ErrorLinesNeverRemoved(t *testing.T) {
	input := `Starting service...
INFO: Listening on :8080
ERROR: connection refused to database
Error: invalid configuration value
warning: deprecated API endpoint used
Warning: certificate expires in 7 days
WARN: disk usage above 90%
FAIL: health check timeout after 30s
panic: runtime error: index out of range [5] with length 3
normal log line
another normal line
`

	errorMarkers := []string{
		"ERROR:", "Error:", "FAIL:", "panic:",
	}
	warningMarkers := []string{
		"warning:", "Warning:", "WARN:",
	}

	t.Run("generic_pipeline", func(t *testing.T) {
		output := fullPipeline(input)
		assertLinesPreserved(t, "errors", input, output, errorMarkers)
		assertLinesPreserved(t, "warnings", input, output, warningMarkers)
	})

	t.Run("go_pipeline", func(t *testing.T) {
		output := fullPipelineWithCmd(detect.CmdGoTool)(input)
		assertLinesPreserved(t, "errors", input, output, errorMarkers)
		assertLinesPreserved(t, "warnings", input, output, warningMarkers)
	})

	t.Run("grep_pipeline", func(t *testing.T) {
		output := fullPipelineWithCmd(detect.CmdGrep)(input)
		assertLinesPreserved(t, "errors", input, output, errorMarkers)
		assertLinesPreserved(t, "warnings", input, output, warningMarkers)
	})

	t.Run("git_pipeline", func(t *testing.T) {
		output := fullPipelineWithCmd(detect.CmdGit)(input)
		assertLinesPreserved(t, "errors", input, output, errorMarkers)
		assertLinesPreserved(t, "warnings", input, output, warningMarkers)
	})
}

func TestQuality_FilePathsWithErrorsPreserved(t *testing.T) {
	input := `src/main.go:12: Error: undefined variable 'x'
src/utils.go:34: warning: unused import "fmt"
src/handler.go:99: FAIL: assertion failed
normal output line
`

	output := fullPipeline(input)

	mustContain := []string{
		"src/main.go:12",
		"src/utils.go:34",
		"src/handler.go:99",
	}

	for _, path := range mustContain {
		if !strings.Contains(output, path) {
			t.Errorf("file path %q associated with error was removed from output", path)
		}
	}
}

func TestQuality_GrepLineNumbersPreserved(t *testing.T) {
	input := `src/main.go:12:func main() {
src/main.go:25:	fmt.Println("hello")
src/utils.go:8:func helper() {
`

	filterFn := fullPipelineWithCmd(detect.CmdGrep)
	output := filterFn(input)

	// Line numbers must appear in the output
	lineNumbers := []string{"12:", "25:", "8:"}
	for _, ln := range lineNumbers {
		if !strings.Contains(output, ln) {
			t.Errorf("grep line number %q was removed from output", ln)
		}
	}
}

func TestQuality_StackTraceFramesPreserved(t *testing.T) {
	input := `panic: runtime error: nil pointer dereference

goroutine 1 [running]:
main.processData(0x0, 0x0)
	/app/src/process.go:45 +0x1a2
main.handleRequest(0xc000123456)
	/app/src/handler.go:89 +0x3f1
main.main()
	/app/src/main.go:12 +0x100
`

	output := fullPipeline(input)

	// All stack frame lines must be preserved
	frameMarkers := []string{
		"panic:",
		"goroutine 1",
		"main.processData",
		"process.go:45",
		"main.handleRequest",
		"handler.go:89",
		"main.main()",
		"main.go:12",
	}

	for _, marker := range frameMarkers {
		if !strings.Contains(output, marker) {
			t.Errorf("stack trace frame containing %q was removed from output", marker)
		}
	}
}

func TestQuality_GoTestFailureDetailsPreserved(t *testing.T) {
	input := `=== RUN   TestSomething
    something_test.go:23: expected 42, got 0
    something_test.go:24: Error: value was not initialized
--- FAIL: TestSomething (0.01s)
FAIL
FAIL	github.com/example/pkg	0.015s
`

	filterFn := fullPipelineWithCmd(detect.CmdGoTool)
	output := filterFn(input)

	mustContain := []string{
		"something_test.go:23",
		"expected 42, got 0",
		"Error: value was not initialized",
		"--- FAIL: TestSomething",
		"FAIL",
	}

	for _, s := range mustContain {
		if !strings.Contains(output, s) {
			t.Errorf("go test failure detail %q was removed from output", s)
		}
	}
}

func TestQuality_WarningLinesNeverRemoved(t *testing.T) {
	// Focused test: warnings through every individual filter
	warningLines := []string{
		"warning: something might be wrong",
		"Warning: deprecated feature used",
		"WARN: memory usage high",
		"[WARN] connection pool exhausted",
	}

	input := strings.Join(warningLines, "\n") + "\n"

	filters := map[string]func(string) string{
		"StripANSI":           filter.StripANSI,
		"NormalizeWhitespace": filter.NormalizeWhitespace,
		"Dedup":               filter.Dedup,
		"TrimEmpty":           filter.TrimEmpty,
	}

	for name, f := range filters {
		t.Run(name, func(t *testing.T) {
			output := f(input)
			for _, line := range warningLines {
				trimmed := strings.TrimSpace(line)
				if !strings.Contains(output, trimmed) {
					t.Errorf("filter %s removed warning line: %q", name, line)
				}
			}
		})
	}
}

func TestQuality_ErrorLinesNeverRemovedByIndividualFilters(t *testing.T) {
	errorLines := []string{
		"ERROR: critical failure",
		"Error: something went wrong",
		"error: compilation failed",
		"FAIL: test assertion",
		"panic: nil pointer dereference",
	}

	input := strings.Join(errorLines, "\n") + "\n"

	filters := map[string]func(string) string{
		"StripANSI":           filter.StripANSI,
		"NormalizeWhitespace": filter.NormalizeWhitespace,
		"Dedup":               filter.Dedup,
		"TrimEmpty":           filter.TrimEmpty,
	}

	for name, f := range filters {
		t.Run(name, func(t *testing.T) {
			output := f(input)
			for _, line := range errorLines {
				trimmed := strings.TrimSpace(line)
				if !strings.Contains(output, trimmed) {
					t.Errorf("filter %s removed error line: %q", name, line)
				}
			}
		})
	}
}
