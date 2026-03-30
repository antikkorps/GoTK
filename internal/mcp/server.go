package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/antikkorps/GoTK/internal/cache"
	"github.com/antikkorps/GoTK/internal/classify"
	"github.com/antikkorps/GoTK/internal/config"
	gotkctx "github.com/antikkorps/GoTK/internal/ctx"
	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/exec"
	"github.com/antikkorps/GoTK/internal/measure"
	"github.com/antikkorps/GoTK/internal/proxy"
)

// Version is set by the CLI entrypoint before calling Serve.
var Version = "dev"

// dangerousBaseCommands is a denylist of base command names that are always blocked.
var dangerousBaseCommands = map[string]bool{
	"mkfs":     true,
	"shutdown": true,
	"reboot":   true,
	"halt":     true,
	"wipefs":   true,
	"format":   true,
	"truncate": true,
	"iptables": true,
	"ufw":      true,
}

// dangerousPipePatterns detects remote code execution patterns like curl|bash.
var dangerousPipePatterns = []string{
	"curl|bash", "curl |bash", "curl | bash",
	"curl|sh", "curl |sh", "curl | sh",
	"wget|bash", "wget |bash", "wget | bash",
	"wget|sh", "wget |sh", "wget | sh",
}

// dangerousArgPattern describes a command with specific dangerous argument combinations.
type dangerousArgPattern struct {
	cmd      string   // base command name
	flags    []string // required flags (checked as substrings for combined flags like -rf)
	exactArg string   // an argument that must appear as an exact field match (e.g., "/" or "/*")
}

// dangerousPatterns is a list of command + argument patterns that need special matching.
var dangerousPatterns = []dangerousArgPattern{
	{cmd: "rm", flags: []string{"-rf"}, exactArg: "/"},
	{cmd: "rm", flags: []string{"-rf"}, exactArg: "/*"},
	{cmd: "rm", flags: []string{"-fr"}, exactArg: "/"},
	{cmd: "rm", flags: []string{"-fr"}, exactArg: "/*"},
	{cmd: "dd", flags: []string{"if="}},
	{cmd: "chmod", flags: []string{"-R", "777"}, exactArg: "/"},
	{cmd: "chown", flags: []string{"-R"}},
	{cmd: "mv", exactArg: "/"},
	{cmd: "mv", exactArg: "/*"},
	{cmd: "init", exactArg: "0"},
	{cmd: "init", exactArg: "6"},
}

// readOnlyCommands is the allowlist of commands permitted in sandbox mode.
// Only commands that cannot modify files or system state are allowed.
var readOnlyCommands = map[string]bool{
	// File viewing
	"cat": true, "head": true, "tail": true, "less": true, "more": true, "bat": true,
	// Search
	"grep": true, "rg": true, "ag": true, "find": true, "fd": true, "locate": true,
	// Listing
	"ls": true, "tree": true, "exa": true, "eza": true, "dir": true,
	// File info
	"file": true, "stat": true, "wc": true, "du": true, "df": true,
	"which": true, "whereis": true, "type": true, "realpath": true, "basename": true, "dirname": true,
	// Text processing (read-only)
	"sort": true, "uniq": true, "cut": true, "tr": true, "column": true,
	"jq": true, "yq": true, "xargs": true, "diff": true, "comm": true,
	// Version control (read)
	"git": true, "hg": true, "svn": true,
	// Build tools (read/compile)
	"go": true, "make": true, "cargo": true, "npm": true, "yarn": true, "pnpm": true,
	"python": true, "python3": true, "node": true, "ruby": true, "perl": true,
	// System info
	"uname": true, "hostname": true, "whoami": true, "id": true,
	"date": true, "uptime": true, "env": true, "printenv": true,
	"echo": true, "printf": true, "test": true,
	// Network (read)
	"curl": true, "wget": true, "ping": true, "dig": true, "nslookup": true, "host": true,
	// Process info
	"ps": true, "pgrep": true, "lsof": true,
}

// sandboxWritePatterns detects output redirections and other write operations in commands.
var sandboxWritePatterns = []string{
	">>", // append redirect
	">",  // write redirect (checked after >> to avoid false match)
}

// validateSandbox checks whether a command is allowed in sandbox mode.
// Returns an error message if blocked, empty string if allowed.
func validateSandbox(command string) string {
	fields := strings.Fields(strings.TrimSpace(command))
	if len(fields) == 0 {
		return ""
	}

	// Check for output redirections
	for _, p := range sandboxWritePatterns {
		if strings.Contains(command, p) {
			return "sandbox: output redirection is not allowed"
		}
	}

	// Check each command in a pipeline
	parts := strings.Split(command, "|")
	for _, part := range parts {
		partFields := strings.Fields(strings.TrimSpace(part))
		if len(partFields) == 0 {
			continue
		}
		cmdName := extractCommand(partFields)
		if !readOnlyCommands[cmdName] {
			return fmt.Sprintf("sandbox: command %q is not in the read-only allowlist", cmdName)
		}
	}

	return ""
}

