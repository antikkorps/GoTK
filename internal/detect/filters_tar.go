package detect

import (
	"regexp"
	"strings"
)

var (
	// tar verbose listing: "drwxr-xr-x user/group  0 2024-01-15 10:30 path/"
	tarVerboseLine = regexp.MustCompile(`^[drwxlstST\-]{10}\s+\S+/\S+\s+\d+\s+\d{4}-\d{2}-\d{2}\s+\d{2}:\d{2}\s+`)
	// tar extraction progress: "x path/to/file"
	tarExtractLine = regexp.MustCompile(`^x\s+\S`)
	// zip listing header: "  Length      Date    Time    Name"
	zipListHeader = regexp.MustCompile(`^\s*Length\s+Date\s+Time\s+Name`)
	// zip separator: "  --------    ---------- -----   ----"
	zipSeparator = regexp.MustCompile(`^\s*-{5,}`)
	// gzip verbose: "path:    80.5% -- replaced with path.gz"
	gzipVerbose = regexp.MustCompile(`:\s+[\d.]+%\s+--\s+(created|replaced)`)
)

// compressTarOutput compresses tar/zip listing and extraction output.
// Preserves: file names, errors, summary counts.
// Removes: verbose metadata (permissions, user/group, dates), extraction progress for large archives.
func compressTarOutput(input string) string {
	lines := strings.Split(input, "\n")

	if len(lines) <= 30 {
		return input // small listing, keep as-is
	}

	var result []string
	extractCount := 0
	var fileList []string
	inVerboseListing := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			flushTarCounters(&result, &extractCount, &fileList)
			result = append(result, line)
			continue
		}

		// Compress tar verbose listing: strip metadata, keep paths
		if tarVerboseLine.MatchString(trimmed) {
			inVerboseListing = true
			// Extract just the path (last field)
			fields := strings.Fields(trimmed)
			if len(fields) >= 6 {
				path := fields[len(fields)-1]
				fileList = append(fileList, path)
			}
			continue
		}

		// Compress tar extraction progress lines
		if tarExtractLine.MatchString(trimmed) {
			extractCount++
			continue
		}

		// Skip zip separators
		if zipSeparator.MatchString(trimmed) {
			continue
		}

		// Skip zip listing header
		if zipListHeader.MatchString(trimmed) {
			result = append(result, "Archive contents:")
			continue
		}

		// Skip gzip verbose percentage lines
		if gzipVerbose.MatchString(trimmed) {
			continue
		}

		// Flush counters before other content
		flushTarCounters(&result, &extractCount, &fileList)
		inVerboseListing = false

		result = append(result, line)
	}

	flushTarCounters(&result, &extractCount, &fileList)
	_ = inVerboseListing

	return strings.Join(result, "\n")
}

func flushTarCounters(result *[]string, extractCount *int, fileList *[]string) {
	if len(*fileList) > 0 {
		if len(*fileList) <= 20 {
			for _, f := range *fileList {
				*result = append(*result, f)
			}
		} else {
			// Show first 10, last 5, and count
			for _, f := range (*fileList)[:10] {
				*result = append(*result, f)
			}
			*result = append(*result, "... "+itoa(len(*fileList)-15)+" more files ...")
			for _, f := range (*fileList)[len(*fileList)-5:] {
				*result = append(*result, f)
			}
		}
		*fileList = nil
	}
	if *extractCount > 0 {
		*result = append(*result, "Extracted "+itoa(*extractCount)+" files")
		*extractCount = 0
	}
}
