package filter

import (
	"fmt"
	"regexp"
	"strings"
)

// --- Node.js stack traces ---

// compressNodeStackTraces detects Node.js error stacks and compresses them
// by keeping the error message and first 2 application frames, then
// summarizing node_modules and node internal frames.
func compressNodeStackTraces(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	i := 0

	for i < len(lines) {
		// Detect a Node.js stack trace: a line that looks like an error message
		// followed by "    at ..." lines.
		if i+1 < len(lines) && isNodeStackStart(lines[i], lines[i+1]) {
			// Error message line.
			result = append(result, lines[i])
			i++

			// Collect all stack frames.
			var appFrames []string
			nodeModulesCount := 0
			nodeInternalCount := 0

			for i < len(lines) && isNodeStackFrame(lines[i]) {
				trimmed := strings.TrimSpace(lines[i])
				if isNodeInternalFrame(trimmed) {
					nodeInternalCount++
				} else if isNodeModulesFrame(trimmed) {
					nodeModulesCount++
				} else {
					appFrames = append(appFrames, lines[i])
				}
				i++
			}

			// Emit first 2 application frames.
			limit := 2
			if len(appFrames) < limit {
				limit = len(appFrames)
			}
			for j := 0; j < limit; j++ {
				result = append(result, appFrames[j])
			}

			// Build summary of omitted frames.
			omittedApp := len(appFrames) - limit
			var parts []string
			if nodeModulesCount > 0 {
				parts = append(parts, fmt.Sprintf("%d node_modules frames", nodeModulesCount))
			}
			if nodeInternalCount > 0 {
				parts = append(parts, fmt.Sprintf("%d node internals", nodeInternalCount))
			}
			if omittedApp > 0 {
				parts = append(parts, fmt.Sprintf("%d more app frames", omittedApp))
			}
			if len(parts) > 0 {
				result = append(result, fmt.Sprintf("    [... %s]", strings.Join(parts, ", ")))
			}
		} else {
			result = append(result, lines[i])
			i++
		}
	}

	return strings.Join(result, "\n")
}

// nodeEmbeddedError finds an error class followed by its message anywhere on
// the line: console.error('Send failed:', err) prints "Send failed: Error: ..."
// before the stack, and Jest echoes such calls with the whole runner stack
// under them (issue #95). The class must start with an uppercase letter, so
// identifiers like "myError:" do not count.
var nodeEmbeddedError = regexp.MustCompile(`\b(?:[A-Z]\w*)?(?:Error|Exception): `)

// nodeFrameShape matches the line:col location V8 puts on every frame with a
// source. JVM frames carry a single line number, which keeps Java "Caused by:"
// blocks out of the looser embedded-error path.
var nodeFrameShape = regexp.MustCompile(`:\d+:\d+\)?\s*$|\((?:<anonymous>|node:[^)]*)\)\s*$`)

// isNodeStackStart reports whether header opens a Node.js stack whose first
// frame is firstFrame. A header that starts with an error class is trusted
// as before; one that only embeds it also needs a V8-shaped first frame.
func isNodeStackStart(header, firstFrame string) bool {
	if !isNodeStackFrame(firstFrame) {
		return false
	}
	if isNodeErrorLine(header) {
		return true
	}
	return nodeEmbeddedError.MatchString(header) && nodeFrameShape.MatchString(firstFrame)
}

func isNodeErrorLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	// Common patterns: "Error: ...", "TypeError: ...", "SomeCustomError: ...".
	colonIdx := strings.Index(trimmed, ": ")
	if colonIdx <= 0 {
		return false
	}
	prefix := trimmed[:colonIdx]
	if prefix[0] < 'A' || prefix[0] > 'Z' {
		return false
	}
	return strings.HasSuffix(prefix, "Error") || strings.HasSuffix(prefix, "Exception")
}

func isNodeStackFrame(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "at ")
}

// "(<anonymous>)" marks a frame in V8 builtins (new Promise, Array.forEach):
// it carries no file or line an LLM could act on.
func isNodeInternalFrame(trimmedLine string) bool {
	return strings.Contains(trimmedLine, "(node:") ||
		strings.Contains(trimmedLine, "(<anonymous>)") ||
		strings.Contains(trimmedLine, "node:internal/") ||
		strings.Contains(trimmedLine, "node:events") ||
		strings.Contains(trimmedLine, "node:net") ||
		strings.Contains(trimmedLine, "node:async_hooks") ||
		strings.Contains(trimmedLine, "node:stream") ||
		strings.Contains(trimmedLine, "node:fs") ||
		strings.Contains(trimmedLine, "node:_")
}

func isNodeModulesFrame(trimmedLine string) bool {
	return strings.Contains(trimmedLine, "node_modules/") || strings.Contains(trimmedLine, "node_modules\\")
}
