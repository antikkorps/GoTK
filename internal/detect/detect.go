package detect

import (
	"path"
	"path/filepath"
	"strconv"
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
	CmdNode
)

// cmdEntry defines a command type's name, binary aliases, and filters.
type cmdEntry struct {
	name     string
	binaries []string
	filters  func() []filter.FilterFunc
}

// registry maps CmdType to its registration entry.
// Filters are returned via a func to avoid init-order issues with package-level functions.
var registry = map[CmdType]cmdEntry{
	CmdGrep:   {name: "grep", binaries: []string{"grep", "rg", "ag", "ack"}, filters: func() []filter.FilterFunc { return []filter.FilterFunc{filter.CompressPaths, compressGrepOutput} }},
	CmdFind:   {name: "find", binaries: []string{"find", "fd"}, filters: func() []filter.FilterFunc { return []filter.FilterFunc{filter.CompressPaths, compressFindOutput} }},
	CmdGit:    {name: "git", binaries: []string{"git", "gh"}, filters: func() []filter.FilterFunc { return []filter.FilterFunc{compressGitOutput} }},
	CmdGoTool: {name: "go", binaries: []string{"go"}, filters: func() []filter.FilterFunc { return []filter.FilterFunc{filter.CompressPaths, compressGoOutput} }},
	CmdLs:     {name: "ls", binaries: []string{"ls", "exa", "eza", "lsd"}, filters: func() []filter.FilterFunc { return []filter.FilterFunc{compressLsOutput} }},
	CmdDocker: {name: "docker", binaries: []string{"docker", "docker-compose", "podman"}, filters: func() []filter.FilterFunc { return []filter.FilterFunc{compressDockerOutput} }},
	CmdNpm: {name: "npm", binaries: []string{"npm", "yarn", "pnpm", "bun"}, filters: func() []filter.FilterFunc {
		return []filter.FilterFunc{stripJestConsoleBlocks, compressNpmOutput, compressNodeOutput}
	}},
	CmdNode:      {name: "node", binaries: []string{"node", "npx", "tsx", "ts-node", "deno"}, filters: func() []filter.FilterFunc { return []filter.FilterFunc{stripJestConsoleBlocks, compressNodeOutput} }},
	CmdCargo:     {name: "cargo", binaries: []string{"cargo", "rustc"}, filters: func() []filter.FilterFunc { return []filter.FilterFunc{compressCargoOutput} }},
	CmdMake:      {name: "make", binaries: []string{"make", "cmake", "ninja"}, filters: func() []filter.FilterFunc { return []filter.FilterFunc{compressMakeOutput} }},
	CmdCurl:      {name: "curl", binaries: []string{"curl", "wget", "http", "httpie"}, filters: func() []filter.FilterFunc { return []filter.FilterFunc{compressCurlOutput} }},
	CmdPython:    {name: "python", binaries: []string{"python", "python3", "python2", "pip", "pip3"}, filters: func() []filter.FilterFunc { return []filter.FilterFunc{compressPythonOutput} }},
	CmdTree:      {name: "tree", binaries: []string{"tree"}, filters: func() []filter.FilterFunc { return []filter.FilterFunc{compressTreeOutput} }},
	CmdTerraform: {name: "terraform", binaries: []string{"terraform", "tofu", "tf"}, filters: func() []filter.FilterFunc { return []filter.FilterFunc{compressTerraformOutput} }},
	CmdKubectl:   {name: "kubectl", binaries: []string{"kubectl", "helm", "k9s", "oc"}, filters: func() []filter.FilterFunc { return []filter.FilterFunc{compressKubectlOutput} }},
	CmdJq:        {name: "jq", binaries: []string{"jq", "yq", "gojq"}, filters: func() []filter.FilterFunc { return []filter.FilterFunc{compressJqOutput} }},
	CmdTar:       {name: "tar", binaries: []string{"tar", "zip", "unzip", "gzip", "7z"}, filters: func() []filter.FilterFunc { return []filter.FilterFunc{compressTarOutput} }},
	CmdSSH:       {name: "ssh", binaries: []string{"ssh", "scp", "sftp", "rsync"}, filters: func() []filter.FilterFunc { return []filter.FilterFunc{compressSSHOutput} }},
}

// binaryIndex is a reverse lookup from binary name to CmdType, built at init.
var binaryIndex map[string]CmdType

func init() {
	binaryIndex = make(map[string]CmdType, 64)
	for cmdType, entry := range registry {
		for _, bin := range entry.binaries {
			binaryIndex[bin] = cmdType
		}
	}
}

// String returns the name of the command type.
func (c CmdType) String() string {
	if entry, ok := registry[c]; ok {
		return entry.name
	}
	return "generic"
}

// Identify detects the command type from the binary name.
func Identify(command string) CmdType {
	base := filepath.Base(command)
	base = strings.TrimSuffix(base, ".exe")

	if cmdType, ok := binaryIndex[base]; ok {
		return cmdType
	}
	return CmdGeneric
}

// FiltersFor returns command-specific filters for the given command type.
func FiltersFor(cmdType CmdType) []filter.FilterFunc {
	if entry, ok := registry[cmdType]; ok {
		return entry.filters()
	}
	return []filter.FilterFunc{filter.CompressPaths}
}

// RegisteredTypes returns all registered command types (for introspection).
func RegisteredTypes() []CmdType {
	types := make([]CmdType, 0, len(registry))
	for t := range registry {
		types = append(types, t)
	}
	return types
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

// compressFindOutput factorizes common path prefix. Operates on
// forward-slash paths; Windows backslash inputs are normalized at the
// boundary so the rest of the function can stay platform-agnostic.
func compressFindOutput(input string) string {
	lines := strings.Split(input, "\n")

	var paths []string
	for _, l := range lines {
		if strings.TrimSpace(l) != "" {
			// Normalize separators so prefix stripping below works on
			// Windows-style paths and the output stays consistent across
			// platforms.
			paths = append(paths, filepath.ToSlash(l))
		}
	}

	if len(paths) < 2 {
		return input
	}

	// Find common directory prefix using slash-only path operations
	// (path.Dir, not filepath.Dir) so the loop's "did the parent change?"
	// invariant holds on Windows where filepath.Dir hits a fixed point at
	// drive roots like "D:\\".
	prefix := path.Dir(paths[0])
	for _, p := range paths[1:] {
		for !strings.HasPrefix(p, prefix+"/") && prefix != "." && prefix != "/" && prefix != "" {
			next := path.Dir(prefix)
			if next == prefix {
				prefix = "."
				break
			}
			prefix = next
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
		result = append(result, groups[dir]...)
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
			result = append(result, "ok "+strconv.Itoa(len(passedPkgs))+" packages: "+strings.Join(passedPkgs, ", "))
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
		result = append(result, "ok "+strconv.Itoa(len(passedPkgs))+" packages: "+strings.Join(passedPkgs, ", "))
	}

	return strings.Join(result, "\n")
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
