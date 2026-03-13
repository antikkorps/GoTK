package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/antikkorps/GoTK/internal/config"
)

// --- validateCommand tests ---

func TestValidateCommand_BlockedCommands(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{"rm -rf /", "rm -rf /"},
		{"rm -rf /*", "rm -rf /*"},
		{"rm -fr /", "rm -fr /"},
		{"rm -fr /* with prefix", "sudo rm -fr /*"},
		{"mkfs", "mkfs /dev/sda"},
		{"mkfs.ext4", "mkfs.ext4 /dev/sda1"},
		{"dd if=", "dd if=/dev/zero of=/dev/sda"},
		{"shutdown", "shutdown -h now"},
		{"reboot", "reboot"},
		{"halt", "halt"},
		{"init 0", "init 0"},
		{"init 6", "init 6"},
		{"fork bomb", ":(){:|:&};:"},
		{"chmod 777 root", "chmod -R 777 /"},
		{"chown -R", "chown -R root:root /etc"},
		{"mv root", "mv / /tmp"},
		{"mv wildcard root", "mv /* /tmp"},
		{"wipefs", "wipefs /dev/sda"},
		{"format c:", "format c:"},
		{"format C:", "format C:"},
		{"dev sda redirect", "> /dev/sda"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateCommand(tt.command)
			if result == "" {
				t.Errorf("validateCommand(%q) should block the command", tt.command)
			}
			if !strings.Contains(result, "command blocked") {
				t.Errorf("validateCommand(%q) = %q, should contain 'command blocked'", tt.command, result)
			}
		})
	}
}

func TestValidateCommand_AllowedCommands(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{"grep", "grep -rn func ./src/"},
		{"ls", "ls -la /tmp"},
		{"git status", "git status"},
		{"git log", "git log --oneline"},
		{"find", "find . -name '*.go'"},
		{"cat", "cat /etc/hostname"},
		{"echo", "echo hello world"},
		{"go build", "go build ./..."},
		{"go test", "go test ./..."},
		{"npm install", "npm install"},
		{"make", "make build"},
		{"curl", "curl https://example.com"},
		{"rm single file", "rm myfile.txt"},
		{"rm -rf non-root", "rm -rf ./build"},
		{"mv non-root", "mv file1 file2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateCommand(tt.command)
			if result != "" {
				t.Errorf("validateCommand(%q) = %q, should allow the command", tt.command, result)
			}
		})
	}
}

func TestValidateCommand_CaseInsensitive(t *testing.T) {
	result := validateCommand("SHUTDOWN")
	if result == "" {
		t.Error("validateCommand should be case-insensitive for 'SHUTDOWN'")
	}
}

func TestValidateCommand_WhitespaceHandling(t *testing.T) {
	result := validateCommand("  shutdown  ")
	if result == "" {
		t.Error("validateCommand should trim whitespace and still block 'shutdown'")
	}
}

// --- buildToolsList tests ---

func TestBuildToolsList_ReturnsExpectedTools(t *testing.T) {
	result := buildToolsList()

	if len(result.Tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(result.Tools))
	}

	// Verify tool names
	toolNames := make(map[string]bool)
	for _, tool := range result.Tools {
		toolNames[tool.Name] = true
	}

	if !toolNames["gotk_exec"] {
		t.Error("missing tool: gotk_exec")
	}
	if !toolNames["gotk_filter"] {
		t.Error("missing tool: gotk_filter")
	}
}

