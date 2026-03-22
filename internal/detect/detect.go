package detect

import (
	"path/filepath"
	"strings"

	"github.com/antikkorps/GoTK/internal/filter"
)

// CmdType represents a category of command for specialized filtering.
type CmdType int

const (
	CmdGeneric CmdType = iota
	CmdGrep
	CmdFind
	CmdGit
	CmdGoTool
	CmdLs
	CmdDocker
	CmdNpm
	CmdCargo
	CmdMake
	CmdCurl
	CmdPython
	CmdTree
	CmdTerraform
	CmdKubectl
	CmdJq
	CmdTar
	CmdSSH
)

// String returns the name of the command type.
func (c CmdType) String() string {
	switch c {
	case CmdGrep:
		return "grep"
	case CmdFind:
		return "find"
	case CmdGit:
		return "git"
	case CmdGoTool:
		return "go"
	case CmdLs:
		return "ls"
	case CmdDocker:
		return "docker"
	case CmdNpm:
		return "npm"
	case CmdCargo:
		return "cargo"
	case CmdMake:
		return "make"
	case CmdCurl:
		return "curl"
	case CmdPython:
		return "python"
	case CmdTree:
		return "tree"
	case CmdTerraform:
		return "terraform"
	case CmdKubectl:
		return "kubectl"
	case CmdJq:
		return "jq"
	case CmdTar:
		return "tar"
	case CmdSSH:
		return "ssh"
	default:
		return "generic"
	}
}

// Identify detects the command type from the binary name.
func Identify(command string) CmdType {
	base := filepath.Base(command)
	base = strings.TrimSuffix(base, ".exe")

	switch {
	case base == "grep" || base == "rg" || base == "ag" || base == "ack":
		return CmdGrep
	case base == "find" || base == "fd":
		return CmdFind
	case base == "git" || base == "gh":
		return CmdGit
	case base == "go":
		return CmdGoTool
	case base == "ls" || base == "exa" || base == "eza" || base == "lsd":
		return CmdLs
	case base == "docker" || base == "docker-compose" || base == "podman":
		return CmdDocker
	case base == "npm" || base == "yarn" || base == "pnpm" || base == "npx" || base == "bun":
		return CmdNpm
	case base == "cargo" || base == "rustc":
		return CmdCargo
	case base == "make" || base == "cmake" || base == "ninja":
		return CmdMake
	case base == "curl" || base == "wget" || base == "http" || base == "httpie":
		return CmdCurl
	case base == "python" || base == "python3" || base == "python2" || base == "pip" || base == "pip3":
		return CmdPython
	case base == "tree":
		return CmdTree
	case base == "terraform" || base == "tofu" || base == "tf":
		return CmdTerraform
	case base == "kubectl" || base == "helm" || base == "k9s" || base == "oc":
		return CmdKubectl
	case base == "jq" || base == "yq" || base == "gojq":
		return CmdJq
	case base == "tar" || base == "zip" || base == "unzip" || base == "gzip" || base == "7z":
		return CmdTar
	case base == "ssh" || base == "scp" || base == "sftp" || base == "rsync":
		return CmdSSH
	default:
		return CmdGeneric
	}
}

// FiltersFor returns command-specific filters for the given command type.
func FiltersFor(cmdType CmdType) []filter.FilterFunc {
	switch cmdType {
	case CmdGrep:
		return []filter.FilterFunc{filter.CompressPaths, compressGrepOutput}
	case CmdFind:
		return []filter.FilterFunc{filter.CompressPaths, compressFindOutput}
	case CmdGit:
		return []filter.FilterFunc{compressGitOutput}
	case CmdGoTool:
		return []filter.FilterFunc{filter.CompressPaths, compressGoOutput}
	case CmdLs:
		return []filter.FilterFunc{compressLsOutput}
	case CmdDocker:
		return []filter.FilterFunc{compressDockerOutput}
	case CmdNpm:
		return []filter.FilterFunc{compressNpmOutput}
	case CmdCargo:
		return []filter.FilterFunc{compressCargoOutput}
	case CmdMake:
		return []filter.FilterFunc{compressMakeOutput}
	case CmdCurl:
		return []filter.FilterFunc{compressCurlOutput}
	case CmdPython:
		return []filter.FilterFunc{compressPythonOutput}
	case CmdTree:
		return []filter.FilterFunc{compressTreeOutput}
	case CmdTerraform:
		return []filter.FilterFunc{compressTerraformOutput}
	case CmdKubectl:
		return []filter.FilterFunc{compressKubectlOutput}
	case CmdJq:
		return []filter.FilterFunc{compressJqOutput}
	case CmdTar:
		return []filter.FilterFunc{compressTarOutput}
	case CmdSSH:
		return []filter.FilterFunc{compressSSHOutput}
	default:
		return []filter.FilterFunc{filter.CompressPaths}
	}
}

// compressGrepOutput groups results by file and strips redundant prefixes.
func compressGrepOutput(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	lastFile := ""

	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Detect file:line:content or file:content pattern
		file, rest, ok := splitGrepLine(line)
		if !ok {
			result = append(result, line)
			lastFile = ""
			continue
		}

		if file != lastFile {
			// New file — emit header
			if lastFile != "" {
				result = append(result, "") // blank line between file groups
			}
			result = append(result, ">> "+file)
			lastFile = file
		}

		result = append(result, "  "+rest)
	}

	return strings.Join(result, "\n")
}

