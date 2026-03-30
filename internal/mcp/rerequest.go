package mcp

import (
	"crypto/sha256"
	"fmt"
	"strings"
	"sync"
	"time"
)

// DefaultReRequestWindow is the time window for detecting re-requests.
const DefaultReRequestWindow = 2 * time.Minute

// ReRequestResult describes whether a request was a re-request.
type ReRequestResult struct {
	IsReRequest   bool
	Type          string // "exact", "similar", "escalation"
	TimeSincePrev time.Duration
}

// requestRecord stores a past request for similarity comparison.
type requestRecord struct {
	timestamp time.Time
	tool      string // "gotk_exec", "gotk_read", "gotk_grep", "gotk_filter"
	key       string // normalized key for similarity
	rawKey    string // exact key (full command/args)
	maxLines  int
	noTrunc   bool
}

// ReRequestTracker detects when an LLM re-requests similar commands.
type ReRequestTracker struct {
	mu      sync.Mutex
	history []requestRecord
	window  time.Duration
}

// NewReRequestTracker creates a tracker with the given detection window.
func NewReRequestTracker(window time.Duration) *ReRequestTracker {
	return &ReRequestTracker{window: window}
}

// Check tests whether a new request matches a recent one.
func (t *ReRequestTracker) Check(tool, rawKey, normKey string, maxLines int, noTrunc bool) ReRequestResult {
	if t == nil {
		return ReRequestResult{}
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.pruneExpired(now)

	for i := len(t.history) - 1; i >= 0; i-- {
		prev := t.history[i]
		if prev.tool != tool {
			continue
		}

		since := now.Sub(prev.timestamp)

		// Check normalized key first to detect escalation
		if prev.key == normKey {
			// Escalation: same base command but requesting more output
			if isEscalation(prev.maxLines, prev.noTrunc, maxLines, noTrunc) {
				return ReRequestResult{IsReRequest: true, Type: "escalation", TimeSincePrev: since}
			}
			// Exact repeat: same raw key, same params
			if prev.rawKey == rawKey {
				return ReRequestResult{IsReRequest: true, Type: "exact", TimeSincePrev: since}
			}
			// Same base command, different args → similar
			return ReRequestResult{IsReRequest: true, Type: "similar", TimeSincePrev: since}
		}
	}

	return ReRequestResult{}
}

// Record adds a request to history.
func (t *ReRequestTracker) Record(tool, rawKey, normKey string, maxLines int, noTrunc bool) {
	if t == nil {
		return
	}
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	t.pruneExpired(now)

	t.history = append(t.history, requestRecord{
		timestamp: now,
		tool:      tool,
		key:       normKey,
		rawKey:    rawKey,
		maxLines:  maxLines,
		noTrunc:   noTrunc,
	})
}

// pruneExpired removes entries older than the window. Must be called with mu held.
func (t *ReRequestTracker) pruneExpired(now time.Time) {
	cutoff := now.Add(-t.window)
	i := 0
	for i < len(t.history) && t.history[i].timestamp.Before(cutoff) {
		i++
	}
	if i > 0 {
		t.history = t.history[i:]
	}
}

// isEscalation returns true if the new request asks for more output than the previous one.
func isEscalation(prevMaxLines int, prevNoTrunc bool, newMaxLines int, newNoTrunc bool) bool {
	// Went from truncated to no-truncate
	if !prevNoTrunc && newNoTrunc {
		return true
	}
	// Increased max lines
	if newMaxLines > prevMaxLines && prevMaxLines > 0 {
		return true
	}
	return false
}

// NormalizeExecKey returns a normalized key for exec commands.
// Extracts the base command name for similarity matching.
func NormalizeExecKey(command string) string {
	fields := strings.Fields(command)
	if len(fields) == 0 {
		return "exec:"
	}
	return "exec:" + fields[0]
}

// NormalizeReadKey returns a normalized key for read requests.
func NormalizeReadKey(path string) string {
	return "read:" + path
}

// NormalizeGrepKey returns a normalized key for grep requests.
func NormalizeGrepKey(pattern, path string) string {
	return "grep:" + pattern + ":" + path
}

// NormalizeFilterKey returns a normalized key for filter requests.
// Uses a hash prefix since input can be large.
func NormalizeFilterKey(input string) string {
	h := sha256.Sum256([]byte(input))
	return fmt.Sprintf("filter:%x", h[:8])
}
