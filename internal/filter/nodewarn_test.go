package filter

import (
	"strings"
	"testing"
)

func TestCollapseNodeWarnings(t *testing.T) {
	tests := []struct {
		name           string
		input          string
		wantContains   []string
		wantNotContain []string
		wantUnchanged  bool
	}{
		{
			name: "six identical warnings different PIDs",
			input: strings.Repeat(
				"(node:96788) Warning: `--localstorage-file` was provided without a valid path\n"+
					"(Use `node --trace-warnings ...` to show where the warning was created)\n",
				1,
			) + strings.Repeat(
				"(node:96787) Warning: `--localstorage-file` was provided without a valid path\n"+
					"(Use `node --trace-warnings ...` to show where the warning was created)\n",
				1,
			) + strings.Repeat(
				"(node:96781) Warning: `--localstorage-file` was provided without a valid path\n"+
					"(Use `node --trace-warnings ...` to show where the warning was created)\n",
				1,
			) + strings.Repeat(
				"(node:96779) Warning: `--localstorage-file` was provided without a valid path\n"+
					"(Use `node --trace-warnings ...` to show where the warning was created)\n",
				1,
			) + strings.Repeat(
				"(node:96780) Warning: `--localstorage-file` was provided without a valid path\n"+
					"(Use `node --trace-warnings ...` to show where the warning was created)\n",
				1,
			) + strings.Repeat(
				"(node:96778) Warning: `--localstorage-file` was provided without a valid path\n"+
					"(Use `node --trace-warnings ...` to show where the warning was created)\n",
				1,
			),
			wantContains: []string{
				// First block kept verbatim (with original PID).
				"(node:96788) Warning: `--localstorage-file` was provided without a valid path",
				"(Use `node --trace-warnings ...` to show where the warning was created)",
				// Count marker for the remaining 5 (format aligned with existing
				// compressNodeOutput to keep one canonical marker across paths).
				"... and 5 identical warnings from other workers",
			},
			wantNotContain: []string{
				"(node:96787)",
				"(node:96781)",
				"(node:96779)",
				"(node:96780)",
				"(node:96778)",
			},
		},
		{
			name: "single warning unchanged",
			input: "(node:12345) Warning: something\n" +
				"(Use `node --trace-warnings ...` to show where the warning was created)\n",
			wantUnchanged: true,
		},
		{
			name:          "no node warnings leaves input unchanged",
			input:         "some output\nmore output\nfinal line\n",
			wantUnchanged: true,
		},
		{
			name: "two different warnings not collapsed",
			input: "(node:100) Warning: about deprecation A\n" +
				"(Use `node --trace-warnings ...` to show where the warning was created)\n" +
				"(node:101) Warning: about deprecation B\n" +
				"(Use `node --trace-warnings ...` to show where the warning was created)\n",
			wantContains: []string{
				"(node:100) Warning: about deprecation A",
				"(node:101) Warning: about deprecation B",
			},
			wantNotContain: []string{
				"identical warnings from other workers",
			},
		},
		{
			name: "non-consecutive duplicates not collapsed",
			input: "(node:1) Warning: same thing\n" +
				"(Use `node --trace-warnings ...` to show where the warning was created)\n" +
				"unrelated log line\n" +
				"(node:2) Warning: same thing\n" +
				"(Use `node --trace-warnings ...` to show where the warning was created)\n",
			wantContains: []string{
				"(node:1) Warning: same thing",
				"unrelated log line",
				"(node:2) Warning: same thing",
			},
			wantNotContain: []string{
				"identical warnings from other workers",
			},
		},
		{
			name: "run of 3 then different warning then run of 2",
			input: "(node:1) Warning: A\n" +
				"(Use `node --trace-warnings ...` to show where the warning was created)\n" +
				"(node:2) Warning: A\n" +
				"(Use `node --trace-warnings ...` to show where the warning was created)\n" +
				"(node:3) Warning: A\n" +
				"(Use `node --trace-warnings ...` to show where the warning was created)\n" +
				"(node:4) Warning: B\n" +
				"(node:5) Warning: B\n",
			wantContains: []string{
				"(node:1) Warning: A",
				"... and 2 identical warnings from other workers",
				"(node:4) Warning: B",
				"... and 1 identical warning from other workers",
			},
		},
		{
			name: "two duplicates → 'warning' singular",
			input: "(node:1) Warning: msg\n" +
				"(Use `node --trace-warnings ...` to show where the warning was created)\n" +
				"(node:2) Warning: msg\n" +
				"(Use `node --trace-warnings ...` to show where the warning was created)\n",
			wantContains: []string{
				"... and 1 identical warning from other workers",
			},
			wantNotContain: []string{
				"identical warnings from other workers",
			},
		},
		{
			name:          "empty input",
			input:         "",
			wantUnchanged: true,
		},
		{
			name: "trailing newline preserved",
			input: "(node:1) Warning: x\n" +
				"(node:2) Warning: x\n",
			wantContains: []string{
				"(node:1) Warning: x\n",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CollapseNodeWarnings(tt.input)
			if tt.wantUnchanged && got != tt.input {
				t.Errorf("expected input unchanged\ngot:\n%s\nwant:\n%s", got, tt.input)
			}
			for _, w := range tt.wantContains {
				if !strings.Contains(got, w) {
					t.Errorf("missing %q in output:\n%s", w, got)
				}
			}
			for _, w := range tt.wantNotContain {
				if strings.Contains(got, w) {
					t.Errorf("unexpected %q in output:\n%s", w, got)
				}
			}
		})
	}
}
