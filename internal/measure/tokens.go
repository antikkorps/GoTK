package measure

import "strings"

// EstimateTokens returns a heuristic token count for text.
// It splits on whitespace and applies a ~1.3x multiplier for
// code/terminal output (subword splitting). Floors at len(text)/4.
func EstimateTokens(text string) int {
	if text == "" {
		return 0
	}

	words := len(strings.Fields(text))
	// Terminal/code output has more subword tokens than prose
	tokens := int(float64(words) * 1.3)

	// Floor: at minimum len/4 (average ~4 chars per token)
	floor := len(text) / 4
	if tokens < floor {
		tokens = floor
	}

	return tokens
}
