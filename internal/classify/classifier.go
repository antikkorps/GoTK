package classify

import (
	"regexp"
	"strings"
)

// Level represents the semantic importance of a line of output.
type Level int

const (
	Noise    Level = iota // Pure noise: ANSI codes, decorative separators, empty lines
	Debug    Level = 1    // Debug info: timestamps, verbose logging, trace details
	Info     Level = 2    // Informational: normal output, status messages
	Warning  Level = 3    // Warnings: "warning", "WARN", deprecation notices
	Error    Level = 4    // Errors: "error", "ERROR", "FAIL", "panic", assertion failures
	Critical Level = 5    // Critical: stack traces, crash reports, fatal errors
)

// String returns a human-readable name for the level.
func (l Level) String() string {
	switch l {
	case Noise:
		return "Noise"
	case Debug:
		return "Debug"
	case Info:
		return "Info"
	case Warning:
		return "Warning"
	case Error:
		return "Error"
	case Critical:
		return "Critical"
	default:
		return "Unknown"
	}
}

// Compiled patterns for classification.
var (
	// Critical patterns
	panicPrefix       = regexp.MustCompile(`(?i)^panic:`)
	fatalPrefix       = regexp.MustCompile(`(?i)^fatal:`)
	fatalUpper        = regexp.MustCompile(`^FATAL`)
	goStackHeader     = regexp.MustCompile(`goroutine \d+ \[`)
	stackFrame        = regexp.MustCompile(`^\s+\S+\.go:\d+`)
	stackFrameGeneric = regexp.MustCompile(`^\s+/.+:\d+`)
	pythonTraceback   = regexp.MustCompile(`Traceback \(most recent call last\)`)
	pythonFileLine    = regexp.MustCompile(`^\s+File ".+", line \d+`)
	nodeStack         = regexp.MustCompile(`^\s+at (Object|Module|Function|internal)\.`)
	segfault          = regexp.MustCompile(`(?i)(segmentation fault|SIGSEGV|segfault)`)

	// Error patterns
	errorWord       = regexp.MustCompile(`(?i)\berror\b`)
	pythonException = regexp.MustCompile(`^[A-Z]\w*(Error|Exception|Warning|Fault):`)
	zeroErrors      = regexp.MustCompile(`(?i)\b0 errors?\b`)
	failWord        = regexp.MustCompile(`(?i)\b(FAIL|failed|failure)\b`)
	assertionFailed = regexp.MustCompile(`(?i)assertion failed`)
	expectGot       = regexp.MustCompile(`(?i)(expected .+ got|want .+ got)`)
	exitNonZero     = regexp.MustCompile(`(?i)exit (code|status) [1-9]`)
	compileError    = regexp.MustCompile(`(?i)\b(cannot|undefined|not found|syntax error)\b`)

	// Warning patterns
	warningWord    = regexp.MustCompile(`(?i)\b(warning|warn)\b`)
	deprecatedWord = regexp.MustCompile(`(?i)\bdeprecated\b`)
	todoFixme      = regexp.MustCompile(`\b(TODO|FIXME|HACK)\b`)
	skippedWord    = regexp.MustCompile(`(?i)\b(skipped|SKIP)\b`)

	// Debug patterns
	timestampOnly = regexp.MustCompile(`^\d{4}[-/]\d{2}[-/]\d{2}[T ]\d{2}:\d{2}:\d{2}`)
	verbosePrefix = regexp.MustCompile(`(?i)^\[?(debug|trace|verbose)\]?:?\s`)
	progressInd   = regexp.MustCompile(`^\s*\d+%\s*$|^[\s|/\\-]+$`)

	// Noise patterns
	pureANSI       = regexp.MustCompile(`^\x1b\[[0-9;]*[a-zA-Z]$`)
	ansiContent    = regexp.MustCompile(`\x1b\[`)
	decorativeLine = regexp.MustCompile(`^[-=_~]{10,}$`)
)

// Classify returns the semantic importance level of a line.
func Classify(line string) Level {
	trimmed := strings.TrimSpace(line)

	// Noise: empty lines, pure ANSI, decorative separators
	if trimmed == "" {
		return Noise
	}
	if pureANSI.MatchString(trimmed) {
		return Noise
	}
	if decorativeLine.MatchString(trimmed) {
		return Noise
	}
	// Line that is only ANSI codes with no visible text
	if ansiContent.MatchString(trimmed) {
		stripped := regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`).ReplaceAllString(trimmed, "")
		if strings.TrimSpace(stripped) == "" {
			return Noise
		}
	}

	// Critical: always checked first (highest priority)
	if panicPrefix.MatchString(trimmed) {
		return Critical
	}
	if fatalPrefix.MatchString(trimmed) || fatalUpper.MatchString(trimmed) {
		return Critical
	}
	if goStackHeader.MatchString(trimmed) {
		return Critical
	}
	if stackFrame.MatchString(line) || stackFrameGeneric.MatchString(line) {
		return Critical
	}
	if pythonTraceback.MatchString(trimmed) {
		return Critical
	}
	if pythonFileLine.MatchString(line) {
		return Critical
	}
	if nodeStack.MatchString(line) {
		return Critical
	}
	if segfault.MatchString(trimmed) {
		return Critical
	}

	// Error
	if errorWord.MatchString(trimmed) && !zeroErrors.MatchString(trimmed) {
		return Error
	}
	if pythonException.MatchString(trimmed) {
		return Error
	}
	if failWord.MatchString(trimmed) {
		return Error
	}
	if assertionFailed.MatchString(trimmed) {
		return Error
	}
	if expectGot.MatchString(trimmed) {
		return Error
	}
	if exitNonZero.MatchString(trimmed) {
		return Error
	}
	if compileError.MatchString(trimmed) {
		return Error
	}

	// Warning
	if warningWord.MatchString(trimmed) {
		return Warning
	}
	if deprecatedWord.MatchString(trimmed) {
		return Warning
	}
	if todoFixme.MatchString(trimmed) {
		return Warning
	}
	if skippedWord.MatchString(trimmed) {
		return Warning
	}

	// Debug
	if verbosePrefix.MatchString(trimmed) {
		return Debug
	}
	if progressInd.MatchString(trimmed) {
		return Debug
	}
	// Timestamp-only lines (no other meaningful content after the timestamp)
	if timestampOnly.MatchString(trimmed) {
		afterTS := timestampOnly.ReplaceAllString(trimmed, "")
		afterTS = strings.TrimSpace(afterTS)
		if afterTS == "" || len(afterTS) < 3 {
			return Debug
		}
	}

	// Default: Info
	return Info
}

// ClassifyLines splits the input into lines, classifies each one,
// and returns the lines along with their corresponding levels.
func ClassifyLines(input string) ([]string, []Level) {
	lines := strings.Split(input, "\n")
	levels := make([]Level, len(lines))

	for i, line := range lines {
		levels[i] = Classify(line)
	}

	return lines, levels
}
