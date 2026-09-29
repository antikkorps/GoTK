package filter

import (
	"strings"
	"testing"
)

func TestChain(t *testing.T) {
	tests := []struct {
		name    string
		filters []FilterFunc
		input   string
		want    string
	}{
		{
			name:    "empty chain is identity",
			filters: nil,
			input:   "hello world",
			want:    "hello world",
		},
		{
			name: "single filter",
			filters: []FilterFunc{
				strings.ToUpper,
			},
			input: "hello",
			want:  "HELLO",
		},
		{
			name: "multiple filters compose correctly",
			filters: []FilterFunc{
				strings.ToUpper,
				func(s string) string { return s + "!" },
			},
			input: "hello",
			want:  "HELLO!",
		},
		{
			name: "filters apply in order",
			filters: []FilterFunc{
				func(s string) string { return s + " first" },
				func(s string) string { return s + " second" },
				func(s string) string { return s + " third" },
			},
			input: "start",
			want:  "start first second third",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChain()
			for _, f := range tt.filters {
				c.Add(f)
			}
			got := c.Apply(tt.input)
			if got != tt.want {
				t.Errorf("Chain.Apply(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

// TestChainNormalizesCRLF verifies Windows-style line endings are normalized
// to LF at chain entry, so downstream filters that split on "\n" don't see
// trailing "\r" characters.
func TestChainNormalizesCRLF(t *testing.T) {
	c := NewChain()
	// A filter that asserts no \r leaks through.
	c.Add(func(s string) string {
		if strings.Contains(s, "\r") {
			t.Errorf("filter received input containing \\r: %q", s)
		}
		return s
	})

	got := c.Apply("line1\r\nline2\r\nline3")
	want := "line1\nline2\nline3"
	if got != want {
		t.Errorf("Chain.Apply with CRLF = %q, want %q", got, want)
	}
}

// Issue #85: the chain output ends with a newline exactly when its input does.
func TestChainApply_PreservesTrailingNewline(t *testing.T) {
	stripAll := func(s string) string { return strings.TrimRight(s, "\n") }
	addOne := func(s string) string { return s + "\n\n" }

	tests := []struct {
		name  string
		f     FilterFunc
		input string
		want  string
	}{
		{"restores removed newline", stripAll, "x\n", "x\n"},
		{"drops added newline", addOne, "a\nb", "a\nb"},
		{"collapses extra newlines", addOne, "a\n", "a\n"},
		{"empty stays empty", stripAll, "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewChain()
			c.Add(tt.f)
			if got := c.Apply(tt.input); got != tt.want {
				t.Errorf("Apply(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
