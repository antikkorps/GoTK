package ctx

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

// FormatDetail formats results in detail mode: N-line context windows
// around each match, with overlapping windows merged.
func FormatDetail(results []FileResult, opts Options) string {
	if len(results) == 0 {
		return "[no matches found]\n"
	}

	ctx := opts.Context
	if ctx <= 0 {
		ctx = 3
	}

	var b strings.Builder
	for i, fr := range results {
		if i > 0 {
			b.WriteByte('\n')
		}

		// Read full file into memory for context extraction
		lines, err := readFileLines(fr.Path)
		if err != nil {
			b.WriteString(fmt.Sprintf("--- %s (read error) ---\n", fr.Path))
			continue
		}

		// Build merged windows from match positions
		windows := mergeWindows(fr.Matches, ctx, len(lines))

		for wi, w := range windows {
			if wi > 0 {
				b.WriteString("  ...\n")
			}
			b.WriteString(fmt.Sprintf("--- %s:%d ---\n", fr.Path, w.start+1))
			for ln := w.start; ln < w.end; ln++ {
				prefix := "  "
				if isMatchLine(fr.Matches, ln+1) {
					prefix = "> "
				}
				b.WriteString(fmt.Sprintf("%s%d: %s\n", prefix, ln+1, lines[ln]))
			}
		}
	}
	return b.String()
}

type window struct {
	start, end int // 0-based line indices, end is exclusive
}

// mergeWindows computes context windows around matches and merges overlapping ones.
func mergeWindows(matches []Match, ctx, totalLines int) []window {
	if len(matches) == 0 {
		return nil
	}

	var windows []window
	for _, m := range matches {
		start := m.LineNum - 1 - ctx // 0-based
		if start < 0 {
			start = 0
		}
		end := m.LineNum + ctx // 0-based exclusive
		if end > totalLines {
			end = totalLines
		}
		windows = append(windows, window{start, end})
	}

	// Merge overlapping windows
	merged := []window{windows[0]}
	for _, w := range windows[1:] {
		last := &merged[len(merged)-1]
		if w.start <= last.end {
			if w.end > last.end {
				last.end = w.end
			}
		} else {
			merged = append(merged, w)
		}
	}

	return merged
}

func isMatchLine(matches []Match, lineNum int) bool {
	for _, m := range matches {
		if m.LineNum == lineNum {
			return true
		}
	}
	return false
}

func readFileLines(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var lines []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
