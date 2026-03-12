package filter

import "testing"

func TestIsDecorative(t *testing.T) {
	tests := []struct {
		name string
		line string
		want bool
	}{
		{
			name: "long dashes separator",
			line: "----------",
			want: true,
		},
		{
			name: "long equals separator",
			line: "==========",
			want: true,
		},
		{
			name: "long underscores separator",
			line: "___________",
			want: true,
		},
		{
			name: "long tildes separator",
			line: "~~~~~~~~~~~",
			want: true,
		},
		{
			name: "dashes with spaces (long enough)",
			line: "--- --- ---",
			want: true,
		},
		{
			name: "short dashes preserved (3 chars - could be meaningful)",
			line: "---",
			want: false,
		},
		{
			name: "short dashes preserved (5 chars)",
			line: "-----",
			want: false,
		},
		{
			name: "9 chars still preserved",
			line: "---------",
			want: false,
		},
		{
			name: "2 chars preserved",
			line: "--",
			want: false,
		},
		{
			name: "empty string not decorative",
			line: "",
			want: false,
		},
		{
			name: "normal text",
			line: "hello world",
			want: false,
		},
		{
			name: "mixed decorative and text",
			line: "---hello---",
			want: false,
		},
		{
			name: "asterisks not decorative (removed from decorChars)",
			line: "***********",
			want: false,
		},
		{
			name: "hashes not decorative (removed from decorChars)",
			line: "###########",
			want: false,
		},
		{
			name: "plus not decorative (removed from decorChars)",
			line: "+++++++++++",
			want: false,
		},
		{
			name: "go test FAIL line preserved",
			line: "--- FAIL: TestSomething (0.00s)",
			want: false,
		},
		{
			name: "go test PASS line preserved",
			line: "--- PASS: TestSomething (0.00s)",
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isDecorative(tt.line)
			if got != tt.want {
				t.Errorf("isDecorative(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestTrimEmpty(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "removes long decorative lines",
			input: "hello\n----------\nworld",
			want:  "hello\nworld",
		},
		{
			name:  "preserves blank lines",
			input: "hello\n\nworld",
			want:  "hello\n\nworld",
		},
		{
			name:  "preserves normal content",
			input: "hello\nworld\nfoo",
			want:  "hello\nworld\nfoo",
		},
		{
			name:  "removes multiple types of long decorative lines",
			input: "hello\n==========\nworld\n~~~~~~~~~~\nend",
			want:  "hello\nworld\nend",
		},
		{
			name:  "preserves short dashes (semantic in diffs/tests)",
			input: "hello\n---\nworld",
			want:  "hello\n---\nworld",
		},
		{
			name:  "preserves go test markers",
			input: "--- FAIL: TestFoo (0.00s)\n    expected 1, got 2",
			want:  "--- FAIL: TestFoo (0.00s)\n    expected 1, got 2",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := TrimEmpty(tt.input)
			if got != tt.want {
				t.Errorf("TrimEmpty(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