// validatePath checks that a path resolves to a location under the project root.
// It prevents directory traversal attacks (../../etc/passwd) and symlink escapes.
// Returns the cleaned absolute path or an error message.
func validatePath(requestedPath string) (string, string) {
	projectRoot, err := os.Getwd()
	if err != nil {
		return "", "cannot determine project root"
	}

	// Resolve to absolute path
	absPath := requestedPath
	if !filepath.IsAbs(absPath) {
		absPath = filepath.Join(projectRoot, absPath)
	}

	// Clean the path to resolve .. components
	absPath = filepath.Clean(absPath)

	// Resolve symlinks to get the real path
	resolved, err := filepath.EvalSymlinks(absPath)
	if err != nil {
		// File may not exist yet (for writes), check the parent
		parentResolved, parentErr := filepath.EvalSymlinks(filepath.Dir(absPath))
		if parentErr != nil {
			return "", fmt.Sprintf("path validation failed: %v", err)
		}
		resolved = filepath.Join(parentResolved, filepath.Base(absPath))
	}

	// Ensure the resolved path is under the project root
	resolvedRoot, err := filepath.EvalSymlinks(projectRoot)
	if err != nil {
		resolvedRoot = projectRoot
	}

	if !strings.HasPrefix(resolved, resolvedRoot+string(filepath.Separator)) && resolved != resolvedRoot {
		return "", fmt.Sprintf("path %q is outside project root", requestedPath)
	}

	return resolved, ""
}

// validateAuditLogPath checks that the audit log path is not a symlink
// and that its parent directory is not world-writable.
func validateAuditLogPath(path string) string {
	absPath := path
	if !filepath.IsAbs(absPath) {
		cwd, err := os.Getwd()
		if err != nil {
			return "cannot determine working directory"
		}
		absPath = filepath.Join(cwd, absPath)
	}
	absPath = filepath.Clean(absPath)

	// Check if path itself is a symlink (if it exists)
	if info, err := os.Lstat(absPath); err == nil {
		if info.Mode()&os.ModeSymlink != 0 {
			return fmt.Sprintf("audit log path %q is a symlink", path)
		}
	}

	// Check parent directory is not world-writable
	parentDir := filepath.Dir(absPath)
	if parentInfo, err := os.Stat(parentDir); err == nil {
		if parentInfo.Mode().Perm()&0002 != 0 {
			return fmt.Sprintf("audit log parent directory %q is world-writable", parentDir)
		}
	}

	return ""
}

// maxInputSize is the maximum allowed input size for MCP requests (10MB).
const maxInputSize = 10 * 1024 * 1024

// defaultMCPTimeout is the default timeout for MCP command execution (5 minutes).
// Higher than CLI default (30s) because MCP is used for heavier operations like test suites.
const defaultMCPTimeout = 300 * time.Second

// extractCommand returns the base command name from a command line,
// skipping common prefixes like sudo, env, nice, etc.
func extractCommand(fields []string) string {
	skip := map[string]bool{"sudo": true, "env": true, "nice": true, "nohup": true}
	for _, f := range fields {
		base := strings.ToLower(filepath.Base(f))
		// Skip env var assignments (FOO=bar)
		if strings.Contains(f, "=") {
			continue
		}
		if skip[base] {
			continue
		}
		return base
	}
	if len(fields) > 0 {
		return strings.ToLower(filepath.Base(fields[0]))
	}
	return ""
}

// validateCommand checks whether a command is on the denylist of dangerous commands.
// Returns an error message if the command is blocked, empty string if allowed.
func validateCommand(command string) string {
	normalized := strings.TrimSpace(command)
	fields := strings.Fields(normalized)
	if len(fields) == 0 {
		return ""
	}

	// Extract the actual command name, skipping sudo/env prefixes
	cmdName := extractCommand(fields)

	// Check if the base command itself is dangerous (exact match or mkfs.* prefix)
	if dangerousBaseCommands[cmdName] || strings.HasPrefix(cmdName, "mkfs.") {
		return fmt.Sprintf("command blocked: dangerous command %q", cmdName)
	}

	// Check dangerous pipe patterns (remote code execution)
	normalizedLower := strings.ToLower(normalized)
	for _, pattern := range dangerousPipePatterns {
		if strings.Contains(normalizedLower, pattern) {
			return "command blocked: remote code execution pattern detected"
		}
	}

	// Check fork bomb pattern
	if strings.Contains(normalized, ":(){ :|:&};:") || strings.Contains(normalized, ":(){:|:&};:") {
		return "command blocked: fork bomb detected"
	}

	// Check redirect to device
	if strings.Contains(normalized, "> /dev/sda") {
		return "command blocked: destructive device write"
	}

	// Check dangerous command + argument patterns using the actual command name
	for _, pattern := range dangerousPatterns {
		if cmdName != pattern.cmd {
			continue
		}

		rest := strings.Join(fields[1:], " ")

		// Check that all required flags are present (substring match for combined flags)
		flagsMatch := true
		for _, flag := range pattern.flags {
			if !strings.Contains(rest, flag) {
				flagsMatch = false
				break
			}
		}
		if !flagsMatch {
			continue
		}

		// If an exact argument is required, check that it appears as a standalone field
		if pattern.exactArg != "" {
			found := false
			for _, f := range fields[1:] {
				if f == pattern.exactArg {
					found = true
					break
				}
			}
			if !found {
				continue
			}
		}

		return fmt.Sprintf("command blocked: dangerous pattern %q", cmdName)
	}

	return ""
}

// JSON-RPC request/response types

type jsonRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type jsonRPCResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{}     `json:"result,omitempty"`
	Error   *rpcError       `json:"error,omitempty"`
}

type rpcError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// MCP-specific types

type initializeResult struct {
	ProtocolVersion string     `json:"protocolVersion"`
	Capabilities    capObject  `json:"capabilities"`
	ServerInfo      serverInfo `json:"serverInfo"`
}

