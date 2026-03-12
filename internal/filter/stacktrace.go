package filter

import (
	"fmt"
	"strings"
)

// CompressStackTraces detects and compresses Go, Python, and Node.js
// stack traces found anywhere in the output. It preserves error messages,
// causes, and top application frames — those are the most important lines
// for debugging.
func CompressStackTraces(input string) string {
	result := compressGoStackTraces(input)
	result = compressPythonTracebacks(result)
	result = compressNodeStackTraces(result)
	return result
}

// --- Go stack traces ---

// compressGoStackTraces detects goroutine blocks and compresses duplicates.
// A goroutine block looks like:
//
//	goroutine N [status]:
//	func(args)
//		/path/file.go:line +0xoffset
//	...
func compressGoStackTraces(input string) string {
	lines := strings.Split(input, "\n")

	// Find goroutine block boundaries.
	type goroutineBlock struct {
		header string   // "goroutine N [status]:"
		body   []string // all lines after the header until the next blank/goroutine
		start  int      // index in lines
		end    int      // exclusive end index in lines
	}

	var blocks []goroutineBlock
	i := 0
	for i < len(lines) {
		if isGoroutineHeader(lines[i]) {
			b := goroutineBlock{
				header: lines[i],
				start:  i,
			}
			i++
			// Collect body lines: stack frames and their file locations.
			// Stop at a blank line or another goroutine header.
			for i < len(lines) && !isGoroutineHeader(lines[i]) && strings.TrimSpace(lines[i]) != "" {
				b.body = append(b.body, lines[i])
				i++
			}
			// Consume a single trailing blank line if present (separator between blocks).
			if i < len(lines) && strings.TrimSpace(lines[i]) == "" {
				b.body = append(b.body, lines[i])
				i++
			}
			b.end = i
			blocks = append(blocks, b)
		} else {
			i++
		}
	}

	if len(blocks) < 2 {
		return input
	}

	// Build stack signature (body without the header) for dedup.
	stackSig := func(b goroutineBlock) string {
		// Strip trailing blank lines for comparison.
		body := b.body
		for len(body) > 0 && strings.TrimSpace(body[len(body)-1]) == "" {
			body = body[:len(body)-1]
		}
		return strings.Join(body, "\n")
	}

	// Group consecutive identical stacks.
	type group struct {
		blocks []goroutineBlock
		sig    string
	}

	var groups []group
	currentSig := stackSig(blocks[0])
	currentGroup := group{blocks: []goroutineBlock{blocks[0]}, sig: currentSig}

	for _, b := range blocks[1:] {
		sig := stackSig(b)
		if sig == currentSig {
			currentGroup.blocks = append(currentGroup.blocks, b)
		} else {
			groups = append(groups, currentGroup)
			currentSig = sig
			currentGroup = group{blocks: []goroutineBlock{b}, sig: sig}
		}
	}
	groups = append(groups, currentGroup)

	// If no duplicates found, return original.
	hasDuplicates := false
	for _, g := range groups {
		if len(g.blocks) > 1 {
			hasDuplicates = true
			break
		}
	}
	if !hasDuplicates {
		return input
	}

	// Rebuild. We keep everything before the first block and after the last block,
	// then emit compressed blocks in between.
	var result []string

	// Lines before first goroutine block
	firstStart := blocks[0].start
	result = append(result, lines[:firstStart]...)

	for _, g := range groups {
		first := g.blocks[0]
		// Emit the first block in full.
		result = append(result, first.header)
		// Clean runtime-only frames from body if there are application frames.
		body := cleanGoRuntimeFrames(first.body)
		result = append(result, body...)

		if len(g.blocks) > 1 {
			extra := len(g.blocks) - 1
			noun := "goroutine"
			if extra > 1 {
				noun = "goroutines"
			}
			result = append(result, fmt.Sprintf("[... %d more %s with identical stack]", extra, noun))
			result = append(result, "")
		}
	}

	// Lines after last goroutine block
	lastEnd := blocks[len(blocks)-1].end
	if lastEnd < len(lines) {
		result = append(result, lines[lastEnd:]...)
	}

	return strings.Join(result, "\n")
}

// cleanGoRuntimeFrames removes runtime.goexit and runtime.main frames
// if there are other (application) frames present.
func cleanGoRuntimeFrames(body []string) []string {
	// Parse frames: a frame is a function line + a file line (indented with tab).
	type frame struct {
		funcLine string
		fileLine string
	}

	var frames []frame
	var nonFrameLines []string

	i := 0
	for i < len(body) {
		line := body[i]
		trimmed := strings.TrimSpace(line)

		// A frame starts with a function call (not indented with tab, not blank).
		if trimmed != "" && !strings.HasPrefix(line, "\t") {
			f := frame{funcLine: line}
			if i+1 < len(body) && strings.HasPrefix(body[i+1], "\t") {
				f.fileLine = body[i+1]
				i += 2
			} else {
				i++
			}
			frames = append(frames, f)
		} else {
			nonFrameLines = append(nonFrameLines, line)
			i++
		}
	}

	if len(frames) <= 1 {
		return body
	}

	// Check if we have any application frames.
	hasApp := false
	for _, f := range frames {
		if !isGoRuntimeFrame(f.funcLine) {
			hasApp = true
			break
		}
	}

	if !hasApp {
		return body
	}

	// Remove runtime.goexit and runtime.main.
	var filtered []frame
	for _, f := range frames {
		if isGoRuntimeFrame(f.funcLine) {
			continue
		}
		filtered = append(filtered, f)
	}

	var result []string
	for _, f := range filtered {
		result = append(result, f.funcLine)
		if f.fileLine != "" {
			result = append(result, f.fileLine)
		}
	}
	result = append(result, nonFrameLines...)

	return result
}

