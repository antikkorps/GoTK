//go:build !windows

package main

import (
	"os"
	"syscall"
)

// signalChildProcesses sends SIGTERM to every process in gotk's process
// group so any children spawned by the wrapped command are torn down
// alongside us. The negative PID selects the group as a whole.
//
// We exclude ourselves implicitly: by the time this runs the handler has
// already decided to exit, so an extra SIGTERM arriving in our own mailbox
// is harmless — we fall through to os.Exit right after.
func signalChildProcesses() {
	pgid, err := syscall.Getpgid(os.Getpid())
	if err != nil {
		return
	}
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
}
