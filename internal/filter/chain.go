package filter

// FilterFunc takes raw output and returns cleaned output.
type FilterFunc func(string) string

// Chain applies a series of filters in order.
type Chain struct {
	filters []FilterFunc
}

// NewChain creates an empty filter chain.
func NewChain() *Chain {
	return &Chain{}
}

// Add appends a filter to the chain.
func (c *Chain) Add(f FilterFunc) {
	c.filters = append(c.filters, f)
}

// Apply runs all filters in sequence.
func (c *Chain) Apply(input string) string {
	result := input
	for _, f := range c.filters {
		result = f(result)
	}
	return result
}
