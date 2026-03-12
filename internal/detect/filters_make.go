package detect

import (
	"regexp"
	"strings"
)

var (
	makeEnteringPattern   = regexp.MustCompile(`^make\[\d+\]: Entering directory`)
	makeLeavingPattern    = regexp.MustCompile(`^make\[\d+\]: Leaving directory`)
	makeNothingPattern    = regexp.MustCompile(`^make\[\d+\]: Nothing to be done for`)
	makeErrorPattern      = regexp.MustCompile(`^make(\[\d+\])?: \*\*\*`)
	// Matches gcc/g++/cc compilation command lines with flags
	compilerCmdPattern    = regexp.MustCompile(`^\s*(gcc|g\+\+|cc|c\+\+|clang|clang\+\+)\s+`)
	// Extract -o output or source file from compiler command
	compilerSourcePattern = regexp.MustCompile(`\s(\S+\.(c|cc|cpp|cxx|m|mm|s|S))\b`)
	compilerOutputPattern = regexp.MustCompile(`-o\s+(\S+)`)
)

// compressMakeOutput removes make enter/leave directory noise and compresses
// verbose compiler command lines while preserving errors.
func compressMakeOutput(input string) string {
	lines := strings.Split(input, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			continue
		}

		// Remove entering/leaving directory lines
		if makeEnteringPattern.MatchString(trimmed) {
			continue
		}
		if makeLeavingPattern.MatchString(trimmed) {
			continue
		}

		// Remove "nothing to be done" lines
		if makeNothingPattern.MatchString(trimmed) {
			continue
		}

		// Always keep make error lines
		if makeErrorPattern.MatchString(trimmed) {
			result = append(result, line)
			continue
		}

		// Compress verbose compiler command lines into short form
		if compilerCmdPattern.MatchString(trimmed) {
			short := compressCompilerCmd(trimmed)
			result = append(result, short)
			continue
		}

		// Keep everything else (errors, linker output, etc.)
		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// compressCompilerCmd shortens a long gcc/g++ command to just show compiler + source file.
func compressCompilerCmd(cmd string) string {
	// Find the compiler name
	compMatch := compilerCmdPattern.FindStringSubmatch(cmd)
	compiler := "cc"
	if len(compMatch) >= 2 {
		compiler = compMatch[1]
	}

	// Try to find the source file
	srcMatch := compilerSourcePattern.FindStringSubmatch(cmd)
	if len(srcMatch) >= 2 {
		src := srcMatch[1]
		// Also check for -o target
		outMatch := compilerOutputPattern.FindStringSubmatch(cmd)
		if len(outMatch) >= 2 {
			return compiler + " " + src + " -o " + outMatch[1]
		}
		return compiler + " " + src
	}

	// Try -o as fallback
	outMatch := compilerOutputPattern.FindStringSubmatch(cmd)
	if len(outMatch) >= 2 {
		return compiler + " ... -o " + outMatch[1]
	}

	// Can't parse, return original but trimmed
	return strings.TrimSpace(cmd)
}
