package mcp

import (
	"testing"
	"time"
)

func TestExactRepeat(t *testing.T) {
	tr := NewReRequestTracker(2 * time.Minute)

	rawKey := "grep -rn foo ."
	normKey := NormalizeExecKey(rawKey)

	tr.Record("gotk_exec", rawKey, normKey, 50, false)
	res := tr.Check("gotk_exec", rawKey, normKey, 50, false)

	if !res.IsReRequest {
		t.Fatal("expected re-request")
	}
	if res.Type != "exact" {
		t.Errorf("type = %q, want exact", res.Type)
	}
}

func TestSimilarCommand(t *testing.T) {
	tr := NewReRequestTracker(2 * time.Minute)

	raw1 := "grep -rn foo ."
	raw2 := "grep -rn foo src/"
	norm := NormalizeExecKey(raw1) // both normalize to "exec:grep"

	tr.Record("gotk_exec", raw1, norm, 50, false)
	res := tr.Check("gotk_exec", raw2, norm, 50, false)

	if !res.IsReRequest {
		t.Fatal("expected re-request")
	}
	if res.Type != "similar" {
		t.Errorf("type = %q, want similar", res.Type)
	}
}

func TestEscalation(t *testing.T) {
	tr := NewReRequestTracker(2 * time.Minute)

	normKey := "exec:grep"

	tr.Record("gotk_exec", "grep -rn foo .", normKey, 50, false)
	res := tr.Check("gotk_exec", "grep -rn foo .", normKey, 50, true) // no_truncate = true

	if !res.IsReRequest {
		t.Fatal("expected re-request")
	}
	if res.Type != "escalation" {
		t.Errorf("type = %q, want escalation", res.Type)
	}
}

func TestEscalationMaxLines(t *testing.T) {
	tr := NewReRequestTracker(2 * time.Minute)

	normKey := "exec:grep"

	tr.Record("gotk_exec", "grep -rn foo .", normKey, 50, false)
	res := tr.Check("gotk_exec", "grep -rn foo .", normKey, 200, false) // higher max_lines

	if !res.IsReRequest {
		t.Fatal("expected re-request")
	}
	if res.Type != "escalation" {
		t.Errorf("type = %q, want escalation", res.Type)
	}
}

func TestOutsideWindow(t *testing.T) {
	tr := NewReRequestTracker(1 * time.Millisecond) // very short window

	rawKey := "grep -rn foo ."
	normKey := NormalizeExecKey(rawKey)

	tr.Record("gotk_exec", rawKey, normKey, 50, false)
	time.Sleep(5 * time.Millisecond) // wait past window
	res := tr.Check("gotk_exec", rawKey, normKey, 50, false)

	if res.IsReRequest {
		t.Fatal("should not be re-request after window expires")
	}
}

func TestDifferentTool(t *testing.T) {
	tr := NewReRequestTracker(2 * time.Minute)

	tr.Record("gotk_exec", "foo", "exec:foo", 50, false)
	res := tr.Check("gotk_read", "foo", "read:foo", 0, false)

	if res.IsReRequest {
		t.Fatal("different tools should not match")
	}
}

func TestReRead(t *testing.T) {
	tr := NewReRequestTracker(2 * time.Minute)

	normKey := NormalizeReadKey("/tmp/test.go")

	tr.Record("gotk_read", "/tmp/test.go", normKey, 200, false)
	res := tr.Check("gotk_read", "/tmp/test.go", normKey, 200, false)

	if !res.IsReRequest {
		t.Fatal("expected re-request for same file read")
	}
	if res.Type != "exact" {
		t.Errorf("type = %q, want exact", res.Type)
	}
}

func TestReGrep(t *testing.T) {
	tr := NewReRequestTracker(2 * time.Minute)

	norm := NormalizeGrepKey("TODO", ".")

	tr.Record("gotk_grep", "grep TODO .", norm, 100, false)
	res := tr.Check("gotk_grep", "grep TODO .", norm, 100, false)

	if !res.IsReRequest {
		t.Fatal("expected re-request for same grep")
	}
}

func TestNoReRequestOnFirstCall(t *testing.T) {
	tr := NewReRequestTracker(2 * time.Minute)

	res := tr.Check("gotk_exec", "ls", "exec:ls", 50, false)

	if res.IsReRequest {
		t.Fatal("first call should never be a re-request")
	}
}

func TestPruning(t *testing.T) {
	tr := NewReRequestTracker(1 * time.Millisecond)

	tr.Record("gotk_exec", "old", "exec:old", 50, false)
	time.Sleep(5 * time.Millisecond)

	// Recording a new entry should prune old ones
	tr.Record("gotk_exec", "new", "exec:new", 50, false)

	tr.mu.Lock()
	count := len(tr.history)
	tr.mu.Unlock()

	if count != 1 {
		t.Errorf("history should have 1 entry after pruning, got %d", count)
	}
}

func TestNormalizeKeys(t *testing.T) {
	tests := []struct {
		name string
		fn   func() string
		want string
	}{
		{"exec", func() string { return NormalizeExecKey("grep -rn foo .") }, "exec:grep"},
		{"exec empty", func() string { return NormalizeExecKey("") }, "exec:"},
		{"read", func() string { return NormalizeReadKey("/tmp/f.go") }, "read:/tmp/f.go"},
		{"grep", func() string { return NormalizeGrepKey("TODO", ".") }, "grep:TODO:."},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.fn()
			if got != tt.want {
				t.Errorf("got %q, want %q", got, tt.want)
			}
		})
	}
}
