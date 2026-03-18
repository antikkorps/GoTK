package ctx

import (
	"bufio"
	"os"
	"regexp"
)

// Match represents a single matching line within a file.
type Match struct {
	LineNum int
	Line    string
}

// FileResult holds all matches for a single file.
type FileResult struct {
	Path       string
	Matches    []Match
	TotalLines int
}

// Search scans the given files for lines matching opts.Pattern and returns results.
// Stops early if opts.MaxResults > 0 and enough files have been found.
func Search(files []string, opts Options) ([]FileResult, error) {
	if opts.Pattern == "" {
		return nil, nil
	}

	re, err := regexp.Compile(opts.Pattern)
	if err != nil {
		return nil, err
	}

	var results []FileResult

	for _, path := range files {
		fr, err := searchFile(path, re)
		if err != nil {
			continue // skip unreadable files
		}
		if len(fr.Matches) > 0 {
			results = append(results, fr)
			if opts.MaxResults > 0 && len(results) >= opts.MaxResults {
				break
			}
		}
	}

	return results, nil
}

// searchFile scans a single file for regex matches.
func searchFile(path string, re *regexp.Regexp) (FileResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return FileResult{}, err
	}
	defer f.Close()

	fr := FileResult{Path: path}
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if re.MatchString(line) {
			fr.Matches = append(fr.Matches, Match{LineNum: lineNum, Line: line})
		}
	}
	fr.TotalLines = lineNum

	return fr, scanner.Err()
}
