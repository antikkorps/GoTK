package detect

import (
	"regexp"
	"strings"
)

// Patterns for auto-detecting command output format in pipe mode.
var (
	// grep-style: file:line:content or file:content
	grepPattern = regexp.MustCompile(`^[^\s:]+:\d+:`)
	// find-style: lines that are all file paths
	findPathPattern = regexp.MustCompile(`^\.?/[^\s]+$`)
	// git log: commit hashes
	gitLogPattern = regexp.MustCompile(`^commit [0-9a-f]{7,40}$`)
	// git diff: diff --git a/... b/...
	gitDiffPattern = regexp.MustCompile(`^diff --git a/`)
	// go test: --- FAIL or ok or PASS
	goTestPattern = regexp.MustCompile(`^(ok|FAIL|---\s+(FAIL|PASS))\s+`)
	// ls -l style: permission bits at start
	lsPattern = regexp.MustCompile(`^[drwxlstST-]{10}`)
)

// AutoDetect analyzes output content to guess the source command type.
// Used in pipe mode where we don't know the command.
func AutoDetect(output string) CmdType {
	lines := strings.Split(output, "\n")

	// Sample first 20 non-empty lines
	var sample []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			sample = append(sample, l)
		}
		if len(sample) >= 20 {
			break
		}
	}

	if len(sample) == 0 {
		return CmdGeneric
	}

	// High-confidence patterns: a single match is enough (very distinctive)
	for _, line := range sample {
		if gitLogPattern.MatchString(line) || gitDiffPattern.MatchString(line) {
			return CmdGit
		}
	}

	// Count matches for frequency-based patterns
	scores := map[CmdType]int{}

	for _, line := range sample {
		switch {
		case grepPattern.MatchString(line):
			scores[CmdGrep]++
		case goTestPattern.MatchString(line):
			scores[CmdGoTool]++
		case lsPattern.MatchString(line):
			scores[CmdLs]++
		case findPathPattern.MatchString(line):
			scores[CmdFind]++
		}
	}

	// Return the type with highest score (if it matches >40% of sampled lines)
	bestType := CmdGeneric
	bestScore := 0
	threshold := len(sample) * 40 / 100
	if threshold < 2 {
		threshold = 2
	}

	for cmdType, score := range scores {
		if score > bestScore && score >= threshold {
			bestType = cmdType
			bestScore = score
		}
	}

	return bestType
}