type capObject struct {
	Tools map[string]interface{} `json:"tools"`
}

type serverInfo struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type toolsListResult struct {
	Tools []toolDef `json:"tools"`
}

type toolDef struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	InputSchema inputSchema `json:"inputSchema"`
}

type inputSchema struct {
	Type       string             `json:"type"`
	Properties map[string]propDef `json:"properties"`
	Required   []string           `json:"required"`
}

type propDef struct {
	Type        string `json:"type"`
	Description string `json:"description"`
}

type toolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments"`
}

type execArgs struct {
	Command    string `json:"command"`
	MaxLines   int    `json:"max_lines"`
	NoTruncate bool   `json:"no_truncate"`
	Aggressive bool   `json:"aggressive"`
	Timeout    int    `json:"timeout"`
}

type filterArgs struct {
	Input       string `json:"input"`
	CommandHint string `json:"command_hint"`
	MaxLines    int    `json:"max_lines"`
}

type readArgs struct {
	Path     string `json:"path"`
	MaxLines int    `json:"max_lines"`
	Offset   int    `json:"offset"`
	Limit    int    `json:"limit"`
}

type grepArgs struct {
	Pattern    string `json:"pattern"`
	Path       string `json:"path"`
	MaxLines   int    `json:"max_lines"`
	Recursive  bool   `json:"recursive"`
	IgnoreCase bool   `json:"ignore_case"`
	LineNumber bool   `json:"line_number"`
}

type ctxArgs struct {
	Pattern  string `json:"pattern"`
	Path     string `json:"path"`
	Mode     string `json:"mode"`
	Context  int    `json:"context"`
	FileType string `json:"file_type"`
	MaxFiles int    `json:"max_files"`
}

type toolCallResult struct {
	Content []contentBlock `json:"content"`
}

type contentBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// cacheMaxEntries is the default maximum number of cached filter results.
const cacheMaxEntries = 128

// buildConfigHash computes a fingerprint of config fields that affect filtering.
func buildConfigHash(cfg *config.Config) string {
	filtersStr := fmt.Sprintf("%v", cfg.Filters)
	rulesStr := fmt.Sprintf("%v|%v", cfg.Rules.AlwaysKeep, cfg.Rules.AlwaysRemove)
	return cache.ConfigHash(filtersStr, rulesStr, string(cfg.General.Mode), cfg.Security.RedactSecrets)
}

// mcpMeasureLogger is the measurement logger for MCP mode. Set by Serve() if enabled.
var mcpMeasureLogger *measure.Logger

// reReqTracker detects re-requests in MCP mode. Set by Serve().
var reReqTracker *ReRequestTracker

// Serve starts the MCP server, reading JSON-RPC from stdin and writing to stdout.
func Serve(cfg *config.Config) {
	// Initialize file-based audit log if configured
	if cfg.Security.AuditLog != "" {
		auditPath := cfg.Security.AuditLog

		// Validate audit log path: reject symlinks and world-writable parent dirs
		if auditPathErr := validateAuditLogPath(auditPath); auditPathErr != "" {
			fmt.Fprintf(os.Stderr, "[gotk-mcp] WARNING: audit log path rejected: %s\n", auditPathErr)
		} else {
			f, err := os.OpenFile(auditPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			if err != nil {
				fmt.Fprintf(os.Stderr, "[gotk-mcp] WARNING: cannot open audit log %q: %v\n", auditPath, err)
			} else {
				auditFile = f
				defer f.Close()
			}
		}
	}

	// Initialize measurement logger — always enabled in MCP mode
	{
		ml, err := measure.NewLogger(cfg.Measure.LogPath, measure.SessionID(), cfg.Measure.MaxLogSize)
		if err != nil {
			fmt.Fprintf(os.Stderr, "[gotk-mcp] WARNING: cannot init measure log: %v\n", err)
		} else {
			mcpMeasureLogger = ml
			defer ml.Close()
		}
	}

	limiter := newRateLimiter(cfg.Security.RateLimit, cfg.Security.RateBurst)
	fc := cache.New(cacheMaxEntries, buildConfigHash(cfg))
	reReqTracker = NewReRequestTracker(DefaultReRequestWindow)

	scanner := bufio.NewScanner(os.Stdin)
	// Allow large input lines (up to 10 MB)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}

		var req jsonRPCRequest
		if err := json.Unmarshal([]byte(line), &req); err != nil {
			logErr("failed to parse JSON-RPC request: %v", err)
			sendError(nil, -32700, "Parse error")
			continue
		}

		// Notifications have no ID and expect no response
		if req.ID == nil {
			handleNotification(req.Method)
			continue
		}

		handleRequest(cfg, limiter, fc, req)
	}

	if err := scanner.Err(); err != nil {
		logErr("stdin scanner error: %v", err)
	}
}

func handleNotification(method string) {
	switch method {
	case "notifications/initialized":
		// Acknowledged, no response needed
	default:
		logErr("unknown notification: %s", method)
	}
}