// splitGrepLine splits "file:linenum:content" into (file, rest).
// Line numbers are PRESERVED because they are essential for LLM code navigation.
func splitGrepLine(line string) (file, rest string, ok bool) {
	idx := strings.Index(line, ":")
	if idx <= 0 {
		return "", "", false
	}

	file = line[:idx]
	remaining := line[idx+1:]

	// Skip if file part looks like it has spaces (probably not a grep line)
	if strings.Contains(file, " ") {
		return "", "", false
	}

	// A valid grep file path should contain a dot (extension) or slash (path separator).
	// Plain words like ERROR, FAIL, panic, Warning are not file paths.
	if !strings.Contains(file, ".") && !strings.Contains(file, "/") {
		return "", "", false
	}

	// Keep linenum:content as-is — line numbers are semantically important
	return file, remaining, true
}

// compressFindOutput factorizes common path prefix.
func compressFindOutput(input string) string {
	lines := strings.Split(input, "\n")

	var paths []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			paths = append(paths, l)
		}
	}

	if len(paths) < 2 {
		return input
	}

	// Find common directory prefix
	prefix := filepath.Dir(paths[0])
	for _, p := range paths[1:] {
		for !strings.HasPrefix(p, prefix+"/") && prefix != "." && prefix != "/" && prefix != "" {
			prefix = filepath.Dir(prefix)
		}
	}

	// Only compress if prefix saves meaningful tokens
	if len(prefix) < 3 || prefix == "." || prefix == "/" {
		return strings.Join(paths, "\n") + "\n"
	}

	// Group by immediate subdirectory
	groups := map[string][]string{}
	var order []string

	for _, p := range paths {
		rel := strings.TrimPrefix(p, prefix+"/")
		dir := strings.SplitN(rel, "/", 2)[0]
		if _, exists := groups[dir]; !exists {
			order = append(order, dir)
		}
		groups[dir] = append(groups[dir], rel)
	}

	var result []string
	result = append(result, "[base: "+prefix+"/]")
	for _, dir := range order {
		for _, f := range groups[dir] {
			result = append(result, f)
		}
	}

	return strings.Join(result, "\n") + "\n"
}

// compressGitOutput cleans up verbose git output.
// Only removes truly redundant metadata (index hashes, mode changes).
// Preserves Author, Date, commit messages, and all diff content.
func compressGitOutput(input string) string {
	lines := strings.Split(input, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip only truly redundant diff metadata
		if strings.HasPrefix(trimmed, "index ") && strings.Contains(trimmed, "..") {
			continue // index abc1234..def5678 100644 — hash range, not useful
		}
		if strings.HasPrefix(trimmed, "old mode") || strings.HasPrefix(trimmed, "new mode") {
			continue // mode changes are rarely relevant
		}
		if strings.HasPrefix(trimmed, "similarity index") {
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

// compressGoOutput cleans up go test / go build output.
// Preserves all FAIL lines, test names, and build error context.
// Only compresses consecutive passing package lines into a summary.
func compressGoOutput(input string) string {
	lines := strings.Split(input, "\n")
	var result []string
	var passedPkgs []string

	for i, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Collect consecutive "ok" lines
		if strings.HasPrefix(trimmed, "ok") {
			fields := strings.Fields(trimmed)
			if len(fields) >= 2 {
				passedPkgs = append(passedPkgs, fields[1])
			}
			continue
		}

		// Flush pass summary before non-ok line
		if len(passedPkgs) > 0 {
			result = append(result, "ok "+itoa(len(passedPkgs))+" packages: "+strings.Join(passedPkgs, ", "))
			passedPkgs = nil
		}

		// For go build errors, look ahead to capture the context
		// Error lines look like: "./file.go:10:5: error message"
		// Package header looks like: "# package/path"
		if strings.HasPrefix(trimmed, "#") && i+1 < len(lines) {
			// Package header — check if next lines are errors
			// Keep the header, it provides essential context
			result = append(result, line)
			continue
		}

		result = append(result, line)
	}

	if len(passedPkgs) > 0 {
		result = append(result, "ok "+itoa(len(passedPkgs))+" packages: "+strings.Join(passedPkgs, ", "))
	}

	return strings.Join(result, "\n")
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	digits := []byte{}
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// compressLsOutput strips verbose metadata from ls -la style output.
func compressLsOutput(input string) string {
	lines := strings.Split(input, "\n")
	var result []string

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Skip "total NNN" line
		if strings.HasPrefix(trimmed, "total ") {
			continue
		}

		// For long-format lines, extract just permissions + name
		fields := strings.Fields(trimmed)
		if len(fields) >= 9 && isPermString(fields[0]) {
			// permissions size name (skip user, group, date, etc.)
			name := strings.Join(fields[8:], " ")
			size := fields[4]
			perm := fields[0]
			result = append(result, perm+" "+size+" "+name)
			continue
		}

		result = append(result, line)
	}

	return strings.Join(result, "\n")
}

func isPermString(s string) bool {
	if len(s) < 10 {
		return false
	}
	first := s[0]
	return first == '-' || first == 'd' || first == 'l' || first == 'c' || first == 'b'
}
