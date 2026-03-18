package ctx

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// skeletonPatterns define what constitutes "structural" lines per language.
var skeletonPatterns = map[string][]*regexp.Regexp{
	".go": {
		regexp.MustCompile(`^package\s+`),
		regexp.MustCompile(`^import\s+`),
		regexp.MustCompile(`^\s*"[^"]+"\s*$`), // import path
		regexp.MustCompile(`^type\s+\w+`),
		regexp.MustCompile(`^func\s+`),
		regexp.MustCompile(`^\)`), // closing import/const/var block
	},
	".py": {
		regexp.MustCompile(`^(import|from)\s+`),
		regexp.MustCompile(`^(class|def)\s+`),
		regexp.MustCompile(`^@\w+`), // decorators
	},
	".js": {
		regexp.MustCompile(`^(import|export|const|let|var|function|class)\s+`),
		regexp.MustCompile(`^(export\s+)?default\s+`),
		regexp.MustCompile(`^module\.exports`),
	},
	".ts": {
		regexp.MustCompile(`^(import|export|const|let|var|function|class|type|interface|enum)\s+`),
		regexp.MustCompile(`^(export\s+)?default\s+`),
	},
	".tsx": {
		regexp.MustCompile(`^(import|export|const|let|var|function|class|type|interface|enum)\s+`),
		regexp.MustCompile(`^(export\s+)?default\s+`),
	},
	".rs": {
		regexp.MustCompile(`^(use|mod|pub|fn|struct|enum|trait|impl|type|const|static)\s+`),
	},
	".java": {
		regexp.MustCompile(`^(package|import)\s+`),
		regexp.MustCompile(`^\s*(public|private|protected|static|final|abstract)*\s*(class|interface|enum|record)\s+`),
		regexp.MustCompile(`^\s*(public|private|protected|static|final|abstract)+\s+[\w<>\[\]]+\s+\w+\s*\(`),
	},
	".rb": {
		regexp.MustCompile(`^(require|include|extend|class|module|def|attr_)\s*`),
	},
	".c": {
		regexp.MustCompile(`^#include\s+`),
		regexp.MustCompile(`^#define\s+`),
		regexp.MustCompile(`^(typedef|struct|enum|union)\s+`),
		regexp.MustCompile(`^[\w*]+\s+\w+\s*\(`),
	},
	".h": {
		regexp.MustCompile(`^#(include|define|ifndef|ifdef|endif|pragma)\s*`),
		regexp.MustCompile(`^(typedef|struct|enum|union)\s+`),
		regexp.MustCompile(`^[\w*]+\s+\w+\s*\(`),
	},
}

// FormatTree formats results in tree mode: extracts the structural skeleton
// (imports, types, functions) for each file that had matches.
func FormatTree(results []FileResult, opts Options) string {
	if len(results) == 0 {
		return "[no matches found]\n"
	}

	var b strings.Builder

	for i, fr := range results {
		if i > 0 {
			b.WriteByte('\n')
		}

		ext := strings.ToLower(filepath.Ext(fr.Path))
		patterns := skeletonPatterns[ext]

		b.WriteString(fmt.Sprintf("== %s ==\n", fr.Path))

		lines, err := readFileLinesForTree(fr.Path)
		if err != nil {
			b.WriteString("  (read error)\n")
			continue
		}

		skeleton := extractSkeleton(lines, patterns)
		if len(skeleton) == 0 {
			b.WriteString("  (no structure detected)\n")
			continue
		}

		for _, sl := range skeleton {
			line := sl.line
			if opts.MaxLine > 0 && len(line)+8 > opts.MaxLine {
				line = line[:opts.MaxLine-11] + "..."
			}
			b.WriteString(fmt.Sprintf("  %d: %s\n", sl.num, line))
		}
	}
	return b.String()
}

type skeletonLine struct {
	num  int
	line string
}

func extractSkeleton(lines []string, patterns []*regexp.Regexp) []skeletonLine {
	var result []skeletonLine

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}
		if isSkeleton(trimmed, patterns) {
			result = append(result, skeletonLine{num: i + 1, line: trimmed})
		}
	}

	return result
}

func isSkeleton(line string, patterns []*regexp.Regexp) bool {
	for _, p := range patterns {
		if p.MatchString(line) {
			return true
		}
	}
	return false
}

func readFileLinesForTree(path string) ([]string, error) {
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
