package ctx

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// FormatSummary formats results in summary mode: directory breakdown table
// with file counts and total matches.
func FormatSummary(results []FileResult, opts Options) string {
	if len(results) == 0 {
		return "[no matches found]\n"
	}

	// Aggregate by directory
	type dirStats struct {
		files   int
		matches int
	}
	dirs := map[string]*dirStats{}
	var order []string

	totalFiles := 0
	totalMatches := 0

	for _, fr := range results {
		dir := filepath.Dir(fr.Path)
		if dir == "" {
			dir = "."
		}
		if _, exists := dirs[dir]; !exists {
			dirs[dir] = &dirStats{}
			order = append(order, dir)
		}
		dirs[dir].files++
		dirs[dir].matches += len(fr.Matches)
		totalFiles++
		totalMatches += len(fr.Matches)
	}

	// Sort by match count descending
	sort.Slice(order, func(i, j int) bool {
		return dirs[order[i]].matches > dirs[order[j]].matches
	})

	var b strings.Builder
	fmt.Fprintf(&b, "Pattern: %s\n", opts.Pattern)
	fmt.Fprintf(&b, "Total: %d matches in %d files\n\n", totalMatches, totalFiles)
	fmt.Fprintf(&b, "%-40s %6s %8s\n", "Directory", "Files", "Matches")
	b.WriteString(strings.Repeat("-", 56) + "\n")

	for _, dir := range order {
		ds := dirs[dir]
		fmt.Fprintf(&b, "%-40s %6d %8d\n", truncDir(dir, 40), ds.files, ds.matches)
	}

	return b.String()
}

func truncDir(dir string, maxLen int) string {
	if len(dir) <= maxLen {
		return dir
	}
	return "..." + dir[len(dir)-maxLen+3:]
}