func handleRequest(cfg *config.Config, limiter *rateLimiter, fc *cache.Cache, req jsonRPCRequest) {
	switch req.Method {
	case "initialize":
		// Auto-detect LLM profile from client info if no profile is configured
		if cfg.Profile == config.ProfileNone {
			cfg.Profile = detectProfileFromInit(req.Params)
			if cfg.Profile != config.ProfileNone {
				cfg.ApplyProfile()
				cfg.ApplyMode()
				logErr("AUTO-PROFILE: detected %q from client info", cfg.Profile)
			}
		}
		result := initializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities:    capObject{Tools: map[string]interface{}{}},
			ServerInfo:      serverInfo{Name: "gotk", Version: Version},
		}
		sendResult(req.ID, result)

	case "ping":
		sendResult(req.ID, map[string]interface{}{})

	case "tools/list":
		sendResult(req.ID, buildToolsList())

	case "tools/call":
		if !limiter.Allow() {
			logErr("RATE LIMITED: tools/call rejected")
			auditLog("RATE_LIMIT", "tools/call rejected — rate limit exceeded")
			sendError(req.ID, -32000, "Rate limit exceeded: too many requests per minute")
			return
		}
		handleToolCall(cfg, fc, req)

	default:
		sendError(req.ID, -32601, "Method not found: "+req.Method)
	}
}

func buildToolsList() toolsListResult {
	return toolsListResult{
		Tools: []toolDef{
			{
				Name:        "gotk_exec",
				Description: "Execute a command and return cleaned output with noise removed. Reduces token usage by 51-96%. For large outputs (test suites, builds), use max_lines to cap output length rather than no_truncate.",
				InputSchema: inputSchema{
					Type: "object",
					Properties: map[string]propDef{
						"command":     {Type: "string", Description: "The command to execute (e.g., 'grep -rn func ./src/')"},
						"max_lines":   {Type: "integer", Description: "Maximum output lines (default: 50)"},
						"no_truncate": {Type: "boolean", Description: "Disable truncation. WARNING: may produce output exceeding token limits on large test suites or verbose commands. Prefer max_lines instead."},
						"aggressive":  {Type: "boolean", Description: "Aggressive filtering mode"},
						"timeout":     {Type: "integer", Description: "Command timeout in seconds (default: 300). Most commands complete well within this limit. Only override if you have a specific reason."},
					},
					Required: []string{"command"},
				},
			},
			{
				Name:        "gotk_filter",
				Description: "Filter raw text through GoTK's cleaning pipeline. Use when you already have command output to clean.",
				InputSchema: inputSchema{
					Type: "object",
					Properties: map[string]propDef{
						"input":        {Type: "string", Description: "Raw text to filter"},
						"command_hint": {Type: "string", Description: "Hint about source command (e.g., 'grep', 'git') for specialized filtering"},
						"max_lines":    {Type: "integer", Description: "Maximum output lines"},
					},
					Required: []string{"input"},
				},
			},
			{
				Name:        "gotk_read",
				Description: "Read a file with noise removal and smart truncation. More token-efficient than cat. Strips ANSI codes, redacts secrets, and applies head+tail truncation for large files.",
				InputSchema: inputSchema{
					Type: "object",
					Properties: map[string]propDef{
						"path":      {Type: "string", Description: "Absolute or relative path to the file to read"},
						"max_lines": {Type: "integer", Description: "Maximum output lines (default: 200). Uses head+tail truncation to preserve both beginning and end of file."},
						"offset":    {Type: "integer", Description: "Start reading from this line number (1-based). Useful for reading a specific section of a large file."},
						"limit":     {Type: "integer", Description: "Maximum number of lines to read from the file before filtering. 0 means read entire file."},
					},
					Required: []string{"path"},
				},
			},
			{
				Name:        "gotk_ctx",
				Description: "Search codebase with 5 output modes optimized for LLM consumption. Built-in exclusions (node_modules, .git, vendor, etc.), binary detection, and GoTK filtering. Modes: scan (default), detail, def, tree, summary.",
				InputSchema: inputSchema{
					Type: "object",
					Properties: map[string]propDef{
						"pattern":   {Type: "string", Description: "Search pattern (regex)"},
						"path":      {Type: "string", Description: "Root directory to search (default: current directory)"},
						"mode":      {Type: "string", Description: "Output mode: scan (default), detail, def, tree, summary"},
						"context":   {Type: "integer", Description: "Lines of context for detail mode (default: 3)"},
						"file_type": {Type: "string", Description: "File extension filter (e.g., 'go', 'py', 'ts')"},
						"max_files": {Type: "integer", Description: "Maximum number of file results (0 = unlimited)"},
					},
					Required: []string{"pattern"},
				},
			},
			{
				Name:        "gotk_grep",
				Description: "Search file contents with noise removal. Results are grouped by file with compressed paths. More token-efficient than grep. Automatically applies recursive search and line numbers.",
				InputSchema: inputSchema{
					Type: "object",
					Properties: map[string]propDef{
						"pattern":     {Type: "string", Description: "Search pattern (regular expression)"},
						"path":        {Type: "string", Description: "File or directory to search in (default: current directory)"},
						"max_lines":   {Type: "integer", Description: "Maximum output lines (default: 100)"},
						"recursive":   {Type: "boolean", Description: "Search recursively in subdirectories (default: true)"},
						"ignore_case": {Type: "boolean", Description: "Case-insensitive search (default: false)"},
						"line_number": {Type: "boolean", Description: "Show line numbers (default: true)"},
					},
					Required: []string{"pattern"},
				},
			},
		},
	}
}

