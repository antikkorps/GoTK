package mcp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/antikkorps/GoTK/internal/config"
	"github.com/antikkorps/GoTK/internal/detect"
	"github.com/antikkorps/GoTK/internal/exec"
	"github.com/antikkorps/GoTK/internal/filter"
	"github.com/antikkorps/GoTK/internal/proxy"
)

// dangerousCommands is a denylist of destructive commands that should be blocked.
var dangerousCommands = []string{
	"rm -rf /",
	"rm -rf /*",
	"rm -fr /",
	"rm -fr /*",
	"mkfs",
	"mkfs.",       // mkfs.ext4, mkfs.xfs, etc.
	"dd if=",
	"format c:",
	"format C:",
	":(){:|:&};:", // fork bomb
	"chmod -R 777 /",
	"chown -R",
	"shutdown",
	"reboot",
	"halt",
	"init 0",
	"init 6",
	"mv / ",
	"mv /* ",
	"> /dev/sda",
	"wipefs",
}

// maxInputSize is the maximum allowed input size for MCP requests (10MB).
const maxInputSize = 10 * 1024 * 1024

// validateCommand checks whether a command is on the denylist of dangerous commands.
// Returns an error message if the command is blocked, empty string if allowed.
func validateCommand(command string) string {
	normalized := strings.TrimSpace(command)
	lower := strings.ToLower(normalized)

	for _, dangerous := range dangerousCommands {
		if strings.Contains(lower, strings.ToLower(dangerous)) {
			return fmt.Sprintf("command blocked: contains dangerous pattern %q", dangerous)
		}
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
	JSONRPC string      `json:"jsonrpc"`
	ID      json.RawMessage `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *rpcError   `json:"error,omitempty"`
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
	Type       string                 `json:"type"`
	Properties map[string]propDef     `json:"properties"`
	Required   []string               `json:"required"`
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

// Serve starts the MCP server, reading JSON-RPC from stdin and writing to stdout.
func Serve(cfg *config.Config) {
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

		handleRequest(cfg, req)
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

func handleRequest(cfg *config.Config, req jsonRPCRequest) {
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
		handleToolCall(cfg, req)

	default:
		sendError(req.ID, -32601, "Method not found: "+req.Method)
	}
}

func buildToolsList() toolsListResult {
	return toolsListResult{
		Tools: []toolDef{
			{
				Name:        "gotk_exec",
				Description: "Execute a command and return cleaned output with noise removed. Reduces token usage by 51-96%.",
				InputSchema: inputSchema{
					Type: "object",
					Properties: map[string]propDef{
						"command":     {Type: "string", Description: "The command to execute (e.g., 'grep -rn func ./src/')"},
						"max_lines":   {Type: "integer", Description: "Maximum output lines (default: 50)"},
						"no_truncate": {Type: "boolean", Description: "Disable truncation"},
						"aggressive":  {Type: "boolean", Description: "Aggressive filtering mode"},
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

func handleToolCall(cfg *config.Config, req jsonRPCRequest) {
	var params toolCallParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		sendError(req.ID, -32602, "Invalid params: "+err.Error())
		return
	}

	switch params.Name {
	case "gotk_exec":
		handleExec(cfg, req.ID, params.Arguments)
	case "gotk_filter":
		handleFilter(cfg, req.ID, params.Arguments)
	default:
		sendError(req.ID, -32602, "Unknown tool: "+params.Name)
	}
}

func handleExec(cfg *config.Config, id json.RawMessage, rawArgs json.RawMessage) {
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

	// Configure max lines
	savedMaxLines := filter.MaxLines
	if args.MaxLines > 0 {
		filter.MaxLines = args.MaxLines
	}
	if args.NoTruncate {
		filter.MaxLines = 0
	}
	defer func() { filter.MaxLines = savedMaxLines }()

	// Parse the command to detect type
	parts := strings.Fields(args.Command)
	if len(parts) == 0 {
		sendError(id, -32602, "empty command")
		return
	}

	// Execute via shell with timeout
	shell := findShell()
	timeout := time.Duration(cfg.Security.CommandTimeout) * time.Second
	if timeout <= 0 {
		timeout = exec.DefaultTimeout
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

	chain := proxy.BuildChain(cfg, cmdType)
	cleaned := chain.Apply(raw)

	// Redact secrets if enabled
	if cfg.Security.RedactSecrets {
		cleaned = filter.RedactSecrets(cleaned)
	}

	// Build stats text
	stats := buildStats(len(raw), len(cleaned))

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

func handleFilter(cfg *config.Config, id json.RawMessage, rawArgs json.RawMessage) {
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

	// Configure max lines
	savedMaxLines := filter.MaxLines
	if args.MaxLines > 0 {
		filter.MaxLines = args.MaxLines
	}
	defer func() { filter.MaxLines = savedMaxLines }()

	// Detect command type from hint or auto-detect
	var cmdType detect.CmdType
	if args.CommandHint != "" {
		cmdType = detect.Identify(args.CommandHint)
	} else {
		cmdType = detect.AutoDetect(args.Input)
	}

	chain := proxy.BuildChain(cfg, cmdType)
	cleaned := chain.Apply(args.Input)

	// Redact secrets if enabled
	if cfg.Security.RedactSecrets {
		cleaned = filter.RedactSecrets(cleaned)
	}

	stats := buildStats(len(args.Input), len(cleaned))

	text := cleaned
	if text != "" && !strings.HasSuffix(text, "\n") {
		text += "\n"
	}
	text += stats

	sendResult(id, toolCallResult{
		Content: []contentBlock{{Type: "text", Text: text}},
	})
}

func buildStats(rawBytes, cleanBytes int) string {
	saved := rawBytes - cleanBytes
	pct := 0
	if rawBytes > 0 {
		pct = saved * 100 / rawBytes
	}
	return fmt.Sprintf("[gotk: %d -> %d bytes, -%d%% reduction]", rawBytes, cleanBytes, pct)
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
