//go:build !windows

package update

import "os"

// replaceBinary atomically places the binary at tmpPath at exePath, preserving
// the original mode. On Unix os.Rename replaces the inode while the running
// process keeps executing from the old in-memory image — safe to call from
// within the running gotk.
func replaceBinary(tmpPath, exePath string) error {
	if info, err := os.Stat(exePath); err == nil {
		_ = os.Chmod(tmpPath, info.Mode().Perm())
	}

	// Move to a sibling path first so the final rename is a same-filesystem
	// operation (os.Rename across filesystems fails with EXDEV).
	stagePath := exePath + ".new"
	if err := moveFile(tmpPath, stagePath); err != nil {
		return err
	}
	if err := os.Rename(stagePath, exePath); err != nil {
		os.Remove(stagePath) //nolint:errcheck
		return err
	}
	return nil
}

// sweepStaleReplacements is a no-op on Unix — os.Rename swaps inodes and the
// previous binary is unlinked immediately by the rename.
func sweepStaleReplacements(exePath string) {}
