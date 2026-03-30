package filter

import (
	"fmt"
	"strings"
	"testing"
)

// genLines creates n lines of the given content.
func genLines(n int, content string) string {
	lines := make([]string, n)
	for i := range lines {
		lines[i] = content
	}
	return strings.Join(lines, "\n") + "\n"
}

// genMixed creates output with a mix of info, error, and warning lines.
func genMixed(infoCount, errorCount, warningCount int) string {
	var lines []string
	for i := 0; i < errorCount; i++ {
		lines = append(lines, fmt.Sprintf("error: something broke at step %d", i+1))
	}
	for i := 0; i < warningCount; i++ {
		lines = append(lines, fmt.Sprintf("warning: deprecation notice %d", i+1))
	}
	for i := 0; i < infoCount; i++ {
		lines = append(lines, fmt.Sprintf("building package %d", i+1))
	}
	return strings.Join(lines, "\n") + "\n"
}

func TestSummarize(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantSummary    bool // whether [gotk summary] header should be present
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:        "small output no summary",
			input:       genLines(50, "hello world"),
			wantSummary: false,
		},
		{
			name:        "empty input",
			input:       "",
			wantSummary: false,
		},
		{
			name:        "exactly at threshold no summary",
			input:       genLines(99, "some info line"),
			wantSummary: false,
		},
		{
			name:        "at threshold boundary triggers summary",
			input:       genLines(100, "some info line"),
			wantSummary: true,
			wantContains: []string{
				"[gotk summary]",
				"[/gotk summary]",
				"total: 100 lines",
				"errors: 0",
				"result: unknown",
			},
		},
		{
			name:        "large output with errors",
			input:       genMixed(95, 5, 0),
			wantSummary: true,
			wantContains: []string{
				"errors: 5",
				"→ error: something broke at step 1",
				"→ error: something broke at step 2",
				"→ error: something broke at step 3",
			},
			wantNotContain: []string{
				"→ error: something broke at step 4",
			},
		},
		{
			name:        "large output with warnings",
			input:       genMixed(95, 0, 5),
			wantSummary: true,
			wantContains: []string{
				"warnings: 5",
				"→ warning: deprecation notice 1",
				"→ warning: deprecation notice 2",
			},
			wantNotContain: []string{
				"→ warning: deprecation notice 3",
			},
		},
		{
			name:        "mixed output correct counts",
			input:       genMixed(90, 4, 7),
			wantSummary: true,
			wantContains: []string{
				"errors: 4",
				"warnings: 7",
			},
		},
		{
			name:        "result PASS detected",
			input:       genLines(99, "building ok") + "PASS\n",
			wantSummary: true,
			wantContains: []string{
				"result: PASS",
			},
		},
		{
			name:        "result FAIL detected",
			input:       genLines(99, "running tests") + "FAIL something broke\n",
			wantSummary: true,
			wantContains: []string{
				"result: FAIL",
			},
		},
		{
			name:        "FAIL overrides PASS",
			input:       genLines(50, "ok all good") + genLines(49, "building") + "PASS\nFAIL final\n",
			wantSummary: true,
			wantContains: []string{
				"result: FAIL",
			},
		},
		{
			name: "file path extraction",
			input: func() string {
				var lines []string
				lines = append(lines, "src/main.go:42: error: undefined foo")
				lines = append(lines, "src/test.go:15: error: expected 1 got 2")
				lines = append(lines, "src/main.go:50: warning: unused variable")
				for i := 0; i < 100; i++ {
					lines = append(lines, "regular output line")
				}
				return strings.Join(lines, "\n") + "\n"
			}(),
			wantSummary: true,
			wantContains: []string{
				"files: 2 unique paths mentioned",
			},
		},
		{
			name:        "summary format starts with header",
			input:       genLines(110, "info line"),
			wantSummary: true,
			wantContains: []string{
				"[gotk summary]",
				"[/gotk summary]",
				"total: 110 lines",
			},
		},
		{
			name:        "original content preserved after summary",
			input:       genLines(105, "keep this content"),
			wantSummary: true,
			wantContains: []string{
				"keep this content",
			},
		},
		{
			name:        "comma formatting for large numbers",
			input:       genLines(1500, "line of output"),
			wantSummary: true,
			wantContains: []string{
				"1,500 lines",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Summarize(tt.input)

			hasSummary := strings.Contains(got, "[gotk summary]")
			if hasSummary != tt.wantSummary {
				t.Errorf("summary presence = %v, want %v", hasSummary, tt.wantSummary)
			}

			for _, want := range tt.wantContains {
				if !strings.Contains(got, want) {
					t.Errorf("output missing %q", want)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(got, notWant) {
					t.Errorf("output should not contain %q", notWant)
				}
			}

			// If summary was added, verify original content is still present
			if tt.wantSummary && tt.input != "" {
				// The original input should appear after the summary block
				idx := strings.Index(got, "[/gotk summary]")
				if idx < 0 {
					t.Fatal("missing [/gotk summary] closing tag")
				}
				afterSummary := got[idx:]
				// Check that some original content is in the portion after the summary
				firstOrigLine := strings.SplitN(tt.input, "\n", 2)[0]
				if firstOrigLine != "" && !strings.Contains(afterSummary, firstOrigLine) {
					t.Errorf("original content not preserved after summary")
				}
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input int
		want  string
	}{
		{0, "0"},
		{5, "5"},
		{42, "42"},
		{999, "999"},
		{1000, "1,000"},
		{1247, "1,247"},
		{34892, "34,892"},
		{1000000, "1,000,000"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := formatNumber(tt.input)
			if got != tt.want {
				t.Errorf("formatNumber(%d) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
