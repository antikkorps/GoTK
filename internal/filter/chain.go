package filter

import "strings"

// FilterFunc takes raw output and returns cleaned output.
type FilterFunc func(string) string

// namedFilter pairs a filter function with a human-readable name.
type namedFilter struct {
	name string
	fn   FilterFunc
}

// Chain applies a series of filters in order.
type Chain struct {
	filters []namedFilter
}

// NewChain creates an empty filter chain.
func NewChain() *Chain {
	return &Chain{}
}

// Add appends a filter to the chain with an auto-generated name.
func (c *Chain) Add(f FilterFunc) {
	c.filters = append(c.filters, namedFilter{name: "", fn: f})
}

// AddNamed appends a filter to the chain with an explicit name.
func (c *Chain) AddNamed(name string, f FilterFunc) {
	c.filters = append(c.filters, namedFilter{name: name, fn: f})
}

// Apply runs all filters in sequence. CRLF line endings (from Windows-emitted
// output) are normalized to LF at entry so downstream filters can split on
// "\n" without leaving trailing "\r" characters. Streaming mode already handles
// this via bufio.Scanner's default split function.
func (c *Chain) Apply(input string) string {
	result := strings.ReplaceAll(input, "\r\n", "\n")
	for _, nf := range c.filters {
		result = nf.fn(result)
	}
	return result
}

// Names returns the list of filter names in the chain.
// Unnamed filters appear as "(anonymous)".
func (c *Chain) Names() []string {
	names := make([]string, len(c.filters))
	for i, nf := range c.filters {
		if nf.name != "" {
			names[i] = nf.name
		} else {
			names[i] = "(anonymous)"
		}
	}
	return names
}

// Len returns the number of filters in the chain.
func (c *Chain) Len() int {
	return len(c.filters)
}
