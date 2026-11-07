package exec

import (
	"bytes"
	"io"
	"sort"
	"sync"
	"time"
)

// orderWindow is how close two writes on different pipes can be before
// their relative order is treated as unknown. In a burst both pipes fill
// before either is read, so arrival order is noise; stdout landing within
// the window after a stderr line counts as written first, which matches the
// historical stdout-then-stderr output.
const orderWindow = 5 * time.Millisecond

// StderrBlock is a run of stderr lines together with where it arrived
// relative to stdout: AfterLine stdout lines (counting a partially written
// one) had started before the block's first byte. It lets callers put
// stderr back in chronological position after filtering stdout (issue #100).
type StderrBlock struct {
	AfterLine int
	Text      string
}

// blockMark records where a block starts in the stderr buffer and when.
type blockMark struct {
	afterLine int
	start     int
	at        time.Time
}

// outEvent records how many stdout lines had started after a write.
type outEvent struct {
	at      time.Time
	started int
}

// recorder captures both streams into capped buffers and notes, for each
// stderr line, how much stdout preceded it. Ordering is only as precise as
// what the OS delivers on two separate pipes: reliable for lines written
// milliseconds apart (make, docker compose), unknown inside a burst.
type recorder struct {
	mu             sync.Mutex
	now            func() time.Time
	stdout, stderr *LimitedBuffer
	outNewlines    int
	outMidLine     bool
	outEvents      []outEvent
	errAtLineStart bool
	marks          []blockMark
}

func newRecorder(max int) *recorder {
	return &recorder{
		now:            time.Now,
		stdout:         NewLimitedBuffer(max),
		stderr:         NewLimitedBuffer(max),
		errAtLineStart: true,
	}
}

// startedLines counts stdout lines begun so far, a partial one included.
func (r *recorder) startedLines() int {
	if r.outMidLine {
		return r.outNewlines + 1
	}
	return r.outNewlines
}

type streamWriter func([]byte) (int, error)

func (w streamWriter) Write(p []byte) (int, error) { return w(p) }

func (r *recorder) stdoutWriter() io.Writer { return streamWriter(r.writeOut) }
func (r *recorder) stderrWriter() io.Writer { return streamWriter(r.writeErr) }

func (r *recorder) writeOut(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(p) > 0 {
		r.outNewlines += bytes.Count(p, []byte{'\n'})
		r.outMidLine = p[len(p)-1] != '\n'
		r.outEvents = append(r.outEvents, outEvent{at: r.now(), started: r.startedLines()})
	}
	return r.stdout.Write(p)
}

func (r *recorder) writeErr(p []byte) (int, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	base := r.stderr.written
	for i := 0; i < len(p); {
		if r.errAtLineStart {
			r.mark(base + i)
		}
		nl := bytes.IndexByte(p[i:], '\n')
		if nl < 0 {
			r.errAtLineStart = false
			break
		}
		i += nl + 1
		r.errAtLineStart = true
	}
	return r.stderr.Write(p)
}

// mark opens a new block at stderr offset off, unless stdout has not moved
// since the previous block (then the line just extends it).
func (r *recorder) mark(off int) {
	if off >= r.stderr.max {
		return
	}
	after := r.startedLines()
	if n := len(r.marks); n > 0 && r.marks[n-1].afterLine == after {
		return
	}
	r.marks = append(r.marks, blockMark{afterLine: after, start: off, at: r.now()})
}

// startedBy returns how many stdout lines had started by time t.
func (r *recorder) startedBy(t time.Time) int {
	i := sort.Search(len(r.outEvents), func(i int) bool { return r.outEvents[i].at.After(t) })
	if i == 0 {
		return 0
	}
	return r.outEvents[i-1].started
}

// blocks slices the captured stderr into its anchored blocks, applying the
// ordering window; blocks that end up at the same position are merged. The
// last block runs to the end of the stderr string, truncation marker
// included.
func (r *recorder) blocks() []StderrBlock {
	text := r.stderr.String()
	var out []StderrBlock
	for i, m := range r.marks {
		end := len(text)
		if i+1 < len(r.marks) {
			end = r.marks[i+1].start
		}
		after := r.startedBy(m.at.Add(orderWindow))
		if n := len(out); n > 0 && out[n-1].AfterLine == after {
			out[n-1].Text += text[m.start:end]
			continue
		}
		out = append(out, StderrBlock{AfterLine: after, Text: text[m.start:end]})
	}
	return out
}
