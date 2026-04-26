package filter

import (
	"os"
	"strings"
)

// StreamConfig controls which filters are applied in streaming mode.
type StreamConfig struct {
	StripANSI           bool
	CompressPaths       bool
	Dedup               bool
	TrimDecorative      bool
	NormalizeWhitespace bool
}

// StreamFilter processes lines one at a time with minimal buffering.
// Some filters (like dedup) need to see the previous line, so they
// maintain a small internal state.
type StreamFilter struct {
	stripANSI      bool
	compressPaths  bool
	dedup          bool
	trimDecorative bool
	normalizeWS    bool

	// dedup state
	prevLine string
	dupCount int
	hasPrev  bool

	// normalize whitespace state: count consecutive blank lines
	blankCount int

	// path compression cache
	cwd  string
	home string
}

// NewStreamFilter creates a stream filter with the given config.
func NewStreamFilter(cfg StreamConfig) *StreamFilter {
	sf := &StreamFilter{
		stripANSI:      cfg.StripANSI,
		compressPaths:  cfg.CompressPaths,
		dedup:          cfg.Dedup,
		trimDecorative: cfg.TrimDecorative,
		normalizeWS:    cfg.NormalizeWhitespace,
	}

	if sf.compressPaths {
		sf.cwd, _ = os.Getwd()
		sf.home, _ = os.UserHomeDir()
	}

	return sf
}

// ProcessLine filters a single line. Returns (output, emit) where
// emit=false means the line is buffered (e.g., dedup accumulating).
func (sf *StreamFilter) ProcessLine(line string) (string, bool) {
	// Strip ANSI escape codes (stateless).
	if sf.stripANSI {
		line = ansiPattern.ReplaceAllString(line, "")
	}

	// Compress paths (stateless).
	if sf.compressPaths {
		line = sf.compressLine(line)
	}

	// Normalize whitespace: trim trailing spaces.
	if sf.normalizeWS {
		line = strings.TrimRight(line, " \t")
	}

	// Trim decorative lines (stateless).
	if sf.trimDecorative {
		trimmed := strings.TrimSpace(line)
		if isDecorative(trimmed) {
			return "", false
		}
	}

	// Normalize whitespace: collapse multiple blank lines.
	if sf.normalizeWS {
		if strings.TrimSpace(line) == "" {
			sf.blankCount++
			if sf.blankCount >= 2 {
				// Suppress extra blank lines (allow at most 1).
				return "", false
			}
		} else {
			sf.blankCount = 0
		}
	}

	// Dedup: compare with previous line.
	if sf.dedup {
		if sf.hasPrev && line == sf.prevLine {
			sf.dupCount++
			return "", false
		}

		// Line differs from previous. Emit pending dup marker if needed.
		var output string
		if sf.dupCount > 0 {
			output = formatDupMarker(sf.dupCount) + "\n" + line
			sf.dupCount = 0
			sf.prevLine = line
			return output, true
		}

		sf.prevLine = line
		sf.hasPrev = true
		return line, true
	}

	return line, true
}

// Flush returns any buffered output (e.g., pending dedup marker).
func (sf *StreamFilter) Flush() string {
	if sf.dedup && sf.dupCount > 0 {
		marker := formatDupMarker(sf.dupCount)
		sf.dupCount = 0
		return marker
	}
	return ""
}

// compressLine shortens absolute paths in a single line.
func (sf *StreamFilter) compressLine(line string) string {
	if sf.cwd != "" {
		line = strings.ReplaceAll(line, sf.cwd+pathSep, "."+pathSep)
		line = strings.ReplaceAll(line, sf.cwd, ".")
	}
	if sf.home != "" {
		line = strings.ReplaceAll(line, sf.home+pathSep, "~"+pathSep)
		line = strings.ReplaceAll(line, sf.home, "~")
	}
	return line
}
