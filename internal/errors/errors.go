package errors

import "fmt"

// TimeoutError indicates a command exceeded its execution time limit.
type TimeoutError struct {
	Cause error
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("command timed out: %v", e.Cause)
}

func (e *TimeoutError) Unwrap() error {
	return e.Cause
}

// CommandNotFoundError indicates the requested command binary was not found.
type CommandNotFoundError struct {
	Command string
}

func (e *CommandNotFoundError) Error() string {
	return fmt.Sprintf("command not found: %s", e.Command)
}

// ValidationError indicates invalid user input (flags, config values).
type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}
