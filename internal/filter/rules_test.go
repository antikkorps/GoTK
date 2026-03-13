package filter

import (
	"strings"
	"testing"
)

func TestRemoveByRules_Basic(t *testing.T) {
	fn := RemoveByRules([]string{`^DEBUG:`, `^\s*$`})
	input := "INFO: hello\nDEBUG: noisy\nERROR: bad\n\nINFO: world"
	got := fn(input)

	if strings.Contains(got, "DEBUG:") {
		t.Errorf("DEBUG line should be removed, got: %q", got)
	}
	if !strings.Contains(got, "INFO: hello") {
		t.Errorf("INFO lines should be preserved, got: %q", got)
	}
	if !strings.Contains(got, "ERROR: bad") {
		t.Errorf("ERROR line should be preserved, got: %q", got)
	}
}

func TestRemoveByRules_NoPatterns(t *testing.T) {
	fn := RemoveByRules(nil)
	input := "keep everything"
	got := fn(input)
	if got != input {
		t.Errorf("no patterns should be a no-op, got: %q", got)
	}
}

func TestRemoveByRules_InvalidRegex(t *testing.T) {
	// Invalid regex should be silently skipped
	fn := RemoveByRules([]string{`[invalid`, `^DEBUG:`})
	input := "DEBUG: noisy\nINFO: ok"
	got := fn(input)
	if strings.Contains(got, "DEBUG:") {
		t.Errorf("valid pattern should still work, got: %q", got)
	}
}

func TestKeepByRules_RestoresMissingLines(t *testing.T) {
	original := "ERROR: critical failure\nDEBUG: noise\nWARNING: watch out"
	// Simulate a filter that removed the ERROR line
	filtered := "DEBUG: noise\nWARNING: watch out"

	fn := KeepByRules([]string{`^ERROR:`}, original)
	got := fn(filtered)

	if !strings.Contains(got, "ERROR: critical failure") {
		t.Errorf("ERROR line should be restored, got: %q", got)
	}
	if !strings.Contains(got, "preserved by always_keep") {
		t.Errorf("should show preservation marker, got: %q", got)
	}
}

func TestKeepByRules_NoOpIfPresent(t *testing.T) {
	original := "ERROR: bad\nINFO: ok"
	filtered := "ERROR: bad\nINFO: ok"

	fn := KeepByRules([]string{`^ERROR:`}, original)
	got := fn(filtered)

	if strings.Contains(got, "preserved by always_keep") {
		t.Errorf("should not add marker if line is already present, got: %q", got)
	}
}

func TestKeepByRules_NoPatterns(t *testing.T) {
	fn := KeepByRules(nil, "anything")
	got := fn("pass through")
	if got != "pass through" {
		t.Errorf("no patterns should be a no-op, got: %q", got)
	}
}

func TestRemoveByRules_RegexPatterns(t *testing.T) {
	fn := RemoveByRules([]string{`\d{4}-\d{2}-\d{2}T\d{2}:\d{2}`, `^#`})
	input := "# comment\n2024-01-15T10:30:00 log line\nreal content\n# another comment"
	got := fn(input)

	if strings.Contains(got, "# comment") {
		t.Errorf("comments should be removed, got: %q", got)
	}
	if strings.Contains(got, "2024-01-15") {
		t.Errorf("timestamp lines should be removed, got: %q", got)
	}
	if !strings.Contains(got, "real content") {
		t.Errorf("real content should be kept, got: %q", got)
	}
}
