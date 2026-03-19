package learn

import (
	"testing"

	"github.com/antikkorps/GoTK/internal/classify"
)

func TestCollectorObserve(t *testing.T) {
	c := NewCollector("go test ./...", "test-session")

	input := `ok  	mypackage	0.5s
FAIL	mypackage/sub	0.1s
--- FAIL: TestSomething (0.00s)
    expected 1, got 2
`

	c.Observe(input)

	obs := c.Observations()
	if len(obs) != 4 {
		t.Fatalf("Observations count = %d, want 4", len(obs))
	}

	// Check command is set on all observations
	for i, o := range obs {
		if o.Command != "go test ./..." {
			t.Errorf("obs[%d].Command = %q, want %q", i, o.Command, "go test ./...")
		}
		if o.SessionID != "test-session" {
			t.Errorf("obs[%d].SessionID = %q, want %q", i, o.SessionID, "test-session")
		}
		if o.Timestamp == "" {
			t.Errorf("obs[%d].Timestamp is empty", i)
		}
		if o.Normalized == "" {
			t.Errorf("obs[%d].Normalized is empty", i)
		}
	}

	// "FAIL" should be classified as Error
	if obs[1].Level != int(classify.Error) {
		t.Errorf("FAIL line level = %d, want %d (Error)", obs[1].Level, classify.Error)
	}
}

func TestCollectorEmpty(t *testing.T) {
	c := NewCollector("echo", "s1")
	c.Observe("")
	if c.Count() != 0 {
		t.Errorf("Count after empty observe = %d, want 0", c.Count())
	}
}

func TestCollectorSkipsBlankLines(t *testing.T) {
	c := NewCollector("cmd", "s1")
	c.Observe("hello\n\n\nworld\n\n")
	if c.Count() != 2 {
		t.Errorf("Count = %d, want 2 (blank lines skipped)", c.Count())
	}
}
