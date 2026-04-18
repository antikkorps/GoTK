package mcp

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
)

// globExcludeDirs are directories skipped entirely during glob walk.
// Kept in sync with internal/ctx/walk.go (local copy to avoid import cycle).
var globExcludeDirs = map[string]bool{
	"node_modules": true,
	".git":         true,
	"dist":         true,
	"vendor":       true,
	"__pycache__":  true,
	".venv":        true,
	"venv":         true,
	"coverage":     true,
	".next":        true,
	".cache":       true,
	".idea":        true,
	".vscode":      true,
}

// globClusterThreshold is the number of results above which output is grouped
// by directory instead of listed line-by-line.
const globClusterThreshold = 30

// globDefaultMaxResults caps returned paths unless the caller overrides it.
// Glob output is path-only so it compresses poorly — a hard cap keeps the
// response small even for patterns that match thousands of files.
const globDefaultMaxResults = 500

// runGlob walks `root` and returns every file whose relative path matches
// `pattern` (shell-style glob). `**/` at the start of the pattern is stripped
// since the walk is already recursive. Patterns containing `/` are matched
// against the full relative path; patterns without `/` match against the
// basename. Excluded directories (node_modules, .git, …) are skipped.
func runGlob(pattern, root string, max int) ([]string, error) {
	if max <= 0 {
		max = globDefaultMaxResults
	}

	pat := strings.TrimPrefix(pattern, "**/")
	matchFull := strings.ContainsRune(pat, '/')

	var matches []string
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		// Don't follow symlinks — prevents escape outside the intended root.
		if d.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if d.IsDir() {
			if globExcludeDirs[d.Name()] {
				return filepath.SkipDir
			}
			return nil
		}

		var candidate string
		if matchFull {
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return nil
			}
			candidate = rel
		} else {
			candidate = d.Name()
		}

		ok, _ := filepath.Match(pat, candidate)
		if !ok {
			return nil
		}

		matches = append(matches, path)
		if len(matches) >= max {
			return filepath.SkipAll
		}
		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Strings(matches)
	return matches, nil
}

// formatGlobResults renders matches in one of two layouts depending on
// count: flat list for small result sets, or directory-clustered groups
// when there are many results (beyond globClusterThreshold). The cluster
// form keeps file names on shared lines to cut token usage dramatically
// when a pattern matches across many folders.
func formatGlobResults(matches []string, root string, forceCluster *bool) string {
	if len(matches) == 0 {
		return "[no matches]\n"
	}

	cluster := len(matches) >= globClusterThreshold
	if forceCluster != nil {
		cluster = *forceCluster
	}

	if !cluster {
		var b strings.Builder
		for _, m := range matches {
			rel, err := filepath.Rel(root, m)
			if err != nil {
				rel = m
			}
			b.WriteString(rel)
			b.WriteByte('\n')
		}
		return b.String()
	}

	// Group by parent directory (relative to root).
	groups := make(map[string][]string)
	var order []string
	for _, m := range matches {
		rel, err := filepath.Rel(root, m)
		if err != nil {
			rel = m
		}
		dir := filepath.Dir(rel)
		if dir == "." {
			dir = "./"
		} else {
			dir += "/"
		}
		if _, seen := groups[dir]; !seen {
			order = append(order, dir)
		}
		groups[dir] = append(groups[dir], filepath.Base(rel))
	}
	sort.Strings(order)

	var b strings.Builder
	for _, dir := range order {
		files := groups[dir]
		fmt.Fprintf(&b, "%s (%d): %s\n", dir, len(files), strings.Join(files, " "))
	}
	return b.String()
}

// capGrepMatchesPerFile walks grouped grep output (as produced by
// detect.compressGrepOutput — "lines starting with '>> file'") and caps the
// number of match lines displayed per file at perFile, replacing the rest
// with a single "… N more in this file" marker. This prevents one verbose
// file from crowding out every other file in the result.
//
// Input format expected:
//
//	>> path/to/file.go
//	  42:some match
//	  43:another match
//	>> path/to/other.go
//	  ...
//
// Lines not matching this format are passed through untouched.
func capGrepMatchesPerFile(input string, perFile int) string {
	if perFile <= 0 {
		return input
	}
	lines := strings.Split(input, "\n")
	out := make([]string, 0, len(lines))
	matches := 0
	overflowed := false
	flushOverflow := func() {
		if overflowed && matches > perFile {
			out = append(out, fmt.Sprintf("  … %d more in this file", matches-perFile))
		}
		matches = 0
		overflowed = false
	}

	for _, line := range lines {
		if strings.HasPrefix(line, ">> ") {
			flushOverflow()
			out = append(out, line)
			continue
		}
		if strings.TrimSpace(line) == "" {
			flushOverflow()
			out = append(out, line)
			continue
		}
		// Match line under a file header (indented with two spaces by compressGrepOutput).
		if strings.HasPrefix(line, "  ") {
			matches++
			if matches <= perFile {
				out = append(out, line)
			} else {
				overflowed = true
			}
			continue
		}
		// Any other line — flush overflow and pass through.
		flushOverflow()
		out = append(out, line)
	}
	flushOverflow()

	return strings.Join(out, "\n")
}