func handleToolCall(cfg *config.Config, fc *cache.Cache, req jsonRPCRequest) {
	var params toolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		sendError(req.ID, -32602, "Invalid params: "+err.Error())
		return
	}

	switch params.Name {
	case "gotk_exec":
		handleExec(cfg, fc, req.ID, params.Arguments)
	case "gotk_filter":
		handleFilter(cfg, fc, req.ID, params.Arguments)
	case "gotk_ctx":
		handleCtx(cfg, req.ID, params.Arguments)
	case "gotk_read":
		handleRead(cfg, fc, req.ID, params.Arguments)
	case "gotk_grep":
		handleGrep(cfg, fc, req.ID, params.Arguments)
	default:
		sendError(req.ID, -32602, "Unknown tool: "+params.Name)
	}
}

func handleExec(cfg *config.Config, fc *cache.Cache, id json.RawMessage, rawArgs json.RawMessage) {
	// Input size validation
	if len(rawArgs) > maxInputSize {
		sendError(id, -32602, fmt.Sprintf("input too large: %d bytes exceeds %d byte limit", len(rawArgs), maxInputSize))
		return
	}

	var args execArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		sendError(id, -32602, "Invalid arguments: "+err.Error())
		return
	}

	if args.Command == "" {
		sendError(id, -32602, "command is required")
		return
	}

	// Command validation: check against denylist
	if reason := validateCommand(args.Command); reason != "" {
		logErr("BLOCKED command: %q - %s", args.Command, reason)
		auditLog("CMD_BLOCKED", fmt.Sprintf("%q — %s", args.Command, reason))
		sendError(id, -32602, reason)
		return
	}

	// Sandbox mode: only allow read-only commands
	if cfg.Security.SandboxMode {
		if reason := validateSandbox(args.Command); reason != "" {
			logErr("SANDBOX BLOCKED: %q - %s", args.Command, reason)
			auditLog("SANDBOX_BLOCKED", fmt.Sprintf("%q — %s", args.Command, reason))
			sendError(id, -32602, reason)
			return
		}
	}

	// Audit log: record all executed commands
	logErr("EXEC: %s", args.Command)

	// Parse the command to detect type
	parts := strings.Fields(args.Command)
	if len(parts) == 0 {
		sendError(id, -32602, "empty command")
		return
	}

	// Determine max lines: explicit arg > per-command config > global default
	maxLines := cfg.MaxLinesForCommand(parts[0])
	if args.MaxLines > 0 {
		maxLines = args.MaxLines
	}
	if args.NoTruncate {
		maxLines = 0
	}

	// Execute via shell with timeout
	shell := findShell()
	timeout := defaultMCPTimeout
	if cfg.Security.CommandTimeout > 0 {
		timeout = time.Duration(cfg.Security.CommandTimeout) * time.Second
	}
	if args.Timeout > 0 {
		timeout = time.Duration(args.Timeout) * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result, err := exec.RunWithTimeout(ctx, shell, "-c", args.Command)
	if err != nil {
		sendError(id, -32603, "execution failed: "+err.Error())
		return
	}

	raw := result.Stdout
	if result.Stderr != "" {
		if raw != "" {
			raw += "\n"
		}
		raw += result.Stderr
	}

	// Detect command type and build filter chain
	cmdType := detect.Identify(parts[0])
	if mapped, ok := cfg.Commands[parts[0]]; ok {
		cmdType = detect.Identify(mapped)
	}

	// Check cache for previously filtered identical output
	filterStart := time.Now()
	cacheKey := fc.Key(raw, int(cmdType), maxLines)
	cleaned, cached := fc.Get(cacheKey)
	if !cached {
		chain := proxy.BuildChainWithKeep(cfg, cmdType, maxLines, raw)
		cleaned = chain.Apply(raw)
		fc.Put(cacheKey, cleaned)
	}
	filterDur := time.Since(filterStart)

	// Re-request detection
	normKey := NormalizeExecKey(args.Command)
	reReq := reReqTracker.Check("gotk_exec", args.Command, normKey, maxLines, args.NoTruncate)
	reReqTracker.Record("gotk_exec", args.Command, normKey, maxLines, args.NoTruncate)

	logMCPMeasurement(args.Command, cmdType.String(), string(cfg.General.Mode), raw, cleaned, filterDur, cached, reReq)

	// Build stats text
	stats := buildStats(raw, cleaned)
	if cached {
		stats += " [cached]"
	}

	text := cleaned
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += stats

	if result.ExitCode != 0 {
		text += fmt.Sprintf("\n[exit code: %d]", result.ExitCode)
	}

	sendResult(id, toolCallResult{
		Content: []contentBlock{{Type: "text", Text: text}},
	})
}

