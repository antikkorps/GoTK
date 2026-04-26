//go:build !windows

package main

import (
	"os"
	"syscall"
)

func terminateProcessGroup() {
	pgid, err := syscall.Getpgid(os.Getpid())
	if err != nil {
		return
	}
	_ = syscall.Kill(-pgid, syscall.SIGTERM)
}
