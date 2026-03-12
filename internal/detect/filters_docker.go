package detect

import (
	"regexp"
	"strings"
)

var (
	// Docker build step prefix: "Step N/M :" or "STEP N/M:"
	dockerStepPattern = regexp.MustCompile(`^Step \d+/\d+ :`)
	// Intermediate container hash lines
	dockerRunningIn  = regexp.MustCompile(`^---> Running in [0-9a-f]+`)
	dockerArrowHash  = regexp.MustCompile(`^---> [0-9a-f]{12}$`)
	dockerRemoving   = regexp.MustCompile(`^Removing intermediate container [0-9a-f]+`)
	// Docker pull layer progress lines (e.g., "abc123: Pulling fs layer", "abc123: Downloading  [==>  ]")
	dockerLayerProgress = regexp.MustCompile(`^[0-9a-f]{12}: (Waiting|Pulling fs layer|Downloading|Extracting|Verifying Checksum|Download complete|Pull complete)`)
	// ANSI spinner/progress (common in docker compose)
	ansiProgressLine = regexp.MustCompile(`\x1b\[[0-9;]*[mGKHJ]`)
)

// compressDockerOutput removes redundant docker build/pull/compose output.
func compressDockerOutput(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	layerCount := 0
	var pullFrom string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip empty lines
		if trimmed == "" {
			continue
		}

		// Docker build: skip intermediate container lines
		if dockerRunningIn.MatchString(trimmed) {
			continue
		}
		if dockerArrowHash.MatchString(trimmed) {
			continue
		}
		if dockerRemoving.MatchString(trimmed) {
			continue
		}

		// Docker build steps: keep the command description, strip "Step N/M :" prefix
		if dockerStepPattern.MatchString(trimmed) {
			// Extract just the command after "Step N/M : "
			parts := strings.SplitN(trimmed, ": ", 2)
			if len(parts) == 2 {
				result = append(result, parts[1])
			} else {
				result = append(result, trimmed)
			}
			continue
		}

		// Docker pull: compress layer progress
		if strings.HasPrefix(trimmed, "Pulling from ") || strings.HasPrefix(trimmed, "Pulling ") {
			pullFrom = trimmed
			layerCount = 0
			continue
		}
		if dockerLayerProgress.MatchString(trimmed) {
			layerCount++
			continue
		}

		// Flush pull summary before a non-pull line
		if pullFrom != "" {
			if layerCount > 0 {
				result = append(result, pullFrom+" ("+itoa(layerCount)+" layers)")
			} else {
				result = append(result, pullFrom)
			}
			pullFrom = ""
			layerCount = 0
		}

		// Docker compose: skip pure ANSI progress lines (no real text content)
		cleaned := ansiProgressLine.ReplaceAllString(trimmed, "")
		if strings.TrimSpace(cleaned) == "" {
			continue
		}

		// Keep errors, FROM, RUN, COPY, and all other meaningful lines
		result = append(result, line)
	}

	// Flush trailing pull summary
	if pullFrom != "" {
		if layerCount > 0 {
			result = append(result, pullFrom+" ("+itoa(layerCount)+" layers)")
		} else {
			result = append(result, pullFrom)
		}
	}

	return strings.Join(result, "\n")
}
