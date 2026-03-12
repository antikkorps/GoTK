package filter

import (
	"strings"
	"testing"
)

func TestStreamFilter_StripANSI(t *testing.T) {
	sf := NewStreamFilter(StreamConfig{StripANSI: true})

	line, emit := sf.ProcessLine("\x1b[31mred text\x1b[0m")
	if !emit {
		t.Fatal("expected emit=true")
	}
	if line != "red text" {
		t.Errorf("expected 'red text', got %q", line)
	}
}

func TestStreamFilter_CompressPaths(t *testing.T) {
	sf := NewStreamFilter(StreamConfig{CompressPaths: true})

	// The cwd should be compressed to "./"
	if sf.cwd == "" {
		t.Skip("could not determine cwd")
	}

	input := sf.cwd + "/main.go"
	line, emit := sf.ProcessLine(input)
	if !emit {
		t.Fatal("expected emit=true")
	}
	if line != "./main.go" {
		t.Errorf("expected './main.go', got %q", line)
	}
}

func TestStreamFilter_TrimDecorative(t *testing.T) {
	sf := NewStreamFilter(StreamConfig{TrimDecorative: true})

	// A long decorative line should be suppressed.
	_, emit := sf.ProcessLine("=====================================")
	if emit {
		t.Error("expected decorative line to be suppressed")
	}

	// A normal line should pass through.
	line, emit := sf.ProcessLine("hello world")
	if !emit {
		t.Fatal("expected emit=true")
	}
	if line != "hello world" {
		t.Errorf("expected 'hello world', got %q", line)
	}
}

func TestStreamFilter_Dedup(t *testing.T) {
	sf := NewStreamFilter(StreamConfig{Dedup: true})

	// First line should always emit.
	line, emit := sf.ProcessLine("AAA")
	if !emit {
		t.Fatal("first line should emit")
	}
	if line != "AAA" {
		t.Errorf("expected 'AAA', got %q", line)
	}

	// Second identical line should be buffered.
	_, emit = sf.ProcessLine("AAA")
	if emit {
		t.Error("duplicate line should not emit")
	}

	// Third identical line should also be buffered.
	_, emit = sf.ProcessLine("AAA")
	if emit {
		t.Error("duplicate line should not emit")
	}

	// A different line should emit the dup marker and the new line.
	line, emit = sf.ProcessLine("BBB")
	if !emit {
		t.Fatal("different line should emit")
	}
	if !strings.Contains(line, "2 duplicate lines") {
		t.Errorf("expected dup marker for 2 duplicates, got %q", line)
	}
	if !strings.Contains(line, "BBB") {
		t.Errorf("expected 'BBB' in output, got %q", line)
	}
}

func TestStreamFilter_Dedup_SingleDuplicate(t *testing.T) {
	sf := NewStreamFilter(StreamConfig{Dedup: true})

	sf.ProcessLine("line1")
	sf.ProcessLine("line1") // 1 dup

	line, emit := sf.ProcessLine("line2")
	if !emit {
		t.Fatal("expected emit")
	}
	if !strings.Contains(line, "1 duplicate line") {
		t.Errorf("expected singular dup marker, got %q", line)
	}
}

func TestStreamFilter_Flush(t *testing.T) {
	sf := NewStreamFilter(StreamConfig{Dedup: true})

	sf.ProcessLine("x")
	sf.ProcessLine("x") // buffered dup
	sf.ProcessLine("x") // buffered dup

	// Flush should return the pending marker.
	flushed := sf.Flush()
	if flushed == "" {
		t.Fatal("expected flush to return dup marker")
	}
	if !strings.Contains(flushed, "2 duplicate lines") {
		t.Errorf("expected '2 duplicate lines' in flush, got %q", flushed)
	}

	// Second flush should return empty.
	flushed2 := sf.Flush()
	if flushed2 != "" {
		t.Errorf("expected empty second flush, got %q", flushed2)
	}
}

func TestStreamFilter_FlushNoPending(t *testing.T) {
	sf := NewStreamFilter(StreamConfig{Dedup: true})

	sf.ProcessLine("a")
	sf.ProcessLine("b")

	flushed := sf.Flush()
	if flushed != "" {
		t.Errorf("expected empty flush with no pending dups, got %q", flushed)
	}
}

func TestStreamFilter_NormalizeWhitespace_TrailingSpaces(t *testing.T) {
	sf := NewStreamFilter(StreamConfig{NormalizeWhitespace: true})

	line, emit := sf.ProcessLine("hello   ")
	if !emit {
		t.Fatal("expected emit=true")
	}
	if line != "hello" {
		t.Errorf("expected trailing spaces trimmed, got %q", line)
	}
}

func TestStreamFilter_NormalizeWhitespace_CollapseBlankLines(t *testing.T) {
	sf := NewStreamFilter(StreamConfig{NormalizeWhitespace: true})

	// First blank line should pass.
	_, emit1 := sf.ProcessLine("")
	if !emit1 {
		t.Error("first blank line should emit")
	}

	// Second consecutive blank line should be suppressed.
	_, emit2 := sf.ProcessLine("")
	if emit2 {
		t.Error("second consecutive blank line should be suppressed")
	}

	// Third blank line also suppressed.
	_, emit3 := sf.ProcessLine("   ")
	if emit3 {
		t.Error("third consecutive blank line should be suppressed")
	}

	// A non-blank line resets the counter.
	_, emit4 := sf.ProcessLine("hello")
	if !emit4 {
		t.Error("non-blank line should emit")
	}

	// Next blank line should emit again.
	_, emit5 := sf.ProcessLine("")
	if !emit5 {
		t.Error("first blank line after content should emit")
	}
}

func TestStreamFilter_AllFilters(t *testing.T) {
	sf := NewStreamFilter(StreamConfig{
		StripANSI:           true,
		CompressPaths:       true,
		Dedup:               true,
		TrimDecorative:      true,
		NormalizeWhitespace: true,
	})

	// ANSI + trailing whitespace.
	line, emit := sf.ProcessLine("\x1b[1mhello\x1b[0m   ")
	if !emit {
		t.Fatal("expected emit")
	}
	if line != "hello" {
		t.Errorf("expected 'hello', got %q", line)
	}

	// Decorative line.
	_, emit = sf.ProcessLine("========================================")
	if emit {
		t.Error("decorative line should be suppressed")
	}
}

func TestStreamFilter_NoFilters(t *testing.T) {
	sf := NewStreamFilter(StreamConfig{})

	line, emit := sf.ProcessLine("\x1b[31mhello\x1b[0m   ")
	if !emit {
		t.Fatal("expected emit")
	}
	// Nothing should be changed with no filters enabled.
	if line != "\x1b[31mhello\x1b[0m   " {
		t.Errorf("expected unchanged line, got %q", line)
	}
}
