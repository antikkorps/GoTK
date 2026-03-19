package learn

import (
	"strings"
	"time"

	"github.com/antikkorps/GoTK/internal/classify"
)

// Collector gathers observations from command output.
// It classifies each line, normalizes it, and tracks frequency.
type Collector struct {
	command   string
	sessionID string
	lines     []Observation
}

// NewCollector creates a Collector for a specific command invocation.
func NewCollector(command, sessionID string) *Collector {
	return &Collector{
		command:   command,
		sessionID: sessionID,
	}
}

// Observe processes raw command output, classifying and normalizing each line.
func (c *Collector) Observe(rawOutput string) {
	ts := time.Now().UTC().Format(time.RFC3339)
	for _, line := range strings.Split(rawOutput, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" {
			continue
		}

		level := classify.Classify(line)
		norm := Normalize(line)

		c.lines = append(c.lines, Observation{
			Timestamp:  ts,
			Command:    c.command,
			Normalized: norm,
			Level:      int(level),
			SessionID:  c.sessionID,
		})
	}
}

// Observations returns the collected observations ready for storage.
func (c *Collector) Observations() []Observation {
	return c.lines
}

// Count returns the number of collected observations.
func (c *Collector) Count() int {
	return len(c.lines)
}
