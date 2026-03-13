package errors

import (
	"errors"
	"fmt"
	"strings"
	"testing"
)

func TestTimeoutError(t *testing.T) {
	cause := fmt.Errorf("context deadline exceeded")
	err := &TimeoutError{Cause: cause}

	if !strings.Contains(err.Error(), "command timed out") {
		t.Errorf("expected 'command timed out' in %q", err.Error())
	}
	if !errors.Is(err.Unwrap(), cause) {
		t.Error("Unwrap should return cause")
	}
}

func TestCommandNotFoundError(t *testing.T) {
	err := &CommandNotFoundError{Command: "foo"}
	if !strings.Contains(err.Error(), "foo") {
		t.Errorf("expected command name in %q", err.Error())
	}
}

func TestValidationError(t *testing.T) {
	err := &ValidationError{Field: "max_lines", Message: "must be positive"}
	if !strings.Contains(err.Error(), "max_lines") {
		t.Errorf("expected field in %q", err.Error())
	}
	if !strings.Contains(err.Error(), "must be positive") {
		t.Errorf("expected message in %q", err.Error())
	}
}
