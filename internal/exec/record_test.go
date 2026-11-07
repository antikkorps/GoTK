package exec

import (
	"reflect"
	"testing"
	"time"
)

type write struct {
	stderr bool
	text   string
}

// record replays writes one second apart, well outside the ambiguity window.
func record(max int, writes ...write) *recorder {
	return recordEvery(time.Second, max, writes...)
}

func recordEvery(gap time.Duration, max int, writes ...write) *recorder {
	r := newRecorder(max)
	clock := time.Unix(0, 0)
	r.now = func() time.Time { clock = clock.Add(gap); return clock }
	for _, w := range writes {
		if w.stderr {
			_, _ = r.stderrWriter().Write([]byte(w.text))
		} else {
			_, _ = r.stdoutWriter().Write([]byte(w.text))
		}
	}
	return r
}

func TestRecorder_AnchorsStderrBlocksToStdoutLines(t *testing.T) {
	r := record(1024,
		write{false, "1 out\n"}, write{true, "2 err\n"},
		write{false, "3 out\n"}, write{true, "4 err\n"}, write{false, "5 out\n"})
	want := []StderrBlock{{AfterLine: 1, Text: "2 err\n"}, {AfterLine: 2, Text: "4 err\n"}}
	if got := r.blocks(); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
	if r.stdout.String() != "1 out\n3 out\n5 out\n" || r.stderr.String() != "2 err\n4 err\n" {
		t.Errorf("full streams must stay intact: %q / %q", r.stdout.String(), r.stderr.String())
	}
}

func TestRecorder_ConsecutiveStderrWritesShareABlock(t *testing.T) {
	r := record(1024, write{true, "a\n"}, write{true, "b"}, write{true, "c\n"}, write{false, "x\n"})
	want := []StderrBlock{{AfterLine: 0, Text: "a\nbc\n"}}
	if got := r.blocks(); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestRecorder_StderrLineSplitByStdoutStaysInFirstBlock(t *testing.T) {
	// A stderr line is anchored where its first byte arrived.
	r := record(1024, write{true, "par"}, write{false, "x\n"}, write{true, "tial\n"})
	want := []StderrBlock{{AfterLine: 0, Text: "partial\n"}}
	if got := r.blocks(); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestRecorder_PartialStdoutLineCountsAsStarted(t *testing.T) {
	// Stderr arriving mid stdout line belongs after that line, not before.
	r := record(1024, write{false, "progress..."}, write{true, "warn\n"}, write{false, "done\n"})
	want := []StderrBlock{{AfterLine: 1, Text: "warn\n"}}
	if got := r.blocks(); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestRecorder_NoStderrNoBlocks(t *testing.T) {
	if got := record(1024, write{false, "only out\n"}).blocks(); got != nil {
		t.Errorf("got %+v, want nil", got)
	}
}

func TestRecorder_CappedStderrKeepsTruncationMarkerInLastBlock(t *testing.T) {
	r := record(4, write{true, "ab\n"}, write{false, "x\n"}, write{true, "cdef\n"})
	blocks := r.blocks()
	if len(blocks) != 2 || blocks[0].Text != "ab\n" {
		t.Fatalf("got %+v", blocks)
	}
	if blocks[1].Text != "c"+truncationMessage {
		t.Errorf("last block = %q, want capped text + marker", blocks[1].Text)
	}
}

// A burst of writes microseconds apart carries no usable order across two
// pipes; stdout that lands within the window counts as written first, which
// matches the historical stdout-then-stderr output.
func TestRecorder_BurstWithinWindowPutsStdoutFirst(t *testing.T) {
	r := recordEvery(10*time.Microsecond, 1024,
		write{true, "2 err\n"}, write{true, "4 err\n"},
		write{false, "1 out\n3 out\n"}, write{false, "5 out\n"})
	want := []StderrBlock{{AfterLine: 3, Text: "2 err\n4 err\n"}}
	if got := r.blocks(); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestRecorder_StdoutAfterWindowStaysAfter(t *testing.T) {
	r := recordEvery(20*time.Millisecond, 1024,
		write{false, "1 out\n"}, write{true, "2 err\n"}, write{false, "3 out\n"})
	want := []StderrBlock{{AfterLine: 1, Text: "2 err\n"}}
	if got := r.blocks(); !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