func handleFilter(cfg *config.Config, fc *cache.Cache, id json.RawMessage, rawArgs json.RawMessage) {
	// Input size validation
	if len(rawArgs) > maxInputSize {
		sendError(id, -32602, fmt.Sprintf("input too large: %d bytes exceeds %d byte limit", len(rawArgs), maxInputSize))
		return
	}

	var args filterArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		sendError(id, -32602, "Invalid arguments: "+err.Error())
		return
	}

	if args.Input == "" {
		sendError(id, -32602, "input is required")
		return
	}

	// Determine max lines for this request
	maxLines := cfg.General.MaxLines
	if args.MaxLines > 0 {
		maxLines = args.MaxLines
	}

	// Detect command type from hint or auto-detect
	var cmdType detect.CmdType
	if args.CommandHint != "" {
		cmdType = detect.Identify(args.CommandHint)
	} else {
		cmdType = detect.AutoDetect(args.Input)
	}

	// Check cache for previously filtered identical input
	filterStart := time.Now()
	cacheKey := fc.Key(args.Input, int(cmdType), maxLines)
	cleaned, cached := fc.Get(cacheKey)
	if !cached {
		chain := proxy.BuildChainWithKeep(cfg, cmdType, maxLines, args.Input)
		cleaned = chain.Apply(args.Input)
		fc.Put(cacheKey, cleaned)
	}
	filterDur := time.Since(filterStart)

	hint := args.CommandHint
	if hint == "" {
		hint = "filter"
	}
	normKey := NormalizeFilterKey(args.Input)
	reReq := reReqTracker.Check("gotk_filter", hint, normKey, args.MaxLines, false)
	reReqTracker.Record("gotk_filter", hint, normKey, args.MaxLines, false)

	logMCPMeasurement(hint, cmdType.String(), string(cfg.General.Mode), args.Input, cleaned, filterDur, cached, reReq)

	stats := buildStats(args.Input, cleaned)
	if cached {
		stats += " [cached]"
	}

	text := cleaned
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += stats

	sendResult(id, toolCallResult{
		Content: []contentBlock{{Type: "text", Text: text}},
	})
}

// defaultReadMaxLines is the default max lines for gotk_read (higher than exec since files are read intentionally).
const defaultReadMaxLines = 200

// defaultGrepMaxLines is the default max lines for gotk_grep.
const defaultGrepMaxLines = 100

func handleRead(cfg *config.Config, fc *cache.Cache, id json.RawMessage, rawArgs json.RawMessage) {
	if len(rawArgs) > maxInputSize {
		sendError(id, -32602, fmt.Sprintf("input too large: %d bytes exceeds %d byte limit", len(rawArgs), maxInputSize))
		return
	}

	var args readArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		sendError(id, -32602, "Invalid arguments: "+err.Error())
		return
	}

	if args.Path == "" {
		sendError(id, -32602, "path is required")
		return
	}

	// Validate path is under project root (prevent traversal)
	safePath, pathErr := validatePath(args.Path)
	if pathErr != "" {
		sendError(id, -32602, pathErr)
		return
	}

	logErr("READ: %s", safePath)

	// Read the file
	data, err := os.ReadFile(safePath)
	if err != nil {
		sendError(id, -32603, "read failed: "+err.Error())
		return
	}

	raw := string(data)

	// Apply offset and limit if specified (line-based)
	if args.Offset > 0 || args.Limit > 0 {
		lines := strings.Split(raw, "\n")
		start := 0
		if args.Offset > 0 {
			start = args.Offset - 1 // 1-based to 0-based
			if start > len(lines) {
				start = len(lines)
			}
		}
		end := len(lines)
		if args.Limit > 0 && start+args.Limit < end {
			end = start + args.Limit
		}
		raw = strings.Join(lines[start:end], "\n")
	}

	// Determine max lines
	maxLines := defaultReadMaxLines
	if args.MaxLines > 0 {
		maxLines = args.MaxLines
	}

	// Filter with CmdGeneric (file content, no command-specific filters)
	filterStart := time.Now()
	cacheKey := fc.Key(raw, int(detect.CmdGeneric), maxLines)
	cleaned, cached := fc.Get(cacheKey)
	if !cached {
		chain := proxy.BuildChainWithKeep(cfg, detect.CmdGeneric, maxLines, raw)
		cleaned = chain.Apply(raw)
		fc.Put(cacheKey, cleaned)
	}
	filterDur := time.Since(filterStart)

	normKey := NormalizeReadKey(args.Path)
	rawKey := fmt.Sprintf("read:%s:%d:%d", args.Path, args.Offset, args.Limit)
	reReq := reReqTracker.Check("gotk_read", rawKey, normKey, maxLines, false)
	reReqTracker.Record("gotk_read", rawKey, normKey, maxLines, false)

	logMCPMeasurement("read:"+args.Path, "generic", string(cfg.General.Mode), raw, cleaned, filterDur, cached, reReq)

	stats := buildStats(raw, cleaned)
	if cached {
		stats += " [cached]"
	}

	text := cleaned
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += stats

	sendResult(id, toolCallResult{
		Content: []contentBlock{{Type: "text", Text: text}},
	})
}

