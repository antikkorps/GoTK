package classify

import (
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name string
		line string
		want Level
	}{
		// Noise
		{"empty line", "", Noise},
		{"pure ANSI", "\x1b[31m", Noise},
		{"decorative separator", "====================", Noise},
		{"decorative dashes", "--------------------", Noise},
		{"ANSI only content", "\x1b[0m\x1b[32m\x1b[0m", Noise},

		// Critical - Go panics and stack traces
		{"go panic", "panic: runtime error: index out of range", Critical},
		{"fatal prefix", "fatal: not a git repository", Critical},
		{"FATAL upper", "FATAL: cannot connect to database", Critical},
		{"goroutine header", "goroutine 1 [running]:", Critical},
		{"go stack frame", "\tmain.go:42 +0x1a3", Critical},
		{"go stack frame path", "\t/home/user/project/main.go:42", Critical},

		// Critical - Python tracebacks
		{"python traceback", "Traceback (most recent call last)", Critical},

		// Critical - Node.js stack traces
		{"node stack at Object", "    at Object.<anonymous> (/app/index.js:1:1)", Critical},
		{"node stack at Module", "    at Module._compile (internal/modules/cjs/loader.js:999:30)", Critical},

		// Critical - segfault
		{"segfault", "Segmentation fault (core dumped)", Critical},
		{"sigsegv", "signal: SIGSEGV", Critical},

		// Error - general error patterns
		{"error word", "Error: file not found", Error},
		{"ERROR upper", "ERROR something went wrong", Error},
		{"lowercase error", "connection error occurred", Error},
		{"zero errors not error", "0 errors found", Info},
		{"FAIL keyword", "FAIL github.com/user/pkg 0.5s", Error},
		{"failed keyword", "test failed: expected true", Error},
		{"failure keyword", "build failure in module", Error},
		{"assertion failed", "assertion failed: x != y", Error},

		// Error - expect/got patterns
		{"expected got", "expected 42 got 43", Error},
		{"want got", "want true got false", Error},

		// Error - exit code
		{"exit code non-zero", "exit code 1", Error},
		{"exit status 2", "process exited with exit status 2", Error},

		// Error - compilation errors
		{"cannot compile", "cannot use x (type int) as string", Error},
		{"undefined var", "undefined: myFunction", Error},
		{"not found", "package not found: foo/bar", Error},
		{"syntax error", "syntax error: unexpected token", Error},

		// Warning
		{"warning word", "warning: unused variable 'x'", Warning},
		{"Warning capitalized", "Warning: this is deprecated", Warning},
		{"WARN upper", "WARN: low disk space", Warning},
		{"deprecated", "This function is deprecated", Warning},
		{"TODO comment", "TODO: fix this later", Warning},
		{"FIXME comment", "FIXME: memory leak here", Warning},
		{"HACK comment", "HACK: workaround for issue #123", Warning},
		{"skipped test", "--- SKIP: TestFoo (0.00s)", Warning},

		// Rust compiler diagnostics (note/help lines)
		{"rust note", "   = note: expected type `String`", Warning},
		{"rust help", "   = help: consider using `.to_string()`", Warning},

		// JSON/YAML/TOML parse errors
		{"json parse error", "json parse error: unexpected token", Error},
		{"yaml scan error", "yaml: line 5: did not find expected key", Error},
		{"json unmarshal", "json unmarshal failed at offset 42", Error},
		{"toml decode error", "toml decode error: invalid key", Error},
		{"unexpected token json", "unexpected token } in json at position 10", Error},

		// Info - normal output
		{"normal output", "Hello, World!", Info},
		{"file listing", "drwxr-xr-x  2 user group 4096 Jan 1 main.go", Info},
		{"test name", "--- PASS: TestFoo (0.00s)", Info},
		{"status message", "Building project...", Info},
		{"go test ok", "ok  github.com/user/pkg 0.3s", Info},

		// Debug
		{"verbose prefix", "[debug] loading config", Debug},
		{"trace prefix", "[trace] entering function", Debug},
		{"verbose colon", "verbose: extra info", Debug},
		{"progress percent", "  75%  ", Debug},
		{"timestamp only", "2024-01-15T10:30:00", Debug},
		{"timestamp with date slash", "2024/01/15 10:30:00", Debug},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.line)
			if got != tt.want {
				t.Errorf("Classify(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

// TestClassify_SourceGrepMatches covers issue #58: bare-word tokens like
// `error`, `warning`, `TODO` inside source-grep matches should NOT be
// promoted to Error/Warning, because they're identifiers / comments, not
// diagnostics. Structural signals (panic:, stack frames, compiler diag
// at start of content, `Error:` prefix) still elevate.
func TestClassify_SourceGrepMatches(t *testing.T) {
	tests := []struct {
		name string
		line string
		want Level
	}{
		// --- The repro from #58 ---
		{
			name: "go error type in func signature (bug repro)",
			line: "internal/foo.go:42:func DoX(err error) error {",
			want: Info,
		},
		{
			name: "go errors package import",
			line: "internal/foo.go:5:\timport \"errors\"",
			want: Info,
		},
		{
			name: "go test function name containing Error",
			line: "internal/errors_test.go:8:func TestErrorHandler(t *testing.T) {",
			want: Info,
		},
		{
			name: "TODO comment in source",
			line: "src/server/handler.go:120:\t// TODO: rate limit this endpoint",
			want: Info,
		},
		{
			name: "warning as variable name",
			line: "src/util/log.go:33:\twarning := \"deprecated\"",
			want: Info,
		},
		{
			name: "fail as function name",
			line: "test/runner.go:88:\treturn fail(\"expected match\")",
			want: Info,
		},

		// --- Structural signals must still elevate even inside source matches ---
		{
			name: "go compiler error: cannot use",
			line: "./main.go:10:5: cannot use x (type int) as type string",
			want: Error,
		},
		{
			name: "go compiler error: undefined",
			line: "./main.go:15:2: undefined: doStuff",
			want: Error,
		},
		{
			name: "rust compiler error keyword",
			line: "src/lib.rs:5:9: error: cannot find value `foo`",
			want: Error,
		},
		{
			name: "panic header inside source match",
			line: "main.go:1:panic: nil pointer dereference",
			want: Critical,
		},

		// --- Non-source-code prefixes are unaffected (regression guard) ---
		{
			name: "log file grep match with ERROR keyword",
			line: "app.log:42:2026-05-11T21:00:00 ERROR connection refused",
			want: Error,
		},
		{
			name: "config file grep match without diagnostic",
			line: "config.ini:7:debug=true",
			want: Info,
		},

		// --- No path prefix at all: existing behavior ---
		{
			name: "bare-word error without grep prefix still classifies",
			line: "An error occurred while loading config",
			want: Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Classify(tt.line)
			if got != tt.want {
				t.Errorf("Classify(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}

func TestClassifyLines(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		wantLevels []Level
	}{
		{
			name: "go compilation error",
			input: strings.Join([]string{
				"# github.com/user/project",
				"./main.go:10:5: cannot use x (type int) as type string",
				"./main.go:15:2: undefined: doStuff",
			}, "\n"),
			wantLevels: []Level{Info, Error, Error},
		},
		{
			name: "go test failure with panic",
			input: strings.Join([]string{
				"=== RUN   TestSomething",
				"panic: runtime error: index out of range [3] with length 2",
				"",
				"goroutine 6 [running]:",
				"main.doWork()",
				"\t/home/user/main.go:42 +0x1a3",
				"FAIL\tgithub.com/user/pkg\t0.005s",
			}, "\n"),
			wantLevels: []Level{Info, Critical, Noise, Critical, Info, Critical, Error},
		},
		{
			name: "python traceback",
			input: strings.Join([]string{
				"Traceback (most recent call last):",
				"  File \"/app/main.py\", line 10, in <module>",
				"    raise ValueError(\"bad value\")",
				"ValueError: bad value",
			}, "\n"),
			wantLevels: []Level{Critical, Critical, Info, Error},
		},
		{
			name: "node.js stack trace",
			input: strings.Join([]string{
				"TypeError: Cannot read property 'foo' of undefined",
				"    at Object.<anonymous> (/app/index.js:5:15)",
				"    at Module._compile (internal/modules/cjs/loader.js:999:30)",
			}, "\n"),
			wantLevels: []Level{Error, Critical, Critical},
		},
		{
			name: "mixed output with warnings and info",
			input: strings.Join([]string{
				"Building project...",
				"warning: unused import 'fmt'",
				"Compiling main.go",
				"TODO: optimize this function",
				"Build complete.",
			}, "\n"),
			wantLevels: []Level{Info, Warning, Info, Warning, Info},
		},
		{
			name: "all levels present",
			input: strings.Join([]string{
				"",
				"[debug] starting up",
				"Server listening on :8080",
				"warning: deprecated API used",
				"Error: connection refused",
				"panic: out of memory",
			}, "\n"),
			wantLevels: []Level{Noise, Debug, Info, Warning, Error, Critical},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			lines, levels := ClassifyLines(tt.input)
			if len(lines) != len(tt.wantLevels) {
				t.Fatalf("ClassifyLines returned %d lines, want %d", len(lines), len(tt.wantLevels))
			}
			for i, wantLevel := range tt.wantLevels {
				if levels[i] != wantLevel {
					t.Errorf("line %d %q: got level %v, want %v", i, lines[i], levels[i], wantLevel)
				}
			}
		})
	}
}

func TestLevelString(t *testing.T) {
	tests := []struct {
		level Level
		want  string
	}{
		{Noise, "Noise"},
		{Debug, "Debug"},
		{Info, "Info"},
		{Warning, "Warning"},
		{Error, "Error"},
		{Critical, "Critical"},
		{Level(99), "Unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.level.String(); got != tt.want {
				t.Errorf("Level(%d).String() = %q, want %q", tt.level, got, tt.want)
			}
		})
	}
}

// TestClassify_DiagnosticWordsInsidePaths covers issue #71: build tools list
// artifact filenames that contain diagnostic words, and a green build must not
// be reported as failing because of them.
func TestClassify_DiagnosticWordsInsidePaths(t *testing.T) {
	tests := []struct {
		name string
		line string
		want Level
	}{
		// --- The repro from #71 (Nuxt / Nitro build listing) ---
		{
			name: "nuxt error-500 css asset",
			line: "ℹ node_modules/.cache/nuxt/.nuxt/dist/client/_nuxt/error-500.ChVycSbP.css    1.91 kB │ gzip:  0.73 kB",
			want: Info,
		},
		{
			name: "nuxt error-404 css asset",
			line: "ℹ node_modules/.cache/nuxt/.nuxt/dist/client/_nuxt/error-404.AlaAyKR2.css    2.43 kB │ gzip:  0.86 kB",
			want: Info,
		},
		{
			name: "nitro server chunk listing",
			line: "├─ .output/server/chunks/_/error-500.mjs (5.09 kB) (2.06 kB gzip)",
			want: Info,
		},
		{
			name: "bare artifact filename",
			line: "  error-404.mjs",
			want: Info,
		},
		{
			name: "directory segment named error",
			line: "  dist/error/index.html  1.2 kB",
			want: Info,
		},
		{
			name: "filename containing failed",
			line: "  build/test-failed.snapshot.js  0.4 kB",
			want: Info,
		},
		{
			name: "filename containing deprecated",
			line: "  dist/deprecated-api.mjs  1.1 kB",
			want: Info,
		},

		// --- Real diagnostics must still be classified, paths and all ---
		{
			name: "error keyword alongside a path",
			line: "ERROR: build failed at src/main.ts",
			want: Error,
		},
		{
			name: "module resolution failure naming an error file",
			line: "Cannot find module './error-handler.js'",
			want: Error,
		},
		{
			name: "node ENOENT mentioning a path",
			line: "Error: ENOENT: no such file or directory, open '/tmp/error.txt'",
			want: Error,
		},
		{
			name: "go test failure with subtest path",
			line: "--- FAIL: TestFoo/subtest (0.00s)",
			want: Error,
		},
		{
			name: "warning keyword alongside a path",
			line: "warning: unused import in src/util/log.ts",
			want: Warning,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Classify(tt.line); got != tt.want {
				t.Errorf("Classify(%q) = %v, want %v", tt.line, got, tt.want)
			}
		})
	}
}
