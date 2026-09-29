package filter

import (
	"strconv"
	"strings"
)

// Dedup removes consecutive duplicate lines.
// Keeps the first occurrence and replaces runs with a count marker.
func Dedup(input string) string {
	if input == "" {
		return input
	}

	lines := strings.Split(input, "\n")

	var result []string
	prev := lines[0]
	result = append(result, prev)
	dupCount := 0

	lines = lines[1:]

	for _, line := range lines {
		if line == prev {
			dupCount++
			continue
		}

		if dupCount > 0 {
			result = append(result, formatDupMarker(dupCount))
			dupCount = 0
		}

		result = append(result, line)
		prev = line
	}

	// Handle trailing duplicates
	if dupCount > 0 {
		result = append(result, formatDupMarker(dupCount))
	}

	return strings.Join(result, "\n")
}

// maxBlockLines bounds the size of a repeated block DedupBlocks looks for.
// Large enough for a typical stack trace or Jest console block, small enough
// to keep the scan cheap on multi-megabyte outputs.
const maxBlockLines = 30

// DedupBlocks collapses consecutive, byte-identical multi-line blocks, the
// case Dedup cannot see because it compares single lines. Typical sources: the
// same stack trace logged on every request by a test web server (issue #84),
// or a Jest console.log block emitted once per suite (issue #87).
//
// The first copy is kept verbatim and the rest become one count marker, so
// the frequency information survives. Only exact repeats are collapsed:
// blocks that differ by a single byte are left alone.
func DedupBlocks(input string) string {
	if input == "" {
		return input
	}
	lines := strings.Split(input, "\n")
	trailingNewline := lines[len(lines)-1] == ""
	if trailingNewline {
		lines = lines[:len(lines)-1]
	}

	out := make([]string, 0, len(lines))
	i := 0
	for i < len(lines) {
		size, reps := longestRepeat(lines, i)
		if reps < 2 {
			out = append(out, lines[i])
			i++
			continue
		}
		out = append(out, lines[i:i+size]...)
		out = append(out, formatBlockDupMarker(size, reps-1))
		i += size * reps
	}

	result := strings.Join(out, "\n")
	if trailingNewline {
		result += "\n"
	}
	return result
}

// longestRepeat finds the block size (2..maxBlockLines) starting at start
// whose consecutive repetitions cover the most lines. It returns the block
// size and repetition count; reps < 2 means nothing repeats. Ties favor the
// smallest block.
func longestRepeat(lines []string, start int) (size, reps int) {
	for k := 2; k <= maxBlockLines && start+2*k <= len(lines); k++ {
		if lines[start] != lines[start+k] || isBlankBlock(lines[start:start+k]) {
			continue
		}
		n := 1
		for start+(n+1)*k <= len(lines) && blocksEqual(lines, start, start+n*k, k) {
			n++
		}
		if n >= 2 && n*k > size*reps {
			size, reps = k, n
		}
	}
	return size, reps
}

func blocksEqual(lines []string, a, b, k int) bool {
	for j := 0; j < k; j++ {
		if lines[a+j] != lines[b+j] {
			return false
		}
	}
	return true
}

func isBlankBlock(block []string) bool {
	for _, l := range block {
		if strings.TrimSpace(l) != "" {
			return false
		}
	}
	return true
}

func formatBlockDupMarker(size, extra int) string {
	times := "times"
	if extra == 1 {
		times = "time"
	}
	return "  ... (previous " + strconv.Itoa(size) + "-line block repeated " +
		strconv.Itoa(extra) + " more " + times + ")"
}

func formatDupMarker(count int) string {
	if count == 1 {
		return "  ... (1 duplicate line)"
	}
	return "  ... (" + strconv.Itoa(count) + " duplicate lines)"
}
