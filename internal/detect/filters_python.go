package detect

import (
	"regexp"
	"strings"
)

var (
	// Python traceback header
	pyTracebackHeader = regexp.MustCompile(`^Traceback \(most recent call last\):`)
	// Python traceback frame: '  File "path", line N, in func' (matched against raw line, not trimmed)
	pyTracebackFrame = regexp.MustCompile(`^\s*File "(.+)", line (\d+), in (.+)`)
	// Python traceback code line (indented, follows a File line)
	pyTracebackCode = regexp.MustCompile(`^\s{4}\S`)
	// pip download/install progress
	pipDownloadProgress = regexp.MustCompile(`^\s*(Downloading|Installing|Collecting|Using cached)\s+`)
	pipProgressBar      = regexp.MustCompile(`\s+\|[█▓▒░#=\-\s]+\|\s+[\d.]+\s*[kMG]?B`)
	// pip "already satisfied" lines
	pipSatisfied = regexp.MustCompile(`^Requirement already satisfied:`)
	// Python deprecation warnings
	pyDeprecationWarning = regexp.MustCompile(`^\S+:\d+:\s*(DeprecationWarning|PendingDeprecationWarning|FutureWarning):`)
)

// compressPythonOutput compresses python/pip output.
// Preserves: tracebacks (condensed), errors, final pip summary.
// Removes: pip progress bars, "already satisfied" spam, verbose download lines.
func compressPythonOutput(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	satisfiedCount := 0
	downloadCount := 0
	deprecationCount := 0
	firstDeprecation := ""

	i := 0
	for i < len(lines) {
		line := lines[i]
		trimmed := strings.TrimSpace(line)

		// Compress pip "Requirement already satisfied" lines
		if pipSatisfied.MatchString(trimmed) {
			satisfiedCount++
			i++
			continue
		}

		// Compress pip download/install progress
		if pipDownloadProgress.MatchString(trimmed) {
			downloadCount++
			i++
			continue
		}

		// Skip pip progress bars
		if pipProgressBar.MatchString(trimmed) {
			i++
			continue
		}

		// Compress deprecation warnings: keep first, count rest
		if pyDeprecationWarning.MatchString(trimmed) {
			deprecationCount++
			if deprecationCount == 1 {
				firstDeprecation = line
			}
			i++
			continue
		}

		// Flush counters before other content
		if satisfiedCount > 0 {
			result = append(result, "Already satisfied: "+itoa(satisfiedCount)+" packages")
			satisfiedCount = 0
		}
		if downloadCount > 0 {
			result = append(result, "pip: "+itoa(downloadCount)+" packages downloaded/installed")
			downloadCount = 0
		}
		if deprecationCount > 0 {
			result = append(result, firstDeprecation)
			if deprecationCount > 1 {
				result = append(result, "... and "+itoa(deprecationCount-1)+" more deprecation warnings")
			}
			deprecationCount = 0
			firstDeprecation = ""
		}

		// Condense Python tracebacks: keep header, first frame, last frame + error
		if pyTracebackHeader.MatchString(trimmed) {
			result = append(result, line)
			i++
			frames := collectTracebackFrames(lines, &i)
			if len(frames) > 0 {
				if len(frames) > 2 {
					result = append(result, frames[0].fileLine)
					result = append(result, frames[0].codeLine)
					result = append(result, "  ... "+itoa(len(frames)-2)+" more frames ...")
					last := frames[len(frames)-1]
					result = append(result, last.fileLine)
					result = append(result, last.codeLine)
				} else {
					for _, f := range frames {
						result = append(result, f.fileLine)
						if f.codeLine != "" {
							result = append(result, f.codeLine)
						}
					}
				}
			}
			// The error line follows the frames
			if i < len(lines) {
				result = append(result, lines[i])
				i++
			}
			continue
		}

		result = append(result, line)
		i++
	}

	// Flush trailing counters
	if satisfiedCount > 0 {
		result = append(result, "Already satisfied: "+itoa(satisfiedCount)+" packages")
	}
	if downloadCount > 0 {
		result = append(result, "pip: "+itoa(downloadCount)+" packages downloaded/installed")
	}
	if deprecationCount > 0 {
		result = append(result, firstDeprecation)
		if deprecationCount > 1 {
			result = append(result, "... and "+itoa(deprecationCount-1)+" more deprecation warnings")
		}
	}

	return strings.Join(result, "\n")
}

type pyFrame struct {
	fileLine string
	codeLine string
}

// collectTracebackFrames collects File/code line pairs from a traceback.
func collectTracebackFrames(lines []string, i *int) []pyFrame {
	var frames []pyFrame
	for *i < len(lines) {
		line := lines[*i]
		if pyTracebackFrame.MatchString(line) {
			frame := pyFrame{fileLine: line}
			*i++
			// Next line might be the code line
			if *i < len(lines) && pyTracebackCode.MatchString(lines[*i]) {
				frame.codeLine = lines[*i]
				*i++
			}
			frames = append(frames, frame)
		} else {
			break
		}
	}
	return frames
}
