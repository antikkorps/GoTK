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

	// Compiler diagnostic patterns (Rust note/help, important for LLM context)
	rustNoteHelp = regexp.MustCompile(`^\s*=\s*(note|help):`)
	// JSON/YAML parse error patterns
	jsonParseError = regexp.MustCompile(`(?i)(json|yaml|toml)\s*(parse|syntax|decode|unmarshal)\s*(error|failed|invalid)`)
	jsonUnexpected = regexp.MustCompile(`(?i)unexpected (token|end|character).*json`)
	yamlScanError  = regexp.MustCompile(`(?i)yaml:\s*(line \d+|did not find|could not find|mapping values)`)

	// Debug patterns
	timestampOnly = regexp.MustCompile(`^\d{4}[-/]\d{2}[-/]\d{2}[T ]\d{2}:\d{2}:\d{2}`)
	verbosePrefix = regexp.MustCompile(`(?i)^\[?(debug|trace|verbose)\]?:?\s`)
	progressInd   = regexp.MustCompile(`^\s*\d+%\s*$|^[\s|/\\-]+$`)

	// Noise patterns
	pureANSI       = regexp.MustCompile(`^\x1b\[[0-9;]*[a-zA-Z]$`)
	ansiContent    = regexp.MustCompile(`\x1b\[`)
	decorativeLine = regexp.MustCompile(`^[-=_~]{10,}$`)

	// ansiStripPattern is used to strip ANSI escape sequences for content check.
	ansiStripPattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

	// sourceGrepPrefix matches `path.<ext>:LINE[:COL]:` where the path ends
	// in a recognised source-code extension. Used to detect grep matches
	// over source code, where bare-word tokens like "error" or "TODO" are
	// usually identifiers / comments, not diagnostics.
	sourceGrepPrefix = regexp.MustCompile(`^[^\s:]+\.(?:go|rs|py|js|jsx|ts|tsx|mjs|cjs|java|kt|scala|c|cc|cpp|cxx|h|hpp|hxx|m|mm|rb|php|swift|ex|exs|erl|hs|ml|clj|cljs|cljc|dart|lua|sh|bash|zsh|fish|ps1|md|markdown|astro|svelte|vue|css|scss|sass|less|html|xml|yaml|yml|toml|json|jsonc|sql|proto|graphql|gql):\d+(?::\d+)?:\s*`)

	// strictErrorPrefix only matches structured diagnostic forms inside
	// source-grep content (e.g. "error: undefined" from a compiler), not
	// bare-word occurrences like `func DoX(err error) error`.
	strictErrorPrefix   = regexp.MustCompile(`^(?:error|fatal):\s`)
	strictWarningPrefix = regexp.MustCompile(`^(?:warning|warn):\s`)

	// strictCompilerDiag matches compiler-diagnostic messages that conventionally
	// appear as the first token of the content after `path:line:col:` — e.g.
	// `cannot use x (type int)`, `undefined: doStuff`, `expected ';' before '}'`.
	// Requires the verb at line start so it doesn't fire on identifiers like
	// `cannotDoX` or assignments like `undefined := nil`.
	strictCompilerDiag = regexp.MustCompile(`^(?:cannot |undefined: |undefined reference|not found|syntax error|expected |missing )`)
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
		stripped := ansiStripPattern.ReplaceAllString(trimmed, "")
		if strings.TrimSpace(stripped) == "" {
			return Noise
		}
	}

	// Source-code grep matches (e.g. `internal/foo.go:42:func DoX(err error) error {`):
	// classify the content with stricter rules, so identifiers like `error`,
	// `Warning`, or comment markers like `TODO:` don't get promoted to
	// Error/Warning just because they appear as substrings of source code.
	if m := sourceGrepPrefix.FindString(trimmed); m != "" {
		return classifySourceMatch(trimmed[len(m):])
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

	// Compiler diagnostics (Rust note/help) — classify as Warning so they're never removed
	if rustNoteHelp.MatchString(trimmed) {
		return Warning
	}
	// JSON/YAML/TOML parse errors
	if jsonParseError.MatchString(trimmed) || jsonUnexpected.MatchString(trimmed) || yamlScanError.MatchString(trimmed) {
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

// classifySourceMatch is the stricter classifier used for content extracted
// from `path.<ext>:LINE[:COL]:` grep matches over source code.
//
// In source files, bare-word tokens like `error`, `warning`, `TODO`, `FAIL`
// almost always appear as identifiers, type names, comments, or test fixtures
// — not as diagnostics. So we only elevate above Info when the content carries
// a structural signal: a stack-frame shape, a compiler-style `error:` prefix,
// or a panic/fatal marker.
func classifySourceMatch(content string) Level {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return Noise
	}

	// Critical structural signals survive: a grep'd line that itself is a
	// stack frame, a panic header, etc., is genuinely important.
	if panicPrefix.MatchString(trimmed) {
		return Critical
	}
	if fatalPrefix.MatchString(trimmed) || fatalUpper.MatchString(trimmed) {
		return Critical
	}
	if goStackHeader.MatchString(trimmed) {
		return Critical
	}
	if pythonTraceback.MatchString(trimmed) {
		return Critical
	}
	if segfault.MatchString(trimmed) {
		return Critical
	}

	// Compiler-diagnostic form: `error: <msg>` / `fatal: <msg>` at start.
	// Matches gcc/clang/rustc/go output even when shown via a grep that left
	// the path:line: prefix intact (we've already stripped it).
	if strictErrorPrefix.MatchString(trimmed) {
		return Error
	}
	if strictWarningPrefix.MatchString(trimmed) {
		return Warning
	}

	// Go/Rust/C compiler diagnostics often start with a verb like `cannot use`
	// or `undefined: foo` rather than an `error:` keyword. Anchored at start
	// to avoid firing on source identifiers (`func cannotDoX`).
	if strictCompilerDiag.MatchString(trimmed) {
		return Error
	}

	// JSON/YAML parse errors are explicit enough to keep.
	if jsonParseError.MatchString(trimmed) || jsonUnexpected.MatchString(trimmed) || yamlScanError.MatchString(trimmed) {
		return Error
	}

	// Bare-word patterns (\berror\b, \bwarning\b, TODO, FIXME, ...) are
	// intentionally NOT consulted here: in source code they overwhelmingly
	// reflect identifiers or comments, not diagnostics. Demoting them to
	// Info prevents the grep/per-file collapse from tanking the quality
	// score on legitimate source-code searches.
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
