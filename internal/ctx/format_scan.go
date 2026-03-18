package ctx

import (
	"fmt"
	"strings"
)

// FormatScan formats results in scan mode: file path with match count,
// indented matches truncated to maxLine characters.
func FormatScan(results []FileResult, opts Options) string {
	if len(results) == 0 {
		return "[no matches found]\n"
	}

	var b strings.Builder
	for i, fr := range results {
		if i > 0 {
			b.WriteByte('\n')
		}
		b.WriteString(fmt.Sprintf("%dx %s\n", len(fr.Matches), fr.Path))
		for _, m := range fr.Matches {
			line := fmt.Sprintf("  %d: %s", m.LineNum, strings.TrimSpace(m.Line))
			if opts.MaxLine > 0 && len(line) > opts.MaxLine {
				line = line[:opts.MaxLine-3] + "..."
			}
			b.WriteString(line)
			b.WriteByte('\n')
		}
	}
	return b.String()
}
