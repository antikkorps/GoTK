//go:build windows

package main

// Windows has no Unix-style process groups. The os/exec child process
// receives the console signal (Ctrl+C) directly via the shared console,
// so explicit propagation here is a no-op.
func terminateProcessGroup() {}
