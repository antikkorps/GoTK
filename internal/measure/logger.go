package measure

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/antikkorps/GoTK/internal/paths"
)

// Logger writes measurement entries as JSONL to a file.
type Logger struct {
	mu        sync.Mutex
	file      *os.File
	path      string
	sessionID string
	maxSize   int // max file size in bytes, 0 = unlimited
}

// NewLogger creates a Logger that appends entries to the given path.
// It creates parent directories if needed. sessionID identifies this process.
// maxSize sets the maximum log file size in bytes (0 = unlimited).
// When exceeded, the oldest half of entries is discarded.
func NewLogger(path, sessionID string, maxSize ...int) (*Logger, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("measure: create dir %s: %w", dir, err)
	}

	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("measure: open %s: %w", path, err)
	}

	ms := 0
	if len(maxSize) > 0 {
		ms = maxSize[0]
	}

	return &Logger{file: f, path: path, sessionID: sessionID, maxSize: ms}, nil
}

// SessionID returns a unique session identifier: {unix_ms}-{pid}.
func SessionID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixMilli(), os.Getpid())
}

// Log writes a single entry to the log file.
// If maxSize > 0 and the file exceeds it, the oldest half of entries is discarded.
func (l *Logger) Log(e Entry) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if e.SessionID == "" {
		e.SessionID = l.sessionID
	}
	if e.Timestamp == "" {
		e.Timestamp = time.Now().UTC().Format(time.RFC3339)
	}

	data, err := json.Marshal(e)
	if err != nil {
		return err
	}

	// Check if rotation is needed before writing
	if l.maxSize > 0 {
		if info, err := l.file.Stat(); err == nil && info.Size() > int64(l.maxSize) {
			if err := l.rotate(); err != nil {
				return fmt.Errorf("measure: rotation failed: %w", err)
			}
		}
	}

	_, err = fmt.Fprintf(l.file, "%s\n", data)
	return err
}

// rotate keeps the most recent half of the log file.
// Uses atomic file ops (write to temp file + rename) to prevent data loss.
// Must be called with l.mu held.
func (l *Logger) rotate() error {
	// Close current file, read contents, keep second half, rewrite atomically
	if err := l.file.Close(); err != nil {
		return fmt.Errorf("measure: close for rotation: %w", err)
	}

	data, err := os.ReadFile(l.path)
	if err != nil {
		// Reopen in append mode and continue
		l.file, err = os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
		if err != nil {
			return fmt.Errorf("measure: reopen after failed read: %w", err)
		}
		return fmt.Errorf("measure: read for rotation: %w", err)
	}

	lines := strings.Split(string(data), "\n")
	// Remove trailing empty line from split
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}

	// Keep the most recent half
	half := len(lines) / 2
	kept := strings.Join(lines[half:], "\n") + "\n"

	// Atomic write: temp file + rename to prevent corruption
	tmpPath := l.path + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(kept), 0600); err != nil {
		fmt.Fprintf(os.Stderr, "[gotk] measure: rotation write failed: %v\n", err)
	} else if err := os.Rename(tmpPath, l.path); err != nil {
		fmt.Fprintf(os.Stderr, "[gotk] measure: rotation rename failed: %v\n", err)
	}

	l.file, err = os.OpenFile(l.path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return fmt.Errorf("measure: reopen after rotation: %w", err)
	}
	return nil
}

// Close closes the underlying log file.
func (l *Logger) Close() error {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.file.Close()
}

// ReadEntries reads all entries from a JSONL file.
func ReadEntries(path string) ([]Entry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return parseEntries(string(data))
}

// ReadEntriesSince reads entries from the log file with timestamps at or after since.
func ReadEntriesSince(path string, since time.Time) ([]Entry, error) {
	all, err := ReadEntries(path)
	if err != nil {
		return nil, err
	}

	sinceStr := since.UTC().Format(time.RFC3339)
	var filtered []Entry
	for _, e := range all {
		if e.Timestamp >= sinceStr {
			filtered = append(filtered, e)
		}
	}
	return filtered, nil
}

func parseEntries(data string) ([]Entry, error) {
	var entries []Entry
	for _, line := range strings.Split(data, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var e Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			continue // skip malformed lines
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// DefaultLogPath returns the default measurement log path.
func DefaultLogPath() string {
	if dir, ok := paths.DataDir(); ok {
		return filepath.Join(dir, "measure.jsonl")
	}
	return "measure.jsonl"
}