func TestBuildToolsList_ExecToolSchema(t *testing.T) {
	result := buildToolsList()

	var execTool *toolDef
	for i := range result.Tools {
		if result.Tools[i].Name == "gotk_exec" {
			execTool = &result.Tools[i]
			break
		}
	}

	if execTool == nil {
		t.Fatal("gotk_exec tool not found")
	}

	if execTool.InputSchema.Type != "object" {
		t.Errorf("InputSchema.Type = %q, want %q", execTool.InputSchema.Type, "object")
	}

	requiredProps := []string{"command"}
	for _, req := range requiredProps {
		found := false
		for _, r := range execTool.InputSchema.Required {
			if r == req {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("missing required property: %s", req)
		}
	}

	expectedProps := []string{"command", "max_lines", "no_truncate", "aggressive", "timeout"}
	for _, prop := range expectedProps {
		if _, ok := execTool.InputSchema.Properties[prop]; !ok {
			t.Errorf("missing property: %s", prop)
		}
	}
}

func TestBuildToolsList_FilterToolSchema(t *testing.T) {
	result := buildToolsList()

	var filterTool *toolDef
	for i := range result.Tools {
		if result.Tools[i].Name == "gotk_filter" {
			filterTool = &result.Tools[i]
			break
		}
	}

	if filterTool == nil {
		t.Fatal("gotk_filter tool not found")
	}

	if len(filterTool.InputSchema.Required) != 1 || filterTool.InputSchema.Required[0] != "input" {
		t.Errorf("Required = %v, want [input]", filterTool.InputSchema.Required)
	}

	expectedProps := []string{"input", "command_hint", "max_lines"}
	for _, prop := range expectedProps {
		if _, ok := filterTool.InputSchema.Properties[prop]; !ok {
			t.Errorf("missing property: %s", prop)
		}
	}
}

// --- buildStats tests ---

func TestBuildStats(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		cleaned string
		wantPct string
	}{
		{"50% reduction", strings.Repeat("x", 1000), strings.Repeat("x", 500), "-50%"},
		{"no reduction", strings.Repeat("x", 100), strings.Repeat("x", 100), "-0%"},
		{"full reduction", strings.Repeat("x", 100), "", "-100%"},
		{"zero input", "", "", "-0%"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildStats(tt.raw, tt.cleaned)

			if !strings.Contains(got, tt.wantPct) {
				t.Errorf("buildStats = %q, should contain %q", got, tt.wantPct)
			}

			if !strings.Contains(got, "[gotk:") {
				t.Errorf("buildStats result should contain '[gotk:'")
			}

			if !strings.Contains(got, "bytes") {
				t.Errorf("buildStats result should contain 'bytes'")
			}
		})
	}
}

func TestBuildStats_IncludesLines(t *testing.T) {
	raw := "line1\nline2\nline3\nline4\nline5\n"
	cleaned := "line1\nline5\n"
	got := buildStats(raw, cleaned)

	if !strings.Contains(got, "5 →") {
		t.Errorf("should show raw line count, got: %q", got)
	}
	if !strings.Contains(got, "→ 2 lines") {
		t.Errorf("should show clean line count, got: %q", got)
	}
}

func TestBuildStats_ShowsImportantLines(t *testing.T) {
	raw := "ok\nERROR: bad\nWARNING: watch out\n"
	cleaned := "ok\nERROR: bad\nWARNING: watch out\n"
	got := buildStats(raw, cleaned)

	if !strings.Contains(got, "errors/warnings preserved") {
		t.Errorf("should mention preserved errors/warnings, got: %q", got)
	}
}

func TestBuildStats_NoImportantLines(t *testing.T) {
	raw := "line1\nline2\n"
	cleaned := "line1\n"
	got := buildStats(raw, cleaned)

	if strings.Contains(got, "errors/warnings") {
		t.Errorf("should not mention errors/warnings when none present, got: %q", got)
	}
}

// --- handleRequest routing tests ---

func TestHandleRequest_Initialize(t *testing.T) {
	// Capture stdout to read the response
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := config.Default()
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`1`),
		Method:  "initialize",
	}
	handleRequest(cfg, req)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	var resp jsonRPCResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v (raw: %s)", err, buf.String())
	}

	if resp.Error != nil {
		t.Errorf("initialize should not return error, got: %s", resp.Error.Message)
	}
	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want %q", resp.JSONRPC, "2.0")
	}

	// Check that result contains serverInfo
	resultJSON, _ := json.Marshal(resp.Result)
	if !strings.Contains(string(resultJSON), "gotk") {
		t.Error("initialize result should contain server name 'gotk'")
	}
	if !strings.Contains(string(resultJSON), "protocolVersion") {
		t.Error("initialize result should contain protocolVersion")
	}
}

