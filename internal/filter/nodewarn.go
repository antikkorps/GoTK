package filter

import (
	"fmt"
	"regexp"
	"strings"
)

// nodeWarnHeader matches the first line of a Node.js native warning block:
//
//	(node:12345) Warning: `--flag` was provided without a valid path
//
// The PID varies per worker; on parallel test runs (jest --workers, vitest
// threads) the exact same warning fires N times, byte-identical except for
// the PID. The generic Dedup filter requires strictly consecutive identical
// lines, which this pattern defeats because each worker emits (header, body)
// interleaved. See issue #37.
var nodeWarnHeader = regexp.MustCompile(`^\(node:\d+\)\s`)

// nodeWarnPID normalizes the PID so two warnings from different workers
// compare equal.
var nodeWarnPID = regexp.MustCompile(`^\(node:\d+\)`)

// CollapseNodeWarnings groups strictly consecutive Node.js native warning
// blocks that differ only by PID and replaces duplicate blocks with a count
// marker. The first block is emitted verbatim (with its original PID) so the
// reader can still trace one example; the remaining duplicates become one
// "... repeated N more times" line.
//
// A "block" is the (node:PID) header line plus any continuation lines up to
// the next (node:PID) header, a blank line, or any line that looks like a
// distinct piece of output (no leading whitespace and no opening parenthesis).
// This keeps the collapse narrow: only the classic Node warning output format
// is touched, everything else passes through.
func CollapseNodeWarnings(input string) string {
	if input == "" {
		return input
	}
	lines := strings.Split(input, "\n")
	trailingNewline := len(lines) > 0 && lines[len(lines)-1] == ""
	if trailingNewline {
		lines = lines[:len(lines)-1]
	}

	var out []string
	i := 0
	for i < len(lines) {
		if !nodeWarnHeader.MatchString(lines[i]) {
			out = append(out, lines[i])
			i++
			continue
		}

		// Collect consecutive node warning blocks starting at i.
		blocks, next := collectNodeWarnBlocks(lines, i)
		if len(blocks) == 1 {
			// No duplicates — emit the block as-is.
			out = append(out, blocks[0]...)
			i = next
			continue
		}

		// Emit the first block verbatim, then a single count marker.
		out = append(out, blocks[0]...)
		extra := len(blocks) - 1
		noun := "time"
		if extra > 1 {
			noun = "times"
		}
		out = append(out, fmt.Sprintf("  ... (repeated %d more %s, different PIDs)", extra, noun))
		i = next
	}

	result := strings.Join(out, "\n")
	if trailingNewline {
		result += "\n"
	}
	return result
}

// collectNodeWarnBlocks pulls consecutive node warning blocks starting at
// lines[start]. Blocks are grouped only when their PID-normalized content is
// identical. Returns the collected blocks (each itself a slice of lines) and
// the index of the first line after the run.
func collectNodeWarnBlocks(lines []string, start int) ([][]string, int) {
	var blocks [][]string
	i := start
	var sig string

	for i < len(lines) && nodeWarnHeader.MatchString(lines[i]) {
		block := []string{lines[i]}
		j := i + 1
		for j < len(lines) && isNodeWarnContinuation(lines[j]) {
			block = append(block, lines[j])
			j++
		}
		blockSig := normalizeNodeWarnBlock(block)
		if len(blocks) == 0 {
			sig = blockSig
			blocks = append(blocks, block)
			i = j
			continue
		}
		if blockSig != sig {
			// Different warning kind — stop grouping here so the caller can
			// start a fresh run.
			break
		}
		blocks = append(blocks, block)
		i = j
	}

	return blocks, i
}

// isNodeWarnContinuation reports whether a line belongs to the currently-open
// node warning block. Node emits trailer lines like:
//
//	(Use `node --trace-warnings ...` to show where the warning was created)
//
// or indented detail. A blank line, a new (node:PID) header, or any other
// unindented non-parenthesized content ends the block.
func isNodeWarnContinuation(line string) bool {
	if line == "" {
		return false
	}
	if nodeWarnHeader.MatchString(line) {
		return false
	}
	if strings.HasPrefix(line, "(") {
		return true
	}
	if line[0] == ' ' || line[0] == '\t' {
		return true
	}
	return false
}

// normalizeNodeWarnBlock builds a comparison signature by stripping the PID.
func normalizeNodeWarnBlock(block []string) string {
	norm := make([]string, len(block))
	for i, l := range block {
		norm[i] = nodeWarnPID.ReplaceAllString(l, "(node:*)")
	}
	return strings.Join(norm, "\n")
}
