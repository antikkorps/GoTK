package mcp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/antikkorps/GoTK/internal/cache"
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

// --- validateSandbox tests ---

func TestValidateSandbox_AllowedCommands(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{"grep", "grep -rn func ./src/"},
		{"ls", "ls -la /tmp"},
		{"cat", "cat /etc/hostname"},
		{"find", "find . -name '*.go'"},
		{"git status", "git status"},
		{"git log", "git log --oneline"},
		{"go test", "go test ./..."},
		{"head", "head -20 file.txt"},
		{"wc", "wc -l file.txt"},
		{"echo", "echo hello"},
		{"pipeline", "grep -rn func . | head -10"},
		{"pipeline with sort", "ls -la | sort | head"},
		{"jq", "cat file.json | jq '.key'"},
		{"curl", "curl https://example.com"},
		{"diff", "diff file1 file2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateSandbox(tt.command)
			if result != "" {
				t.Errorf("validateSandbox(%q) = %q, should allow", tt.command, result)
			}
		})
	}
}

func TestValidateSandbox_BlockedCommands(t *testing.T) {
	tests := []struct {
		name    string
		command string
	}{
		{"rm", "rm file.txt"},
		{"cp", "cp file1 file2"},
		{"mv", "mv file1 file2"},
		{"chmod", "chmod 755 file"},
		{"chown", "chown user file"},
		{"mkdir", "mkdir newdir"},
		{"touch", "touch newfile"},
		{"tee", "echo hello | tee file.txt"},
		{"sed -i", "sed -i 's/a/b/' file"},
		{"redirect write", "echo hello > file.txt"},
		{"redirect append", "echo hello >> file.txt"},
		{"unknown command", "mycustomtool --flag"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := validateSandbox(tt.command)
			if result == "" {
				t.Errorf("validateSandbox(%q) should block the command", tt.command)
			}
			if !strings.Contains(result, "sandbox") {
				t.Errorf("validateSandbox(%q) = %q, should contain 'sandbox'", tt.command, result)
			}
		})
	}
}

func TestValidateSandbox_PipelineWithBlockedCommand(t *testing.T) {
	result := validateSandbox("ls -la | rm file.txt")
	if result == "" {
		t.Error("should block pipeline containing non-allowed command")
	}
}

// --- buildToolsList tests ---

