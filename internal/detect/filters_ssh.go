package detect

import (
	"regexp"
	"strconv"
	"strings"
)

var (
	// SSH connection banners and info
	sshBanner  = regexp.MustCompile(`^(Warning: Permanently added|Pseudo-terminal|Connection to .+ closed|Authenticated to)`)
	sshDebug   = regexp.MustCompile(`^debug\d+:`)
	sshHostKey = regexp.MustCompile(`^(ECDSA|RSA|ED25519) key fingerprint is`)
	// SCP progress: "file.txt    100%   50KB   1.2MB/s   00:00"
	scpProgress = regexp.MustCompile(`\d+%\s+[\d.]+[kKMG]?B\s+[\d.]+[kKMG]?B/s`)
	// rsync progress and stats
	rsyncProgress = regexp.MustCompile(`^\s*[\d,]+\s+\d+%\s+[\d.]+[kKMG]?B/s`)
	rsyncStats    = regexp.MustCompile(`^(sent|total size|speedup is)`)
	// SSH remote MOTD banners (multi-line, often decorative)
	// Matches lines that start AND end with 3+ of the same decoration character
	sshMOTDLine = regexp.MustCompile(`^(#{3,}|^\*{3,}|^={3,})`)
	// Lines that are framed with # (e.g., "#  Welcome to Server  #")
	sshMOTDFramed = regexp.MustCompile(`^#.+#\s*$`)
)

// compressSSHOutput compresses ssh/scp/rsync output.
// Preserves: command output, errors, rsync summary stats.
// Removes: connection banners, debug lines, SCP progress bars, MOTD decorations.
func compressSSHOutput(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	scpFileCount := 0
	rsyncProgressCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			result = append(result, line)
			continue
		}

		// Skip SSH connection banners and host key info
		if sshBanner.MatchString(trimmed) {
			continue
		}
		if sshHostKey.MatchString(trimmed) {
			continue
		}

		// Skip SSH debug lines
		if sshDebug.MatchString(trimmed) {
			continue
		}

		// Skip MOTD decorative lines and framed banners
		if sshMOTDLine.MatchString(trimmed) || sshMOTDFramed.MatchString(trimmed) {
			continue
		}

		// Compress SCP progress lines: count files transferred
		if scpProgress.MatchString(trimmed) {
			scpFileCount++
			continue
		}

		// Compress rsync progress lines
		if rsyncProgress.MatchString(trimmed) {
			rsyncProgressCount++
			continue
		}

		// Flush counters
		if scpFileCount > 0 {
			result = append(result, "scp: "+strconv.Itoa(scpFileCount)+" files transferred")
			scpFileCount = 0
		}
		if rsyncProgressCount > 0 {
			result = append(result, "("+strconv.Itoa(rsyncProgressCount)+" progress updates)")
			rsyncProgressCount = 0
		}

		// Keep rsync summary stats
		if rsyncStats.MatchString(trimmed) {
			result = append(result, line)
			continue
		}

		// Keep everything else (actual command output, errors)
		result = append(result, line)
	}

	// Flush trailing
	if scpFileCount > 0 {
		result = append(result, "scp: "+strconv.Itoa(scpFileCount)+" files transferred")
	}
	if rsyncProgressCount > 0 {
		result = append(result, "("+strconv.Itoa(rsyncProgressCount)+" progress updates)")
	}

	return strings.Join(result, "\n")
}