func isGoRuntimeFrame(funcLine string) bool {
	trimmed := strings.TrimSpace(funcLine)
	return strings.HasPrefix(trimmed, "runtime.goexit") ||
		strings.HasPrefix(trimmed, "runtime.main(")
}

func isGoroutineHeader(line string) bool {
	return strings.HasPrefix(line, "goroutine ") && strings.Contains(line, " [") && strings.HasSuffix(strings.TrimSpace(line), ":")
}

// --- Python tracebacks ---

// compressPythonTracebacks detects Python tracebacks and compresses middle
// frames when there are more than 5 frames. Preserves the exception line,
// the first frame, and the last frame.
func compressPythonTracebacks(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	i := 0

	for i < len(lines) {
		if strings.TrimSpace(lines[i]) == "Traceback (most recent call last):" {
			// Found a traceback. Collect all lines until the exception line.
			tbStart := i
			tbLines := []string{lines[i]}
			i++

			// Collect frames: each frame is "  File ..." + "    code..."
			type pyFrame struct {
				fileLine string
				codeLine string
			}
			var frames []pyFrame

			for i < len(lines) {
				trimmed := strings.TrimSpace(lines[i])
				if strings.HasPrefix(trimmed, "File ") {
					f := pyFrame{fileLine: lines[i]}
					i++
					if i < len(lines) && !strings.HasPrefix(strings.TrimSpace(lines[i]), "File ") &&
						strings.TrimSpace(lines[i]) != "Traceback (most recent call last):" &&
						strings.TrimSpace(lines[i]) != "" &&
						!isExceptionLine(lines[i]) {
						f.codeLine = lines[i]
						i++
					}
					frames = append(frames, f)
				} else {
					break
				}
			}

			// Collect exception line(s) — could be multi-line.
			var exceptionLines []string
			for i < len(lines) {
				trimmed := strings.TrimSpace(lines[i])
				if trimmed == "" {
					break
				}
				// Check for chained exception marker.
				if trimmed == "During handling of the above exception, another exception occurred:" ||
					trimmed == "The above exception was the direct cause of the following exception:" {
					break
				}
				exceptionLines = append(exceptionLines, lines[i])
				i++
			}

			// Now decide whether to compress.
			if len(frames) > 5 {
				result = append(result, tbLines[0]) // "Traceback (most recent call last):"
				// First frame
				first := frames[0]
				result = append(result, first.fileLine)
				if first.codeLine != "" {
					result = append(result, first.codeLine)
				}
				// Compressed middle
				middle := len(frames) - 2
				result = append(result, fmt.Sprintf("  [... %d more frames]", middle))
				// Last frame
				last := frames[len(frames)-1]
				result = append(result, last.fileLine)
				if last.codeLine != "" {
					result = append(result, last.codeLine)
				}
			} else {
				// Emit all frames as-is.
				result = append(result, tbLines[0])
				for _, f := range frames {
					result = append(result, f.fileLine)
					if f.codeLine != "" {
						result = append(result, f.codeLine)
					}
				}
			}

			// Exception line(s)
			result = append(result, exceptionLines...)
			_ = tbStart
		} else if isChainedExceptionMarker(lines[i]) {
			// Keep chained exception markers — they will be followed by another traceback.
			result = append(result, lines[i])
			i++
		} else {
			result = append(result, lines[i])
			i++
		}
	}

	return strings.Join(result, "\n")
}

func isExceptionLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	// Exception lines typically look like "ExceptionType: message" or just "ExceptionType".
	// They don't start with "File " and don't start with whitespace followed by code.
	if trimmed == "" {
		return false
	}
	if strings.HasPrefix(trimmed, "File ") {
		return false
	}
	// Check if it looks like ClassName or ClassName: message.
	if len(trimmed) > 0 && trimmed[0] >= 'A' && trimmed[0] <= 'Z' {
		return true
	}
	return false
}

func isChainedExceptionMarker(line string) bool {
	trimmed := strings.TrimSpace(line)
	return trimmed == "During handling of the above exception, another exception occurred:" ||
		trimmed == "The above exception was the direct cause of the following exception:"
}

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
		if isNodeErrorLine(lines[i]) && i+1 < len(lines) && isNodeStackFrame(lines[i+1]) {
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

func isNodeErrorLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if trimmed == "" {
		return false
	}
	// Common patterns: "Error: ...", "TypeError: ...", "RangeError: ...", etc.
	// Also "SomeCustomError: ..."
	colonIdx := strings.Index(trimmed, ": ")
	if colonIdx <= 0 {
		return false
	}
	prefix := trimmed[:colonIdx]
	// Must look like an error class name (starts with uppercase, ends with Error/Exception or is "Error").
	if len(prefix) == 0 {
		return false
	}
	if prefix[0] < 'A' || prefix[0] > 'Z' {
		return false
	}
	return strings.HasSuffix(prefix, "Error") || strings.HasSuffix(prefix, "Exception") || prefix == "Error"
}

func isNodeStackFrame(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "at ")
}

func isNodeInternalFrame(trimmedLine string) bool {
	return strings.Contains(trimmedLine, "(node:") ||
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
