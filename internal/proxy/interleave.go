package proxy

import (
	"fmt"
	"io"
	"math"
	"sort"
	"strings"

	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/exec"
	"github.com/antikkorps/GoTK/internal/filter"
)

// Piece is a chunk of final output bound for one stream.
type Piece struct {
	Stderr bool
	Text   string
}

// FilterStderrBlocks runs the stderr policy chain on each block separately
// (so each keeps its position) and drops blocks that end up empty.
func FilterStderrBlocks(cfg *config.Config, cmdType detect.CmdType, blocks []exec.StderrBlock) []exec.StderrBlock {
	chain := BuildStderrChain(cfg, cmdType)
	var out []exec.StderrBlock
	for _, b := range mergeOpenPrivateKeys(blocks) {
		if text := chain.Apply(b.Text); text != "" {
			out = append(out, exec.StderrBlock{AfterLine: b.AfterLine, Text: text})
		}
	}
	return out
}

// mergeOpenPrivateKeys joins a block that opens a PEM private key with the
// following ones until the key closes: secret redaction matches the whole
// BEGIN..END span, so a key split by interleaved stdout would leak. The
// merged block keeps the first position (moving stderr up is allowed).
func mergeOpenPrivateKeys(blocks []exec.StderrBlock) []exec.StderrBlock {
	var out []exec.StderrBlock
	open := false
	for _, b := range blocks {
		if open {
			out[len(out)-1].Text += b.Text
		} else {
			out = append(out, b)
		}
		open = filter.HasOpenPrivateKey(out[len(out)-1].Text)
	}
	return out
}

// MergeStreams returns stdout and raw stderr as one text in arrival order,
// for single-channel consumers such as the MCP server.
func MergeStreams(res *exec.Result) string {
	blocks := res.StderrBlocks
	if len(blocks) == 0 && res.Stderr != "" {
		blocks = []exec.StderrBlock{{AfterLine: math.MaxInt, Text: res.Stderr}}
	}
	var b strings.Builder
	WritePieces(&b, &b, Interleave(res.Stdout, res.Stdout, blocks))
	return b.String()
}

// Interleave puts the (already filtered) stderr blocks back between the
// lines of the filtered stdout (issue #100). stdout is filtered as a whole,
// so each filtered line is first traced to its raw line; a block then goes
// before the first filtered line whose raw origin came at or after it. A
// block whose neighbours were dropped therefore moves up to the nearest
// surviving line, never past stdout that was written after it.
func Interleave(rawOut, cleanOut string, blocks []exec.StderrBlock) []Piece {
	lines := strings.Split(cleanOut, "\n")
	origin := traceOrigins(rawOut, lines)
	var pieces []Piece
	from := 0
	for _, b := range blocks {
		at := from + sort.Search(len(lines)-from, func(i int) bool { return origin[from+i] >= b.AfterLine })
		if at > from {
			text := strings.Join(lines[from:at], "\n")
			if at < len(lines) { // the newline that separated lines[at-1] from lines[at]
				text += "\n"
			}
			if text != "" {
				pieces = append(pieces, Piece{Text: text})
			}
			from = at
		}
		pieces = append(pieces, Piece{Stderr: true, Text: b.Text})
	}
	if rest := strings.Join(lines[from:], "\n"); rest != "" {
		pieces = append(pieces, Piece{Text: rest})
	}
	return pieces
}

// traceOrigins maps each filtered line to the index of the raw line it came
// from, matching in order on a whitespace- and ANSI-insensitive key (raw
// paths are compressed first, as the stdout chain does). A line with no
// match (summary marker, rewritten line, blank) takes the origin of the next
// matched line: if wrong, that only moves a block up, never past stdout
// written after it. The result is non-decreasing.
func traceOrigins(rawOut string, cleanLines []string) []int {
	rawLines := strings.Split(filter.CompressPaths(rawOut), "\n")
	index := map[string][]int{}
	for i, line := range rawLines {
		if k := matchKey(line); k != "" {
			index[k] = append(index[k], i)
		}
	}
	origin := make([]int, len(cleanLines))
	cursor := 0
	for i, line := range cleanLines {
		candidates := index[matchKey(line)] // blank keys are never indexed
		j := sort.SearchInts(candidates, cursor)
		if j == len(candidates) {
			origin[i] = -1
			continue
		}
		origin[i], cursor = candidates[j], candidates[j]+1
	}
	next := len(rawLines)
	for i := len(origin) - 1; i >= 0; i-- {
		if origin[i] < 0 {
			origin[i] = next
		} else {
			next = origin[i]
		}
	}
	return origin
}

func matchKey(line string) string {
	return strings.Join(strings.Fields(filter.StripANSI(line)), " ")
}

// WritePieces emits pieces to their streams in order. A newline is added
// to a piece's stream when it lacks one and more output follows, so lines
// never fuse under 2>&1 (issue #86). Returns the last text written.
func WritePieces(stdout, stderr io.Writer, pieces []Piece) string {
	last := ""
	for i, p := range pieces {
		w := stdout
		if p.Stderr {
			w = stderr
		}
		fmt.Fprint(w, p.Text) //nolint:errcheck
		if i+1 < len(pieces) && !strings.HasSuffix(p.Text, "\n") {
			fmt.Fprintln(w) //nolint:errcheck
		}
		last = p.Text
	}
	return last
}
