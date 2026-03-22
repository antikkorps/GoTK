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
	// Docker build output: "Step N/M :" or "STEP N:"
	autoDockerBuildPattern = regexp.MustCompile(`^(Step|STEP) \d+(/\d+)?\s*:`)
	// npm output: "npm warn", "added N packages"
	autoNpmPattern = regexp.MustCompile(`^npm (warn|ERR!|http)|^added \d+ packages`)
	// Cargo output: "Compiling ", "Downloading ", "Finished"
	autoCargoPattern = regexp.MustCompile(`^\s*(Compiling|Downloading|Finished)\s+`)
	// Make output: "make[N]:" or "make:"
	autoMakePattern = regexp.MustCompile(`^make(\[\d+\])?:`)
	// curl verbose output: "< HTTP/", "> Host:", "* Connected to"
	autoCurlPattern = regexp.MustCompile(`^[<>*]\s+\S`)
	// Python traceback
	autoPythonPattern = regexp.MustCompile(`^(Traceback \(most recent call last\):|^\s+File ".+", line \d+)`)
	// terraform output: resource refresh/plan lines
	autoTerraformPattern = regexp.MustCompile(`^(\S+\.\S+: (Refreshing|Creating|Modifying|Destroying|Reading)|Plan: \d+ to add)`)
	// kubectl output: resource lines with NAME/READY/STATUS headers or YAML with apiVersion/kind
	autoKubectlPattern = regexp.MustCompile(`^(NAME\s+READY\s+STATUS|apiVersion:\s|kind:\s|metadata:\s)`)
)

// AutoDetect analyzes output content to guess the source command type.
// Used in pipe mode where we don't know the command.
func AutoDetect(output string) CmdType {
	lines := strings.Split(output, "\n")

	// Sample first 50 non-empty lines for better pattern detection
	var sample []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			sample = append(sample, l)
		}
		if len(sample) >= 50 {
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
		if autoDockerBuildPattern.MatchString(line) {
			return CmdDocker
		}
		if autoPythonPattern.MatchString(line) {
			return CmdPython
		}
		if autoTerraformPattern.MatchString(line) {
			return CmdTerraform
		}
		if autoKubectlPattern.MatchString(line) {
			return CmdKubectl
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
		case autoNpmPattern.MatchString(line):
			scores[CmdNpm]++
		case autoCargoPattern.MatchString(line):
			scores[CmdCargo]++
		case autoMakePattern.MatchString(line):
			scores[CmdMake]++
		case autoCurlPattern.MatchString(line):
			scores[CmdCurl]++
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
