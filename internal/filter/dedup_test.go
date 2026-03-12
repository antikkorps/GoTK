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