func TestHandleRequest_Ping(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := config.Default()
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`2`),
		Method:  "ping",
	}
	handleRequest(cfg, req)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	var resp jsonRPCResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("ping should not return error, got: %s", resp.Error.Message)
	}
}

func TestHandleRequest_ToolsList(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := config.Default()
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`3`),
		Method:  "tools/list",
	}
	handleRequest(cfg, req)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	var resp jsonRPCResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error != nil {
		t.Errorf("tools/list should not return error, got: %s", resp.Error.Message)
	}

	resultJSON, _ := json.Marshal(resp.Result)
	if !strings.Contains(string(resultJSON), "gotk_exec") {
		t.Error("tools/list should include gotk_exec")
	}
	if !strings.Contains(string(resultJSON), "gotk_filter") {
		t.Error("tools/list should include gotk_filter")
	}
}

func TestHandleRequest_UnknownMethod(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cfg := config.Default()
	req := jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`4`),
		Method:  "nonexistent/method",
	}
	handleRequest(cfg, req)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	var resp jsonRPCResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}

	if resp.Error == nil {
		t.Fatal("unknown method should return an error")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("error code = %d, want -32601", resp.Error.Code)
	}
	if !strings.Contains(resp.Error.Message, "Method not found") {
		t.Errorf("error message = %q, should contain 'Method not found'", resp.Error.Message)
	}
}

// --- JSON-RPC parsing tests ---

func TestJSONRPC_ValidRequest(t *testing.T) {
	raw := `{"jsonrpc":"2.0","id":1,"method":"ping"}`
	var req jsonRPCRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("failed to parse valid request: %v", err)
	}
	if req.JSONRPC != "2.0" {
		t.Errorf("JSONRPC = %q, want %q", req.JSONRPC, "2.0")
	}
	if req.Method != "ping" {
		t.Errorf("Method = %q, want %q", req.Method, "ping")
	}
}

func TestJSONRPC_InvalidRequest(t *testing.T) {
	tests := []struct {
		name string
		raw  string
	}{
		{"empty object", "{}"},
		{"missing method", `{"jsonrpc":"2.0","id":1}`},
		{"not json", "not json at all"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var req jsonRPCRequest
			err := json.Unmarshal([]byte(tt.raw), &req)
			// For "not json", parsing should fail
			if tt.name == "not json" && err == nil {
				t.Error("expected parse error for invalid JSON")
			}
			// For valid JSON with missing fields, parsing succeeds but fields are zero values
			if tt.name != "not json" && err != nil {
				t.Errorf("unexpected parse error: %v", err)
			}
		})
	}
}

