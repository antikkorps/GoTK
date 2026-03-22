package detect

import (
	"regexp"
	"strings"
)

var (
	// curl progress bar: "  % Total    % Received" or download stats lines
	curlProgressHeader = regexp.MustCompile(`^\s*%\s+Total\s+%\s+Received`)
	curlProgressLine   = regexp.MustCompile(`^\s*\d+\s+\d+.*\d+[kMG]?\s+\d+`)
	// curl/wget verbose headers: "> Header:" or "< Header:" or "* info"
	curlVerboseSend = regexp.MustCompile(`^>\s+\S`)
	curlVerboseRecv = regexp.MustCompile(`^<\s+\S`)
	curlVerboseInfo = regexp.MustCompile(`^\*\s+`)
	// wget progress: "2024-03-22 10:00:00 (1.5 MB/s) - saved" or dots/bar
	wgetProgressDots = regexp.MustCompile(`^\.{5,}`)
	wgetProgressBar  = regexp.MustCompile(`\d+%\[=*>?\s*\]`)
	// HTTP status line: "HTTP/1.1 200 OK" or "HTTP/2 404"
	httpStatusLine = regexp.MustCompile(`^<?\s*HTTP/[\d.]+\s+\d{3}`)
)

// compressCurlOutput removes progress bars and compresses verbose headers.
// Keeps: response body, HTTP status, error messages, final download summary.
func compressCurlOutput(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	var recvHeaders []string
	inProgressTable := false
	sendHeaderCount := 0

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		if trimmed == "" {
			// Flush pending headers before blank line (end of headers section)
			if len(recvHeaders) > 0 {
				result = append(result, compressHeaders(recvHeaders)...)
				recvHeaders = nil
			}
			if sendHeaderCount > 0 {
				result = append(result, "> ("+itoa(sendHeaderCount)+" request headers)")
				sendHeaderCount = 0
			}
			result = append(result, line)
			continue
		}

		// Skip curl progress table header and data lines
		if curlProgressHeader.MatchString(trimmed) {
			inProgressTable = true
			continue
		}
		if inProgressTable && curlProgressLine.MatchString(trimmed) {
			continue
		}
		inProgressTable = false

		// Skip wget progress dots and bars
		if wgetProgressDots.MatchString(trimmed) || wgetProgressBar.MatchString(trimmed) {
			continue
		}

		// Skip curl verbose info lines (connection details, TLS handshake, etc.)
		// but keep error-related info
		if curlVerboseInfo.MatchString(trimmed) {
			lower := strings.ToLower(trimmed)
			if strings.Contains(lower, "error") || strings.Contains(lower, "fail") ||
				strings.Contains(lower, "refused") || strings.Contains(lower, "timeout") {
				result = append(result, line)
			}
			continue
		}

		// Bare ">" or "<" — flush pending and skip
		if trimmed == ">" {
			if sendHeaderCount > 0 {
				result = append(result, "> ("+itoa(sendHeaderCount)+" request headers)")
				sendHeaderCount = 0
			}
			continue
		}
		if trimmed == "<" {
			if len(recvHeaders) > 0 {
				result = append(result, compressHeaders(recvHeaders)...)
				recvHeaders = nil
			}
			continue
		}

		// Compress outgoing request headers: just count them
		if curlVerboseSend.MatchString(trimmed) {
			sendHeaderCount++
			continue
		}

		// Collect incoming response headers for compression
		if curlVerboseRecv.MatchString(trimmed) {
			recvHeaders = append(recvHeaders, trimmed)
			continue
		}

		// Keep HTTP status lines
		if httpStatusLine.MatchString(trimmed) {
			result = append(result, line)
			continue
		}

		// Keep everything else (response body, errors, wget final summary)
		result = append(result, line)
	}

	// Flush trailing
	if len(recvHeaders) > 0 {
		result = append(result, compressHeaders(recvHeaders)...)
	}
	if sendHeaderCount > 0 {
		result = append(result, "> ("+itoa(sendHeaderCount)+" request headers)")
	}

	return strings.Join(result, "\n")
}

// compressHeaders keeps important response headers and summarizes the rest.
func compressHeaders(headers []string) []string {
	var kept []string
	skipped := 0

	importantPrefixes := []string{
		"< HTTP/", "< content-type", "< location", "< www-authenticate",
		"< x-error", "< retry-after", "< status",
	}

	for _, h := range headers {
		lower := strings.ToLower(h)
		important := false
		for _, prefix := range importantPrefixes {
			if strings.HasPrefix(lower, prefix) {
				important = true
				break
			}
		}
		if important {
			kept = append(kept, h)
		} else {
			skipped++
		}
	}

	if skipped > 0 {
		kept = append(kept, "< ("+itoa(skipped)+" other headers)")
	}
	return kept
}
