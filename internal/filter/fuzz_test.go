package filter

import "testing"

// Fuzz tests for all public filter functions.
// Each test verifies the filter does not panic on arbitrary input.

func FuzzStripANSI(f *testing.F) {
	f.Add("")
	f.Add("normal text")
	f.Add("\x1b[31mred\x1b[0m")
	f.Add("\x1b[1;2;3;4;5;6;7;8;9;0m")
	f.Add("\x1b]0;title\x07")
	f.Fuzz(func(t *testing.T, input string) {
		out := StripANSI(input)
		if len(out) > len(input) {
			t.Errorf("StripANSI output larger than input: %d > %d", len(out), len(input))
		}
	})
}

func FuzzNormalizeWhitespace(f *testing.F) {
	f.Add("")
	f.Add("a\n\n\n\nb")
	f.Add("  trailing   \n  spaces  ")
	f.Add("\t\t\ttabs\t\t")
	f.Fuzz(func(t *testing.T, input string) {
		_ = NormalizeWhitespace(input)
	})
}

func FuzzDedup(f *testing.F) {
	f.Add("")
	f.Add("a\na\na\nb\nb")
	f.Add("single line")
	f.Add("\n\n\n")
	f.Fuzz(func(t *testing.T, input string) {
		_ = Dedup(input)
	})
}

func FuzzCompressPaths(f *testing.F) {
	f.Add("")
	f.Add("/usr/local/bin/foo")
	f.Add("/home/user/.config/gotk/config.toml")
	f.Add("relative/path/to/file.go:123: error")
	f.Fuzz(func(t *testing.T, input string) {
		_ = CompressPaths(input)
	})
}

func FuzzTrimEmpty(f *testing.F) {
	f.Add("")
	f.Add("---\ncontent\n===")
	f.Add("  \n  \n  ")
	f.Add("~~~~~~\n--------\n========")
	f.Fuzz(func(t *testing.T, input string) {
		_ = TrimEmpty(input)
	})
}

func FuzzRedactSecrets(f *testing.F) {
	f.Add("")
	f.Add("API_KEY=sk-test1234567890")
	f.Add("ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef")
	f.Add("postgres://user:pass@host:5432/db")
	f.Add("-----BEGIN RSA PRIVATE KEY-----\ndata\n-----END RSA PRIVATE KEY-----")
	f.Add("Authorization: Bearer eyJhbGciOiJIUzI1NiJ9.eyJ0ZXN0IjoxfQ.sig")
	f.Fuzz(func(t *testing.T, input string) {
		_ = RedactSecrets(input)
	})
}

func FuzzCompressStackTraces(f *testing.F) {
	f.Add("")
	f.Add("panic: runtime error: index out of range\ngoroutine 1 [running]:\nmain.main()\n\t/app/main.go:10 +0x1")
	f.Add("Traceback (most recent call last):\n  File \"app.py\", line 10\nValueError: bad")
	f.Add("Error: something\n    at Object.foo (/app.js:1:2)\n    at Module._compile (internal/modules:3:4)")
	f.Fuzz(func(t *testing.T, input string) {
		_ = CompressStackTraces(input)
	})
}

func FuzzSummarize(f *testing.F) {
	f.Add("")
	f.Add("short output")
	// Build a >100 line input
	long := ""
	for i := 0; i < 150; i++ {
		long += "line content here\n"
	}
	f.Add(long)
	f.Fuzz(func(t *testing.T, input string) {
		_ = Summarize(input)
	})
}

func FuzzTruncateWithLimit(f *testing.F) {
	f.Add("", 50)
	f.Add("one\ntwo\nthree", 2)
	f.Add("a\nb\nc\nd\ne\nf\ng\nh\ni\nj", 5)
	long := ""
	for i := 0; i < 200; i++ {
		long += "line\n"
	}
	f.Add(long, 50)
	f.Fuzz(func(t *testing.T, input string, maxLines int) {
		// Clamp to reasonable values to avoid OOM
		if maxLines < 0 {
			maxLines = 0
		}
		if maxLines > 10000 {
			maxLines = 10000
		}
		fn := TruncateWithLimit(maxLines)
		_ = fn(input)
	})
}