func handleGrep(cfg *config.Config, fc *cache.Cache, id json.RawMessage, rawArgs json.RawMessage) {
	if len(rawArgs) > maxInputSize {
		sendError(id, -32602, fmt.Sprintf("input too large: %d bytes exceeds %d byte limit", len(rawArgs), maxInputSize))
		return
	}

	var args grepArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		sendError(id, -32602, "Invalid arguments: "+err.Error())
		return
	}

	if args.Pattern == "" {
		sendError(id, -32602, "pattern is required")
		return
	}

	// Build grep command
	grepArgs := []string{"grep"}
	if args.Recursive {
		grepArgs = append(grepArgs, "-r")
	}
	if args.IgnoreCase {
		grepArgs = append(grepArgs, "-i")
	}
	if args.LineNumber {
		grepArgs = append(grepArgs, "-n")
	}

	// Default: recursive with line numbers
	if !args.Recursive && !args.IgnoreCase && !args.LineNumber {
		grepArgs = []string{"grep", "-rn"}
	}

	grepArgs = append(grepArgs, "--", args.Pattern)

	path := args.Path
	if path == "" {
		path = "."
	}

	// Validate path is under project root (prevent traversal)
	safePath, pathErr := validatePath(path)
	if pathErr != "" {
		sendError(id, -32602, pathErr)
		return
	}
	path = safePath

	grepArgs = append(grepArgs, path)

	command := strings.Join(grepArgs, " ")

	// Sandbox check if enabled
	if cfg.Security.SandboxMode {
		if reason := validateSandbox(command); reason != "" {
			logErr("SANDBOX BLOCKED: %q - %s", command, reason)
			sendError(id, -32602, reason)
			return
		}
	}

	logErr("GREP: %s", command)

	// Execute grep
	shell := findShell()
	timeout := defaultMCPTimeout
	if cfg.Security.CommandTimeout > 0 {
		timeout = time.Duration(cfg.Security.CommandTimeout) * time.Second
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	result, err := exec.RunWithTimeout(ctx, shell, "-c", command)
	if err != nil {
		sendError(id, -32603, "grep failed: "+err.Error())
		return
	}

	raw := result.Stdout

	// Determine max lines
	maxLines := defaultGrepMaxLines
	if args.MaxLines > 0 {
		maxLines = args.MaxLines
	}

	// Filter with CmdGrep for grep-specific compression
	filterStart := time.Now()
	cacheKey := fc.Key(raw, int(detect.CmdGrep), maxLines)
	cleaned, cached := fc.Get(cacheKey)
	if !cached {
		chain := proxy.BuildChainWithKeep(cfg, detect.CmdGrep, maxLines, raw)
		cleaned = chain.Apply(raw)
		fc.Put(cacheKey, cleaned)
	}
	filterDur := time.Since(filterStart)

	normKey := NormalizeGrepKey(args.Pattern, path)
	reReq := reReqTracker.Check("gotk_grep", command, normKey, maxLines, false)
	reReqTracker.Record("gotk_grep", command, normKey, maxLines, false)

	logMCPMeasurement(command, "grep", string(cfg.General.Mode), raw, cleaned, filterDur, cached, reReq)

	stats := buildStats(raw, cleaned)
	if cached {
		stats += " [cached]"
	}

	text := cleaned
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += stats

	if result.ExitCode > 1 {
		// grep exit code 1 = no matches (not an error), >1 = actual error
		text += fmt.Sprintf("\n[exit code: %d]", result.ExitCode)
	}
	if result.ExitCode == 1 {
		text = "[no matches found]\n" + stats
	}

	sendResult(id, toolCallResult{
		Content: []contentBlock{{Type: "text", Text: text}},
	})
}

func handleCtx(cfg *config.Config, id json.RawMessage, rawArgs json.RawMessage) {
	if len(rawArgs) > maxInputSize {
		sendError(id, -32602, fmt.Sprintf("input too large: %d bytes exceeds %d byte limit", len(rawArgs), maxInputSize))
		return
	}

	var args ctxArgs
	if err := json.Unmarshal(rawArgs, &args); err != nil {
		sendError(id, -32602, "Invalid arguments: "+err.Error())
		return
	}

	if args.Pattern == "" {
		sendError(id, -32602, "pattern is required")
		return
	}

	logErr("CTX: pattern=%q mode=%s path=%s", args.Pattern, args.Mode, args.Path)

	// Build CLI-style args for gotkctx.Run
	var ctxCLIArgs []string

	switch args.Mode {
	case "detail":
		ctxCLIArgs = append(ctxCLIArgs, "-d")
		if args.Context > 0 {
			ctxCLIArgs = append(ctxCLIArgs, fmt.Sprintf("%d", args.Context))
		}
	case "def":
		ctxCLIArgs = append(ctxCLIArgs, "--def")
	case "tree":
		ctxCLIArgs = append(ctxCLIArgs, "--tree")
	case "summary":
		ctxCLIArgs = append(ctxCLIArgs, "--summary")
	}

	if args.FileType != "" {
		ctxCLIArgs = append(ctxCLIArgs, "-t", args.FileType)
	}
	if args.MaxFiles > 0 {
		ctxCLIArgs = append(ctxCLIArgs, "-m", fmt.Sprintf("%d", args.MaxFiles))
	}
	if args.Path != "" {
		// Validate path is under project root (prevent traversal)
		safePath, pathErr := validatePath(args.Path)
		if pathErr != "" {
			sendError(id, -32602, pathErr)
			return
		}
		ctxCLIArgs = append(ctxCLIArgs, "-p", safePath)
	}

	ctxCLIArgs = append(ctxCLIArgs, args.Pattern)

	maxLines := cfg.General.MaxLines
	output, err := gotkctx.Run(cfg, ctxCLIArgs, maxLines, false)
	if err != nil {
		sendError(id, -32603, err.Error())
		return
	}

	text := output
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}

	sendResult(id, toolCallResult{
		Content: []contentBlock{{Type: "text", Text: text}},
	})
}

