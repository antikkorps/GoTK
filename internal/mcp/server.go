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
	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/exec"
	"github.com/antikkorps/GoTK/internal/proxy"
)

// dangerousBaseCommands is a denylist of base command names that are always blocked.
var dangerousBaseCommands = map[string]bool{
	"mkfs":     true,
	"shutdown": true,
	"reboot":   true,
	"halt":     true,
	"wipefs":   true,
	"format":   true,
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

// Serve starts the MCP server, reading JSON-RPC from stdin and writing to stdout.
func Serve(cfg *config.Config) {
	limiter := newRateLimiter(cfg.Security.RateLimit, cfg.Security.RateBurst)
	fc := cache.New(cacheMaxEntries, buildConfigHash(cfg))

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
		result := initializeResult{
			ProtocolVersion: "2024-11-05",
			Capabilities:    capObject{Tools: map[string]interface{}{}},
			ServerInfo:      serverInfo{Name: "gotk", Version: "0.2.0"},
		}
		sendResult(req.ID, result)

	case "ping":
		sendResult(req.ID, map[string]interface{}{})

	case "tools/list":
		sendResult(req.ID, buildToolsList())

	case "tools/call":
		if !limiter.Allow() {
			logErr("RATE LIMITED: tools/call rejected")
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
		sendError(id, -32602, reason)
		return
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
	cacheKey := fc.Key(raw, int(cmdType), maxLines)
	cleaned, cached := fc.Get(cacheKey)
	if !cached {
		chain := proxy.BuildChainWithKeep(cfg, cmdType, maxLines, raw)
		cleaned = chain.Apply(raw)
		fc.Put(cacheKey, cleaned)
	}

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
	cacheKey := fc.Key(args.Input, int(cmdType), maxLines)
	cleaned, cached := fc.Get(cacheKey)
	if !cached {
		chain := proxy.BuildChainWithKeep(cfg, cmdType, maxLines, args.Input)
		cleaned = chain.Apply(args.Input)
		fc.Put(cacheKey, cleaned)
	}

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

func logErr(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "[gotk-mcp] "+format+"\n", args...)
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
