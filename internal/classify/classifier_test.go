package classify

import (
	"strings"
	"testing"
)

func TestClassify(t *testing.T) {
	tests := []struct {
		name  string
		line  string
		want  Level
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
