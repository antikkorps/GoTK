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
