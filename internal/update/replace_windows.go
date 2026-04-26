//go:build windows

package update

import "os"

// replaceBinary swaps in the new binary using Windows's pending-replace
// pattern: the running .exe cannot be deleted or overwritten, but it CAN be
// renamed. So we rename the running binary to gotk.exe.old, then move the new
// binary into its place. The .old file is cleaned up by sweepStaleReplacements
// the next time gotk runs (when it is no longer locked).
func replaceBinary(tmpPath, exePath string) error {
	if info, err := os.Stat(exePath); err == nil {
		_ = os.Chmod(tmpPath, info.Mode().Perm())
	}

	oldPath := exePath + ".old"
	// Best-effort: remove a stale .old left over from a prior update.
	_ = os.Remove(oldPath)

	if err := os.Rename(exePath, oldPath); err != nil {
		return err
	}
	if err := moveFile(tmpPath, exePath); err != nil {
		// Try to restore the previous binary so the user is not left without
		// a working gotk.
		_ = os.Rename(oldPath, exePath)
		return err
	}
	// .old still locked by the running process; sweepStaleReplacements will
	// remove it on the next invocation.
	return nil
}

// sweepStaleReplacements removes a leftover gotk.exe.old next to the current
// executable. Called at the start of update.Run so a successful update from a
// previous session is fully cleaned up before we attempt another one.
func sweepStaleReplacements(exePath string) {
	_ = os.Remove(exePath + ".old")
}
