package detect

import (
	"regexp"
	"strings"
)

var (
	// tree summary line: "N directories, M files"
	treeSummaryLine = regexp.MustCompile(`^\d+ director(y|ies), \d+ files?`)
	// tree branch characters
	treeBranchChars = regexp.MustCompile(`^[│├└─┬┤┼╠╚═╔╗╝╬║\s|+\\`+"`"+`\-]+`)
)

// compressTreeOutput compresses tree command output by collapsing deep empty directories.
// Preserves: overall structure, file names, summary line.
// Removes: intermediate directories that contain only a single subdirectory (chain compression).
func compressTreeOutput(input string) string {
	lines := strings.Split(input, "\n")

	if len(lines) <= 20 {
		return input // small tree, keep as-is
	}

	var result []string
	emptyDirChain := []string{}

	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			if len(emptyDirChain) > 0 {
				result = append(result, flushDirChain(emptyDirChain)...)
				emptyDirChain = nil
			}
			result = append(result, line)
			continue
		}

		// Always keep the summary line
		if treeSummaryLine.MatchString(trimmed) {
			if len(emptyDirChain) > 0 {
				result = append(result, flushDirChain(emptyDirChain)...)
				emptyDirChain = nil
			}
			result = append(result, line)
			continue
		}

		// Detect directory lines (no extension, or end with /)
		name := extractTreeName(trimmed)
		depth := treeDepth(line)

		if isTreeDir(name) && idx+1 < len(lines) {
			nextDepth := treeDepth(lines[idx+1])
			// If next line is deeper, this is a parent dir — candidate for chain compression
			if nextDepth > depth {
				nextName := extractTreeName(strings.TrimSpace(lines[idx+1]))
				if isTreeDir(nextName) {
					emptyDirChain = append(emptyDirChain, name)
					continue
				}
			}
		}

		// Non-chainable: flush any pending chain
		if len(emptyDirChain) > 0 {
			emptyDirChain = append(emptyDirChain, name)
			result = append(result, flushDirChain(emptyDirChain)...)
			emptyDirChain = nil
			continue
		}

		result = append(result, line)
	}

	if len(emptyDirChain) > 0 {
		result = append(result, flushDirChain(emptyDirChain)...)
	}

	return strings.Join(result, "\n")
}

// extractTreeName gets the file/dir name from a tree line, stripping branch characters.
func extractTreeName(line string) string {
	// Remove tree drawing characters
	name := treeBranchChars.ReplaceAllString(line, "")
	name = strings.TrimSpace(name)
	return name
}

// treeDepth estimates nesting depth from indentation/branch characters.
func treeDepth(line string) int {
	depth := 0
	for _, ch := range line {
		if ch == ' ' || ch == '│' || ch == '|' || ch == '\t' {
			depth++
		} else if ch == '├' || ch == '└' || ch == '+' || ch == '`' {
			break
		} else {
			break
		}
	}
	return depth
}

// isTreeDir checks if a name looks like a directory (no extension or ends with /).
func isTreeDir(name string) bool {
	if name == "" {
		return false
	}
	if strings.HasSuffix(name, "/") {
		return true
	}
	// No dot in last path component = likely a directory
	return !strings.Contains(name, ".")
}

// flushDirChain collapses a chain of single-child directories into one line.
func flushDirChain(chain []string) []string {
	if len(chain) <= 2 {
		// Not worth collapsing
		var lines []string
		for _, name := range chain {
			lines = append(lines, "├── "+name)
		}
		return lines
	}
	// Collapse: "a/b/c/d"
	return []string{"├── " + strings.Join(chain, "/")}
}