func TestJSONRPC_RequestWithParams(t *testing.T) {
	raw := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"gotk_exec","arguments":{"command":"echo hi"}}}`
	var req jsonRPCRequest
	if err := json.Unmarshal([]byte(raw), &req); err != nil {
		t.Fatalf("failed to parse request: %v", err)
	}
	if req.Method != "tools/call" {
		t.Errorf("Method = %q, want %q", req.Method, "tools/call")
	}
	if req.Params == nil {
		t.Error("Params should not be nil")
	}
}

// --- Full MCP flow test using pipes ---

func TestServe_FullFlow(t *testing.T) {
	// Build a sequence of JSON-RPC requests
	requests := []string{
		`{"jsonrpc":"2.0","id":1,"method":"initialize"}`,
		`{"jsonrpc":"2.0","method":"notifications/initialized"}`,
		`{"jsonrpc":"2.0","id":2,"method":"ping"}`,
		`{"jsonrpc":"2.0","id":3,"method":"tools/list"}`,
	}

	input := strings.Join(requests, "\n") + "\n"

	// Replace stdin and stdout
	oldStdin := os.Stdin
	oldStdout := os.Stdout

	stdinR, stdinW, _ := os.Pipe()
	stdoutR, stdoutW, _ := os.Pipe()

	os.Stdin = stdinR
	os.Stdout = stdoutW

	// Write input and close
	go func() {
		stdinW.WriteString(input)
		stdinW.Close()
	}()

	cfg := config.Default()

	// Run Serve in a goroutine
	done := make(chan struct{})
	go func() {
		Serve(cfg)
		stdoutW.Close()
		close(done)
	}()

	// Read all output
	var output bytes.Buffer
	io.Copy(&output, stdoutR)
	<-done

	// Restore
	os.Stdin = oldStdin
	os.Stdout = oldStdout

	// Parse responses (one per line, notification produces no response)
	scanner := bufio.NewScanner(&output)
	var responses []jsonRPCResponse
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		var resp jsonRPCResponse
		if err := json.Unmarshal([]byte(line), &resp); err != nil {
			t.Errorf("failed to parse response line: %v (raw: %s)", err, line)
			continue
		}
		responses = append(responses, resp)
	}

	// We expect 3 responses: initialize, ping, tools/list
	// (notifications/initialized produces no response)
	if len(responses) != 3 {
		t.Fatalf("expected 3 responses, got %d", len(responses))
	}

	// Verify initialize response
	if responses[0].Error != nil {
		t.Errorf("initialize response has error: %s", responses[0].Error.Message)
	}

	// Verify ping response
	if responses[1].Error != nil {
		t.Errorf("ping response has error: %s", responses[1].Error.Message)
	}

	// Verify tools/list response
	if responses[2].Error != nil {
		t.Errorf("tools/list response has error: %s", responses[2].Error.Message)
	}
}

// --- execArgs timeout tests ---

func TestExecArgs_TimeoutOverride(t *testing.T) {
	raw := `{"command":"sleep 1","timeout":120}`
	var args execArgs
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		t.Fatalf("failed to parse execArgs: %v", err)
	}
	if args.Timeout != 120 {
		t.Errorf("Timeout = %d, want 120", args.Timeout)
	}
}

func TestExecArgs_DefaultTimeout(t *testing.T) {
	raw := `{"command":"echo hi"}`
	var args execArgs
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		t.Fatalf("failed to parse execArgs: %v", err)
	}
	if args.Timeout != 0 {
		t.Errorf("Timeout = %d, want 0 (use config default)", args.Timeout)
	}
}

// --- findShell tests (mcp-local copy) ---

func TestFindShell_RespectsGOTK_SHELL(t *testing.T) {
	origGOTK := os.Getenv("GOTK_SHELL")
	origSHELL := os.Getenv("SHELL")
	defer func() {
		os.Setenv("GOTK_SHELL", origGOTK)
		os.Setenv("SHELL", origSHELL)
	}()

	os.Setenv("GOTK_SHELL", "/usr/local/bin/test-shell")
	got := findShell()
	if got != "/usr/local/bin/test-shell" {
		t.Errorf("findShell() = %q, want /usr/local/bin/test-shell", got)
	}
}

func TestFindShell_AvoidsRecursion(t *testing.T) {
	origGOTK := os.Getenv("GOTK_SHELL")
	origSHELL := os.Getenv("SHELL")
	defer func() {
		os.Setenv("GOTK_SHELL", origGOTK)
		os.Setenv("SHELL", origSHELL)
	}()

	os.Unsetenv("GOTK_SHELL")
	os.Setenv("SHELL", "/usr/bin/gotk")

	got := findShell()
	if got == "/usr/bin/gotk" {
		t.Error("findShell() should not return gotk to avoid recursion")
	}
}

func TestFindShell_ReturnsNonEmpty(t *testing.T) {
	got := findShell()
	if got == "" {
		t.Error("findShell() should never return an empty string")
	}
}
