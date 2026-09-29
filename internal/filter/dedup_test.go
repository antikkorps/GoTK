package filter

import "testing"

func TestDedup(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "consecutive duplicates with count",
			input: "hello\nhello\nhello\nhello\nworld",
			want:  "hello\n  ... (3 duplicate lines)\nworld",
		},
		{
			name:  "single duplicate",
			input: "hello\nhello\nworld",
			want:  "hello\n  ... (1 duplicate line)\nworld",
		},
		{
			name:  "no duplicates",
			input: "hello\nworld\nfoo",
			want:  "hello\nworld\nfoo",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name:  "all lines the same",
			input: "x\nx\nx\nx\nx",
			want:  "x\n  ... (4 duplicate lines)",
		},
		{
			name:  "non-consecutive duplicates preserved",
			input: "hello\nworld\nhello",
			want:  "hello\nworld\nhello",
		},
		{
			name:  "multiple groups of duplicates",
			input: "a\na\na\nb\nb\nc",
			want:  "a\n  ... (2 duplicate lines)\nb\n  ... (1 duplicate line)\nc",
		},
		{
			name:  "trailing duplicates",
			input: "a\nb\nb\nb",
			want:  "a\nb\n  ... (2 duplicate lines)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Dedup(tt.input)
			if got != tt.want {
				t.Errorf("Dedup(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestDedupBlocks(t *testing.T) {
	trace := "Error: No route matches URL \"/x.js\"\n" +
		"    at getInternalRouterError (chunk.mjs:5503:5)\n" +
		"    at Object.query (chunk.mjs:3505:19)\n"
	jest := "  console.log\n" +
		"    ✅ Jest setup loaded\n" +
		"      at Object.log (tests/setup.js:188:9)\n"

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "repeated stack trace collapsed (#84)",
			input: "start\n" + trace + trace + trace + "32 passed\n",
			want:  "start\n" + trace + "  ... (previous 3-line block repeated 2 more times)\n32 passed\n",
		},
		{
			name:  "repeated jest console block collapsed (#87)",
			input: "> jest\n" + jest + jest + "Tests: 5 passed\n",
			want:  "> jest\n" + jest + "  ... (previous 3-line block repeated 1 more time)\nTests: 5 passed\n",
		},
		{
			name:  "blocks differing by one byte kept",
			input: "a\nb1\na\nb2\n",
			want:  "a\nb1\na\nb2\n",
		},
		{
			name:  "no trailing newline preserved",
			input: "a\nb\na\nb",
			want:  "a\nb\n  ... (previous 2-line block repeated 1 more time)",
		},
		{
			name:  "smallest period wins on ties",
			input: "a\nb\na\nb\na\nb\na\nb\n",
			want:  "a\nb\n  ... (previous 2-line block repeated 3 more times)\n",
		},
		{
			name:  "blank blocks untouched",
			input: "x\n\n\n\n\ny\n",
			want:  "x\n\n\n\n\ny\n",
		},
		{
			name:  "empty",
			input: "",
			want:  "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := DedupBlocks(tt.input); got != tt.want {
				t.Errorf("DedupBlocks() =\n%q\nwant\n%q", got, tt.want)
			}
		})
	}
}
