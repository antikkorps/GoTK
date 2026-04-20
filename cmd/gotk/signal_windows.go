//go:build windows

package main

// signalChildProcesses is a no-op on Windows. Windows does not have Unix-
// style process groups; the Go runtime already translates console Ctrl+C
// into a signal that propagates to child console processes attached to the
// same console, so there's nothing useful to add here. If a background
// service scenario comes along later (no attached console), we'll need
// to use Job Objects — not yet required.
func signalChildProcesses() {}
