package detect

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	cargoCompilingPattern   = regexp.MustCompile(`^\s*Compiling\s+\S+\s+v`)
	cargoDownloadingPattern = regexp.MustCompile(`^\s*Downloading\s+`)
	cargoDownloadedPattern  = regexp.MustCompile(`^\s*Downloaded\s+`)
)

// compressCargoOutput compresses cargo build/test output by summarizing
// repetitive compilation and download lines while preserving errors and warnings.
func compressCargoOutput(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	compilingCount := 0
	downloadingCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Count consecutive "Compiling" lines
		if cargoCompilingPattern.MatchString(trimmed) {
			compilingCount++
			continue
		}

		// Count consecutive "Downloading"/"Downloaded" lines
		if cargoDownloadingPattern.MatchString(trimmed) || cargoDownloadedPattern.MatchString(trimmed) {
			downloadingCount++
			continue
		}

		// Flush compiling summary before a non-compiling line
		if compilingCount > 0 {
			result = append(result, "   Compiled "+strconv.Itoa(compilingCount)+" crates")
			compilingCount = 0
		}

		// Flush downloading summary before a non-downloading line
		if downloadingCount > 0 {
			result = append(result, "   Downloaded "+strconv.Itoa(downloadingCount)+" crates")
			downloadingCount = 0
		}

		// Always keep error and warning lines with context
		// Always keep "Finished" summary lines
		// Always keep test names and results
		result = append(result, line)
	}

	// Flush trailing counts
	if compilingCount > 0 {
		result = append(result, "   Compiled "+strconv.Itoa(compilingCount)+" crates")
	}
	if downloadingCount > 0 {
		result = append(result, "   Downloaded "+strconv.Itoa(downloadingCount)+" crates")
	}

	return strings.Join(result, "\n")
}
