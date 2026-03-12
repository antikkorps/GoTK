package filter

import "testing"

func TestStripANSI(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "red color code",
			input: "\x1b[31mERROR\x1b[0m",
			want:  "ERROR",
		},
		{
			name:  "green color code",
			input: "\x1b[32mOK\x1b[0m",
			want:  "OK",
		},
		{
			name:  "bold text",
			input: "\x1b[1mBold\x1b[0m",
			want:  "Bold",
		},
		{
			name:  "cursor movement",
			input: "\x1b[2Ahello\x1b[3B",
			want:  "hello",
		},
		{
			name:  "mixed ANSI and normal text",
			input: "start \x1b[31mred\x1b[0m middle \x1b[1;32mbold green\x1b[0m end",
			want:  "start red middle bold green end",
		},
		{
			name:  "already clean text",
			input: "no escape codes here",
			want:  "no escape codes here",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
		{
			name:  "multiple color codes in sequence",
			input: "\x1b[1m\x1b[31m\x1b[4mstacked\x1b[0m",
			want:  "stacked",
		},
		{
			name:  "OSC sequence (title set)",
			input: "\x1b]0;Window Title\x07rest",
			want:  "rest",
		},
		{
			name:  "multiline with codes",
			input: "\x1b[31mline1\x1b[0m\n\x1b[32mline2\x1b[0m",
			want:  "line1\nline2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripANSI(tt.input)
			if got != tt.want {
				t.Errorf("StripANSI(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
