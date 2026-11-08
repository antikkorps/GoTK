package filter

import (
	"strings"
	"testing"
)

func TestCompressGoStackTraces(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "single goroutine unchanged",
			input: `goroutine 1 [running]:
main.foo(0x1234)
	/app/main.go:42 +0x1a2
main.bar(0x5678)
	/app/main.go:30 +0x8f
main.main()
	/app/main.go:10 +0x25`,
			want: `goroutine 1 [running]:
main.foo(0x1234)
	/app/main.go:42 +0x1a2
main.bar(0x5678)
	/app/main.go:30 +0x8f
main.main()
	/app/main.go:10 +0x25`,
		},
		{
			name: "two identical goroutines compressed",
			input: `goroutine 1 [running]:
main.foo(0x1234)
	/app/main.go:42 +0x1a2
main.bar(0x5678)
	/app/main.go:30 +0x8f
main.main()
	/app/main.go:10 +0x25

goroutine 2 [running]:
main.foo(0x1234)
	/app/main.go:42 +0x1a2
main.bar(0x5678)
	/app/main.go:30 +0x8f
main.main()
	/app/main.go:10 +0x25`,
			want: `goroutine 1 [running]:
main.foo(0x1234)
	/app/main.go:42 +0x1a2
main.bar(0x5678)
	/app/main.go:30 +0x8f
main.main()
	/app/main.go:10 +0x25

[... 1 more goroutine with identical stack]
`,
		},
		{
			name: "three identical goroutines compressed with plural",
			input: `goroutine 1 [running]:
main.handler(0x1)
	/app/main.go:42 +0x1a2

goroutine 2 [running]:
main.handler(0x1)
	/app/main.go:42 +0x1a2

goroutine 3 [running]:
main.handler(0x1)
	/app/main.go:42 +0x1a2`,
			want: `goroutine 1 [running]:
main.handler(0x1)
	/app/main.go:42 +0x1a2

[... 2 more goroutines with identical stack]
`,
		},
		{
			name: "different goroutines not compressed",
			input: `goroutine 1 [running]:
main.foo()
	/app/main.go:42 +0x1a2

goroutine 2 [running]:
main.bar()
	/app/main.go:30 +0x8f`,
			want: `goroutine 1 [running]:
main.foo()
	/app/main.go:42 +0x1a2

goroutine 2 [running]:
main.bar()
	/app/main.go:30 +0x8f`,
		},
		{
			name: "panic with message before goroutines",
			input: `panic: runtime error: index out of range [3] with length 2

goroutine 1 [running]:
main.foo(0x1234)
	/app/main.go:42 +0x1a2

goroutine 2 [running]:
main.foo(0x1234)
	/app/main.go:42 +0x1a2`,
			want: `panic: runtime error: index out of range [3] with length 2

goroutine 1 [running]:
main.foo(0x1234)
	/app/main.go:42 +0x1a2

[... 1 more goroutine with identical stack]
`,
		},
		{
			name: "runtime frames removed when app frames exist",
			input: `goroutine 1 [running]:
main.foo(0x1234)
	/app/main.go:42 +0x1a2
runtime.goexit()
	/usr/local/go/src/runtime/asm_amd64.s:1695 +0x1

goroutine 2 [running]:
main.foo(0x1234)
	/app/main.go:42 +0x1a2
runtime.goexit()
	/usr/local/go/src/runtime/asm_amd64.s:1695 +0x1`,
			want: `goroutine 1 [running]:
main.foo(0x1234)
	/app/main.go:42 +0x1a2

[... 1 more goroutine with identical stack]
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compressGoStackTraces(tt.input)
			if got != tt.want {
				t.Errorf("compressGoStackTraces():\n--- got ---\n%s\n--- want ---\n%s", got, tt.want)
			}
		})
	}
}

func TestCompressPythonTracebacks(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "short traceback kept as-is",
			input: `Traceback (most recent call last):
  File "/app/main.py", line 10, in main
    result = process()
  File "/app/process.py", line 25, in process
    return transform(data)
ValueError: bad input`,
			want: `Traceback (most recent call last):
  File "/app/main.py", line 10, in main
    result = process()
  File "/app/process.py", line 25, in process
    return transform(data)
ValueError: bad input`,
		},
		{
			name: "long traceback compressed",
			input: `Traceback (most recent call last):
  File "/app/main.py", line 10, in main
    result = process()
  File "/app/process.py", line 25, in process
    return transform(data)
  File "/app/transform.py", line 42, in transform
    return parse(raw)
  File "/app/parse.py", line 15, in parse
    return validate(x)
  File "/app/validate.py", line 8, in validate
    return check(y)
  File "/app/check.py", line 3, in check
    raise ValueError("bad input")
ValueError: bad input`,
			want: `Traceback (most recent call last):
  File "/app/main.py", line 10, in main
    result = process()
  [... 4 more frames]
  File "/app/check.py", line 3, in check
    raise ValueError("bad input")
ValueError: bad input`,
		},
		{
			name: "exactly 5 frames kept as-is",
			input: `Traceback (most recent call last):
  File "/app/a.py", line 1, in a
    b()
  File "/app/b.py", line 2, in b
    c()
  File "/app/c.py", line 3, in c
    d()
  File "/app/d.py", line 4, in d
    e()
  File "/app/e.py", line 5, in e
    raise RuntimeError("x")
RuntimeError: x`,
			want: `Traceback (most recent call last):
  File "/app/a.py", line 1, in a
    b()
  File "/app/b.py", line 2, in b
    c()
  File "/app/c.py", line 3, in c
    d()
  File "/app/d.py", line 4, in d
    e()
  File "/app/e.py", line 5, in e
    raise RuntimeError("x")
RuntimeError: x`,
		},
		{
			name: "chained exceptions both present",
			input: `Traceback (most recent call last):
  File "/app/main.py", line 10, in main
    result = process()
ValueError: bad input

During handling of the above exception, another exception occurred:

Traceback (most recent call last):
  File "/app/handler.py", line 5, in handle
    recover()
RuntimeError: recovery failed`,
			want: `Traceback (most recent call last):
  File "/app/main.py", line 10, in main
    result = process()
ValueError: bad input

During handling of the above exception, another exception occurred:

Traceback (most recent call last):
  File "/app/handler.py", line 5, in handle
    recover()
RuntimeError: recovery failed`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compressPythonTracebacks(tt.input)
			if got != tt.want {
				t.Errorf("compressPythonTracebacks():\n--- got ---\n%s\n--- want ---\n%s", got, tt.want)
			}
		})
	}
}

func TestCompressNodeStackTraces(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name: "node stack with mixed frames",
			input: `Error: Connection refused
    at TCPConnectWrap.afterConnect [as oncomplete] (node:net:1595:16)
    at Socket.connect (node:net:1039:16)
    at Object.connect (/app/db.js:42:10)
    at processRequest (/app/server.js:15:8)
    at Layer.handle (/app/node_modules/express/lib/router/layer.js:95:5)
    at next (/app/node_modules/express/lib/router/route.js:144:13)`,
			want: `Error: Connection refused
    at Object.connect (/app/db.js:42:10)
    at processRequest (/app/server.js:15:8)
    [... 2 node_modules frames, 2 node internals]`,
		},
		{
			name: "only app frames preserved",
			input: `TypeError: Cannot read properties of undefined
    at render (/app/views/index.js:10:5)
    at main (/app/app.js:5:3)`,
			want: `TypeError: Cannot read properties of undefined
    at render (/app/views/index.js:10:5)
    at main (/app/app.js:5:3)`,
		},
		{
			name: "only node internals",
			input: `Error: ENOENT
    at FSReqCallback.oncomplete (node:fs:198:21)
    at FSReqCallback.callbackTrampoline (node:internal/async_hooks:130:17)`,
			want: `Error: ENOENT
    [... 2 node internals]`,
		},
		{
			name: "many app frames keeps first 2",
			input: `RangeError: Maximum call stack size exceeded
    at a (/app/a.js:1:1)
    at b (/app/b.js:2:2)
    at c (/app/c.js:3:3)
    at d (/app/d.js:4:4)`,
			want: `RangeError: Maximum call stack size exceeded
    at a (/app/a.js:1:1)
    at b (/app/b.js:2:2)
    [... 2 more app frames]`,
		},
		{
			// console.error('msg:', err) prints the error after a caller
			// prefix; Jest then dumps its own runner frames (issue #95).
			name: "error embedded after a log prefix",
			input: `    Failed to send mail: Error: SMTP down
        at send (/app/src/mail.js:1:31)
        at Promise.then.completed (/app/node_modules/jest-circus/build/utils.js:298:28)
        at callAsyncCircusFn (/app/node_modules/jest-circus/build/utils.js:231:10)`,
			want: `    Failed to send mail: Error: SMTP down
        at send (/app/src/mail.js:1:31)
    [... 2 node_modules frames]`,
		},
		{
			name: "anonymous builtin frames count as internals",
			input: `TypeError: x is not a function
    at main (/app/app.js:5:3)
    at new Promise (<anonymous>)
    at Array.forEach (<anonymous>)`,
			want: `TypeError: x is not a function
    at main (/app/app.js:5:3)
    [... 2 node internals]`,
		},
		{
			// Without an error class the frames may be anything (a Jest
			// console trailer, a hand-written trace): leave them alone.
			name: "prefix without error class is untouched",
			input: `Request done: status 200
    at a (/app/node_modules/x/a.js:1:1)
    at b (/app/node_modules/x/b.js:2:2)`,
			want: `Request done: status 200
    at a (/app/node_modules/x/a.js:1:1)
    at b (/app/node_modules/x/b.js:2:2)`,
		},
		{
			// JVM frames carry one line number, not line:col; an embedded
			// exception class must not drag Java causes into the Node path.
			name: "java caused-by block is untouched",
			input: "Caused by: java.io.IOException: disk\n" +
				"\tat com.a.B.read(B.java:42)\n" +
				"\tat com.a.B.open(B.java:30)\n" +
				"\tat com.a.C.go(C.java:7)",
			want: "Caused by: java.io.IOException: disk\n" +
				"\tat com.a.B.read(B.java:42)\n" +
				"\tat com.a.B.open(B.java:30)\n" +
				"\tat com.a.C.go(C.java:7)",
		},
		{
			name: "node nested cause is folded like a top-level error",
			input: `    [cause]: TypeError: inner
        at a (/app/a.js:1:1)
        at b (/app/b.js:2:2)
        at c (/app/node_modules/x/c.js:3:3)`,
			want: `    [cause]: TypeError: inner
        at a (/app/a.js:1:1)
        at b (/app/b.js:2:2)
    [... 1 node_modules frames]`,
		},
		{
			name: "lowercase identifier ending in Error is not a class",
			input: `handled myError: boom
    at a (/app/node_modules/x/a.js:1:1)
    at b (/app/node_modules/x/b.js:2:2)`,
			want: `handled myError: boom
    at a (/app/node_modules/x/a.js:1:1)
    at b (/app/node_modules/x/b.js:2:2)`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := compressNodeStackTraces(tt.input)
			if got != tt.want {
				t.Errorf("compressNodeStackTraces():\n--- got ---\n%s\n--- want ---\n%s", got, tt.want)
			}
		})
	}
}

func TestCompressStackTraces_PassThrough(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "no stack trace",
			input: "hello world\nfoo bar\nbaz",
		},
		{
			name:  "empty input",
			input: "",
		},
		{
			name:  "normal program output",
			input: "Starting server on :8080\nConnected to database\nReady to accept connections",
		},
		{
			name:  "partial goroutine header not matched",
			input: "goroutine count: 5\nall goroutines running fine",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompressStackTraces(tt.input)
			if got != tt.input {
				t.Errorf("CompressStackTraces() should pass through unchanged:\n--- got ---\n%s\n--- want ---\n%s", got, tt.input)
			}
		})
	}
}

func TestCompressStackTraces_Integrated(t *testing.T) {
	// Test that the top-level function processes all three languages.
	input := `some output before

goroutine 1 [running]:
main.crash()
	/app/main.go:5 +0x10

goroutine 2 [running]:
main.crash()
	/app/main.go:5 +0x10

some middle output

Traceback (most recent call last):
  File "/app/a.py", line 1, in a
    b()
  File "/app/b.py", line 2, in b
    c()
  File "/app/c.py", line 3, in c
    d()
  File "/app/d.py", line 4, in d
    e()
  File "/app/e.py", line 5, in e
    f()
  File "/app/f.py", line 6, in f
    raise ValueError("x")
ValueError: x

Error: ECONNREFUSED
    at connect (/app/db.js:10:5)
    at main (/app/index.js:3:1)
    at Module._compile (node:internal/modules/cjs/loader:1254:14)`

	got := CompressStackTraces(input)

	// Verify Go compression happened.
	if !strings.Contains(got, "[... 1 more goroutine with identical stack]") {
		t.Error("expected Go goroutine compression")
	}

	// Verify Python compression happened.
	if !strings.Contains(got, "[... 4 more frames]") {
		t.Error("expected Python traceback compression")
	}

	// Verify Node compression happened.
	if !strings.Contains(got, "node internals") {
		t.Error("expected Node.js stack compression")
	}

	// Verify error messages are preserved.
	if !strings.Contains(got, "ValueError: x") {
		t.Error("Python exception message must be preserved")
	}
	if !strings.Contains(got, "Error: ECONNREFUSED") {
		t.Error("Node error message must be preserved")
	}
	if !strings.Contains(got, "main.crash()") {
		t.Error("Go top frame must be preserved")
	}
}
