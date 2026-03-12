package filter

import "testing"

func TestNormalizeWhitespace(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "trailing spaces removal",
			input: "hello   \nworld  \n",
			want:  "hello\nworld\n",
		},
		{
			name:  "trailing tabs removal",
			input: "hello\t\nworld\t\t\n",
			want:  "hello\nworld\n",
		},
		{
			name:  "multiple blank lines collapsed",
			input: "hello\n\n\n\n\nworld\n",
			want:  "hello\n\nworld\n",
		},
		{
			name:  "leading whitespace trimmed from output",
			input: "\n\n  \nhello\n",
			want:  "hello\n",
		},
		{
			name:  "trailing whitespace trimmed from output",
			input: "hello\n  \n\n",
			want:  "hello\n",
		},
		{
			name:  "empty input",
			input: "",
			want:  "",
		},
		{
			name:  "only whitespace",
			input: "   \n  \n\t\n",
			want:  "",
		},
		{
			name:  "already normalized",
			input: "hello\nworld\n",
			want:  "hello\nworld\n",
		},
		{
			name:  "single line no trailing newline",
			input: "hello",
			want:  "hello\n",
		},
		{
			name:  "preserves single blank line between content",
			input: "hello\n\nworld\n",
			want:  "hello\n\nworld\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalizeWhitespace(tt.input)
			if got != tt.want {
				t.Errorf("NormalizeWhitespace(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