// detectProfileFromInit extracts the LLM profile from the MCP initialize params.
// MCP clients send {"clientInfo": {"name": "claude-code", ...}} or similar.
func detectProfileFromInit(params json.RawMessage) config.LLMProfile {
	if params == nil {
		return config.ProfileNone
	}
	var p struct {
		ClientInfo struct {
			Name string `json:"name"`
		} `json:"clientInfo"`
	}
	if err := json.Unmarshal(params, &p); err != nil || p.ClientInfo.Name == "" {
		return config.ProfileNone
	}

	name := strings.ToLower(p.ClientInfo.Name)
	switch {
	case strings.Contains(name, "claude"):
		return config.ProfileClaude
	case strings.Contains(name, "cursor"):
		// Cursor uses various models but primarily GPT/Claude
		return config.ProfileGPT
	case strings.Contains(name, "openai"), strings.Contains(name, "gpt"), strings.Contains(name, "chatgpt"):
		return config.ProfileGPT
	case strings.Contains(name, "gemini"), strings.Contains(name, "google"):
		return config.ProfileGemini
	case strings.Contains(name, "continue"):
		// Continue.dev supports multiple models, default to balanced
		return config.ProfileNone
	default:
		return config.ProfileNone
	}
}

// logMCPMeasurement logs a measurement entry for an MCP tool invocation.
func logMCPMeasurement(command, cmdType, mode, raw, cleaned string, dur time.Duration, cached bool, reReq ReRequestResult) {
	if mcpMeasureLogger == nil {
		return
	}
	rawTokens := measure.EstimateTokens(raw)
	cleanTokens := measure.EstimateTokens(cleaned)
	saved := rawTokens - cleanTokens
	var pct float64
	if rawTokens > 0 {
		pct = float64(saved) / float64(rawTokens) * 100
	}
	quality, important := measure.ComputeQualityScore(raw, cleaned)

	entry := measure.Entry{
		Command:        command,
		CommandType:    cmdType,
		RawBytes:       len(raw),
		CleanBytes:     len(cleaned),
		RawTokens:      rawTokens,
		CleanTokens:    cleanTokens,
		TokensSaved:    saved,
		ReductionPct:   pct,
		LinesRaw:       measure.CountLines(raw),
		LinesClean:     measure.CountLines(cleaned),
		ImportantLines: important,
		QualityScore:   quality,
		Mode:           mode,
		Source:         "mcp",
		Cached:         cached,
		DurationUs:     dur.Microseconds(),
	}

	if reReq.IsReRequest {
		entry.ReRequest = true
		entry.ReRequestType = reReq.Type
		logErr("RE-REQUEST [%s]: %q (similar to request %.0fs ago)", reReq.Type, command, reReq.TimeSincePrev.Seconds())
	}

	_ = mcpMeasureLogger.Log(entry)
}

func buildStats(raw, cleaned string) string {
	rawBytes := len(raw)
	cleanBytes := len(cleaned)
	pct := 0
	if rawBytes > 0 {
		pct = (rawBytes - cleanBytes) * 100 / rawBytes
	}

	rawLines := countLines(raw)
	cleanLines := countLines(cleaned)

	// Count important lines (errors/warnings/critical) preserved in output
	important := countImportantLines(cleaned)

	if important > 0 {
		return fmt.Sprintf("[gotk: %d → %d lines, %d → %d bytes (-%d%%), %d errors/warnings preserved]",
			rawLines, cleanLines, rawBytes, cleanBytes, pct, important)
	}
	return fmt.Sprintf("[gotk: %d → %d lines, %d → %d bytes (-%d%%)]",
		rawLines, cleanLines, rawBytes, cleanBytes, pct)
}

func countLines(s string) int {
	if s == "" {
		return 0
	}
	n := strings.Count(s, "\n")
	if !strings.HasSuffix(s, "\n") {
		n++
	}
	return n
}

func countImportantLines(s string) int {
	count := 0
	for _, line := range strings.Split(s, "\n") {
		level := classify.Classify(line)
		if level >= classify.Warning {
			count++
		}
	}
	return count
}

func sendResult(id json.RawMessage, result interface{}) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result:  result,
	}
	writeResponse(resp)
}

func sendError(id json.RawMessage, code int, message string) {
	resp := jsonRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error:   &rpcError{Code: code, Message: message},
	}
	writeResponse(resp)
}

func writeResponse(resp jsonRPCResponse) {
	data, err := json.Marshal(resp)
	if err != nil {
		logErr("failed to marshal response: %v", err)
		return
	}
	fmt.Fprintf(os.Stdout, "%s\n", data)
}

// auditFile is the optional file-based audit log. Set by Serve() if configured.
var auditFile *os.File

func logErr(format string, args ...interface{}) {
	msg := fmt.Sprintf("[gotk-mcp] "+format, args...)
	fmt.Fprintln(os.Stderr, msg)

	if auditFile != nil {
		ts := time.Now().Format("2006-01-02T15:04:05.000Z07:00")
		fmt.Fprintf(auditFile, "%s %s\n", ts, msg)
	}
}

// auditLog writes a security event to the audit log file (if configured).
// Used for security-relevant events like rate limiting, command blocking, etc.
func auditLog(event, detail string) {
	if auditFile != nil {
		ts := time.Now().Format("2006-01-02T15:04:05.000Z07:00")
		fmt.Fprintf(auditFile, "%s [SECURITY] %s: %s\n", ts, event, detail)
	}
}

// findShell returns a shell suitable for command execution.
func findShell() string {
	if s := os.Getenv("GOTK_SHELL"); s != "" {
		return s
	}
	if s := os.Getenv("SHELL"); s != "" {
		base := s
		if idx := strings.LastIndexByte(s, '/'); idx >= 0 {
			base = s[idx+1:]
		}
		if base != "gotk" {
			return s
		}
	}
	for _, sh := range []string{"/bin/bash", "/bin/sh"} {
		if _, err := os.Stat(sh); err == nil {
			return sh
		}
	}
	return "sh"
}
