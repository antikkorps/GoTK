package ctx

import (
	"path/filepath"
	"regexp"
	"strings"
)

// Language-aware declaration patterns.
// Each pattern is designed to match top-level declarations (functions, types, classes, etc.).
var defPatterns = map[string][]*regexp.Regexp{
	".go": {
		regexp.MustCompile(`^func\s+(\([^)]*\)\s+)?\w+`),       // func or method
		regexp.MustCompile(`^type\s+\w+\s+(struct|interface)`),   // type decl
		regexp.MustCompile(`^var\s+\w+`),                         // package-level var
		regexp.MustCompile(`^const\s+`),                          // const block
	},
	".py": {
		regexp.MustCompile(`^(class|def)\s+\w+`),
		regexp.MustCompile(`^\w+\s*=\s*`), // module-level assignment
	},
	".js": {
		regexp.MustCompile(`^(export\s+)?(function|class|const|let|var)\s+\w+`),
		regexp.MustCompile(`^(export\s+)?default\s+`),
		regexp.MustCompile(`^\w+\.\w+\s*=\s*function`),
	},
	".ts": {
		regexp.MustCompile(`^(export\s+)?(function|class|const|let|var|type|interface|enum)\s+\w+`),
		regexp.MustCompile(`^(export\s+)?default\s+`),
	},
	".tsx": {
		regexp.MustCompile(`^(export\s+)?(function|class|const|let|var|type|interface|enum)\s+\w+`),
		regexp.MustCompile(`^(export\s+)?default\s+`),
	},
	".jsx": {
		regexp.MustCompile(`^(export\s+)?(function|class|const|let|var)\s+\w+`),
		regexp.MustCompile(`^(export\s+)?default\s+`),
	},
	".rs": {
		regexp.MustCompile(`^(pub\s+)?(fn|struct|enum|trait|impl|type|const|static|mod)\s+`),
	},
	".java": {
		regexp.MustCompile(`^\s*(public|private|protected|static|final|abstract|synchronized)*\s*(class|interface|enum|record)\s+\w+`),
		regexp.MustCompile(`^\s*(public|private|protected|static|final|abstract|synchronized)+\s+[\w<>\[\]]+\s+\w+\s*\(`),
	},
	".rb": {
		regexp.MustCompile(`^(class|module|def)\s+\w+`),
	},
	".c": {
		regexp.MustCompile(`^[\w*]+\s+\w+\s*\(`),               // function definition
		regexp.MustCompile(`^(typedef|struct|enum|union)\s+`),
	},
	".h": {
		regexp.MustCompile(`^[\w*]+\s+\w+\s*\(`),
		regexp.MustCompile(`^(typedef|struct|enum|union)\s+`),
		regexp.MustCompile(`^#define\s+\w+`),
	},
	".cpp": {
		regexp.MustCompile(`^[\w:*&]+\s+[\w:]+\s*\(`),
		regexp.MustCompile(`^(class|struct|enum|namespace|template)\s+`),
	},
	".sh": {
		regexp.MustCompile(`^\w+\s*\(\)\s*\{`),                 // function def
		regexp.MustCompile(`^function\s+\w+`),
	},
}

// FormatDef formats results in def mode: shows matching lines that are
// language-aware declarations, falling back to keyword match.
func FormatDef(results []FileResult, opts Options) string {
	if len(results) == 0 {
		return "[no matches found]\n"
	}

	var b strings.Builder
	found := false

	for _, fr := range results {
		ext := strings.ToLower(filepath.Ext(fr.Path))
		patterns := defPatterns[ext]

		var defs []Match
		for _, m := range fr.Matches {
			trimmed := strings.TrimSpace(m.Line)
			if isDef(trimmed, patterns) {
				defs = append(defs, m)
			}
		}

		// Fallback: if no definitions matched, show all matches (keyword fallback)
		if len(defs) == 0 {
			defs = fr.Matches
		}

		if len(defs) > 0 {
			found = true
			b.WriteString(fr.Path + "\n")
			for _, m := range defs {
				line := strings.TrimSpace(m.Line)
				if opts.MaxLine > 0 && len(line)+8 > opts.MaxLine {
					line = line[:opts.MaxLine-11] + "..."
				}
				b.WriteString(strings.Repeat(" ", 2))
				b.WriteString(strings.Repeat(" ", 0))
				b.WriteString(formatInt(m.LineNum))
				b.WriteString(": ")
				b.WriteString(line)
				b.WriteByte('\n')
			}
			b.WriteByte('\n')
		}
	}

	if !found {
		return "[no definitions found]\n"
	}
	return b.String()
}

func isDef(line string, patterns []*regexp.Regexp) bool {
	for _, p := range patterns {
		if p.MatchString(line) {
			return true
		}
	}
	return false
}

func formatInt(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}
