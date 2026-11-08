package main

import (
	"os"
	"testing"
)

// /dev/null is a character device, which fooled the ModeCharDevice check:
// `gotk </dev/null` printed the usage and `gotk install claude </dev/null`
// prompted (#112).
func TestIsTerminal_DevNullIsNot(t *testing.T) {
	f, err := os.Open(os.DevNull)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close() //nolint:errcheck
	if isTerminal(f) {
		t.Error("isTerminal(/dev/null) = true, want false")
	}
}