func TestBuildToolsList_ReturnsExpectedTools(t *testing.T) {
	result := buildToolsList()

	if len(result.Tools) != 5 {
		t.Fatalf("expected 5 tools, got %d", len(result.Tools))
	}

	// Verify tool names
	toolNames := make(map[string]bool)
	for _, tool := range result.Tools {
		toolNames[tool.Name] = true
	}

	for _, name := range []string{"gotk_exec", "gotk_filter", "gotk_read", "gotk_grep", "gotk_ctx"} {
		if !toolNames[name] {
			t.Errorf("missing tool: %s", name)
		}
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
	handleRequest(cfg, newRateLimiter(0, 0), cache.New(0, ""), req)

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
	handleRequest(cfg, newRateLimiter(0, 0), cache.New(0, ""), req)

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
	handleRequest(cfg, newRateLimiter(0, 0), cache.New(0, ""), req)

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
	handleRequest(cfg, newRateLimiter(0, 0), cache.New(0, ""), req)

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

func TestHandleRequest_RateLimited(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cfg := config.Default()
	limiter := newRateLimiter(60, 2) // burst of 2

	toolCallReq := func(id int) jsonRPCRequest {
		return jsonRPCRequest{
			JSONRPC: "2.0",
			ID:      json.RawMessage(fmt.Sprintf(`%d`, id)),
			Method:  "tools/call",
			Params:  json.RawMessage(`{"name":"gotk_filter","arguments":{"input":"hello"}}`),
		}
	}

	// First two should succeed (burst=2)
	handleRequest(cfg, limiter, cache.New(0, ""), toolCallReq(1))
	handleRequest(cfg, limiter, cache.New(0, ""), toolCallReq(2))
	// Third should be rate limited
	handleRequest(cfg, limiter, cache.New(0, ""), toolCallReq(3))

	// Ping should still work (not rate limited)
	handleRequest(cfg, limiter, cache.New(0, ""), jsonRPCRequest{
		JSONRPC: "2.0",
		ID:      json.RawMessage(`4`),
		Method:  "ping",
	})

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 responses, got %d: %v", len(lines), lines)
	}

	// Check third response is rate limited
	var resp3 jsonRPCResponse
	if err := json.Unmarshal([]byte(lines[2]), &resp3); err != nil {
		t.Fatalf("failed to parse response 3: %v", err)
	}
	if resp3.Error == nil {
		t.Fatal("third request should be rate limited")
	}
	if resp3.Error.Code != -32000 {
		t.Errorf("error code = %d, want -32000", resp3.Error.Code)
	}
	if !strings.Contains(resp3.Error.Message, "Rate limit exceeded") {
		t.Errorf("error message = %q, should contain 'Rate limit exceeded'", resp3.Error.Message)
	}

	// Check fourth (ping) response succeeds
	var resp4 jsonRPCResponse
	if err := json.Unmarshal([]byte(lines[3]), &resp4); err != nil {
		t.Fatalf("failed to parse response 4: %v", err)
	}
	if resp4.Error != nil {
		t.Error("ping should not be rate limited")
	}
}

// --- gotk_read tests ---

func TestHandleRead_Success(t *testing.T) {
	// Create a temp file to read
	tmpFile, err := os.CreateTemp("", "gotk_read_test_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	content := "line1\nline2\nline3\nline4\nline5\n"
	tmpFile.WriteString(content)
	tmpFile.Close()

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cfg := config.Default()
	fc := cache.New(128, "test")
	argsJSON, _ := json.Marshal(readArgs{Path: tmpFile.Name(), MaxLines: 100})
	handleRead(cfg, fc, json.RawMessage(`1`), argsJSON)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	var resp jsonRPCResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v (raw: %s)", err, buf.String())
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	// Verify response contains file content
	resultJSON, _ := json.Marshal(resp.Result)
	resultStr := string(resultJSON)
	if !strings.Contains(resultStr, "line1") {
		t.Error("response should contain file content")
	}
	if !strings.Contains(resultStr, "gotk:") {
		t.Error("response should contain stats")
	}
}

func TestHandleRead_FileNotFound(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cfg := config.Default()
	fc := cache.New(0, "")
	argsJSON, _ := json.Marshal(readArgs{Path: "/nonexistent/file.txt"})
	handleRead(cfg, fc, json.RawMessage(`1`), argsJSON)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	var resp jsonRPCResponse
	json.Unmarshal(buf.Bytes(), &resp)
	if resp.Error == nil {
		t.Fatal("should return error for nonexistent file")
	}
	if resp.Error.Code != -32603 {
		t.Errorf("error code = %d, want -32603", resp.Error.Code)
	}
}

func TestHandleRead_OffsetAndLimit(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "gotk_read_offset_*.txt")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.WriteString("line1\nline2\nline3\nline4\nline5\n")
	tmpFile.Close()

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cfg := config.Default()
	fc := cache.New(0, "")
	argsJSON, _ := json.Marshal(readArgs{Path: tmpFile.Name(), Offset: 2, Limit: 2})
	handleRead(cfg, fc, json.RawMessage(`1`), argsJSON)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	var resp jsonRPCResponse
	json.Unmarshal(buf.Bytes(), &resp)
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	resultJSON, _ := json.Marshal(resp.Result)
	resultStr := string(resultJSON)
	// Should contain lines 2-3, not line1
	if strings.Contains(resultStr, "line1") {
		t.Error("should not contain line1 with offset=2")
	}
	if !strings.Contains(resultStr, "line2") {
		t.Error("should contain line2")
	}
}

func TestHandleRead_EmptyPath(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cfg := config.Default()
	fc := cache.New(0, "")
	argsJSON, _ := json.Marshal(readArgs{Path: ""})
	handleRead(cfg, fc, json.RawMessage(`1`), argsJSON)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	var resp jsonRPCResponse
	json.Unmarshal(buf.Bytes(), &resp)
	if resp.Error == nil {
		t.Fatal("should return error for empty path")
	}
}

// --- gotk_grep tests ---

func TestHandleGrep_Success(t *testing.T) {
	// Create temp dir with a file to grep
	tmpDir, err := os.MkdirTemp("", "gotk_grep_test_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	os.WriteFile(tmpDir+"/test.txt", []byte("hello world\nfoo bar\nhello again\n"), 0644)

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cfg := config.Default()
	fc := cache.New(128, "test")
	argsJSON, _ := json.Marshal(grepArgs{Pattern: "hello", Path: tmpDir})
	handleGrep(cfg, fc, json.RawMessage(`1`), argsJSON)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	var resp jsonRPCResponse
	if err := json.Unmarshal(buf.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v (raw: %s)", err, buf.String())
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %s", resp.Error.Message)
	}

	resultJSON, _ := json.Marshal(resp.Result)
	resultStr := string(resultJSON)
	if !strings.Contains(resultStr, "hello") {
		t.Error("grep result should contain matching lines")
	}
}

func TestHandleGrep_NoMatches(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "gotk_grep_nomatch_*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)
	os.WriteFile(tmpDir+"/test.txt", []byte("hello world\n"), 0644)

	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cfg := config.Default()
	fc := cache.New(0, "")
	argsJSON, _ := json.Marshal(grepArgs{Pattern: "zzzznotfound", Path: tmpDir})
	handleGrep(cfg, fc, json.RawMessage(`1`), argsJSON)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	var resp jsonRPCResponse
	json.Unmarshal(buf.Bytes(), &resp)
	if resp.Error != nil {
		t.Fatalf("no matches should not be an error, got: %s", resp.Error.Message)
	}

	resultJSON, _ := json.Marshal(resp.Result)
	if !strings.Contains(string(resultJSON), "no matches") {
		t.Error("should indicate no matches found")
	}
}

func TestHandleGrep_EmptyPattern(t *testing.T) {
	r, w, _ := os.Pipe()
	oldStdout := os.Stdout
	os.Stdout = w

	cfg := config.Default()
	fc := cache.New(0, "")
	argsJSON, _ := json.Marshal(grepArgs{Pattern: ""})
	handleGrep(cfg, fc, json.RawMessage(`1`), argsJSON)

	w.Close()
	os.Stdout = oldStdout

	var buf bytes.Buffer
	io.Copy(&buf, r)

	var resp jsonRPCResponse
	json.Unmarshal(buf.Bytes(), &resp)
	if resp.Error == nil {
		t.Fatal("should return error for empty pattern")
	}
}

// --- audit log tests ---

func TestAuditLogFile(t *testing.T) {
	// Create temp file for audit log
	tmpFile, err := os.CreateTemp("", "gotk_audit_*.log")
	if err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	// Set audit file
	f, err := os.OpenFile(tmpFile.Name(), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		t.Fatal(err)
	}
	oldAuditFile := auditFile
	auditFile = f
	defer func() {
		f.Close()
		auditFile = oldAuditFile
	}()

	// Log something
	logErr("EXEC: %s", "test command")

	f.Sync()

	// Read the audit log
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		t.Fatal(err)
	}

	content := string(data)
	if !strings.Contains(content, "EXEC: test command") {
		t.Errorf("audit log should contain the logged command, got: %q", content)
	}
	// Check timestamp format (ISO 8601)
	if !strings.Contains(content, "T") || !strings.Contains(content, "[gotk-mcp]") {
		t.Errorf("audit log should contain timestamp and prefix, got: %q", content)
	}
}

func TestAuditLogDisabled(t *testing.T) {
	oldAuditFile := auditFile
	auditFile = nil
	defer func() { auditFile = oldAuditFile }()

	// Should not panic when audit file is nil
	logErr("EXEC: %s", "test command")
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
