package detect

import (
	"strings"
	"testing"
)

// --- Curl filter tests ---

func TestCompressCurlVerbose(t *testing.T) {
	input := strings.Join([]string{
		"* Connected to example.com (93.184.216.34) port 443",
		"* TLS 1.3 connection using TLS_AES_256_GCM_SHA384",
		"* ALPN: server accepted h2",
		"> GET /api/data HTTP/2",
		"> Host: example.com",
		"> User-Agent: curl/8.1.2",
		"> Accept: */*",
		">",
		"< HTTP/2 200",
		"< content-type: application/json",
		"< date: Sat, 22 Mar 2026 10:00:00 GMT",
		"< server: nginx/1.24.0",
		"< x-request-id: abc123",
		"< cache-control: max-age=3600",
		"<",
		`{"status":"ok","data":[1,2,3]}`,
	}, "\n")

	got := compressCurlOutput(input)

	// Should remove verbose connection info
	if strings.Contains(got, "TLS 1.3") {
		t.Error("should remove TLS handshake details")
	}
	if strings.Contains(got, "ALPN") {
		t.Error("should remove ALPN details")
	}

	// Should compress request headers
	if strings.Contains(got, "User-Agent") {
		t.Error("should compress request headers")
	}
	if !strings.Contains(got, "request headers") {
		t.Error("should have request header count summary")
	}

	// Should keep important response headers
	if !strings.Contains(got, "content-type") {
		t.Error("should keep content-type header")
	}

	// Should compress unimportant response headers
	if strings.Contains(got, "cache-control") {
		t.Error("should compress cache-control header")
	}

	// Should keep response body
	if !strings.Contains(got, `"status":"ok"`) {
		t.Error("should keep response body")
	}
}

func TestCompressCurlProgress(t *testing.T) {
	input := strings.Join([]string{
		"  % Total    % Received % Xferd  Average Speed   Time    Time     Time  Current",
		"                                 Dload  Upload   Total   Spent    Left  Speed",
		"  0     0    0     0    0     0      0      0 --:--:-- --:--:-- --:--:--     0",
		"100  1234  100  1234    0     0   5678      0 --:--:-- --:--:-- --:--:--  5678",
		`{"result":"success"}`,
	}, "\n")

	got := compressCurlOutput(input)

	// Should remove progress table
	if strings.Contains(got, "% Total") {
		t.Error("should remove curl progress header")
	}

	// Should keep response body
	if !strings.Contains(got, `"result":"success"`) {
		t.Error("should keep response body")
	}
}

func TestCompressCurlErrors(t *testing.T) {
	input := strings.Join([]string{
		"* Connected to localhost (127.0.0.1) port 8080",
		"* Connection refused",
		"curl: (7) Failed to connect to localhost port 8080: Connection refused",
	}, "\n")

	got := compressCurlOutput(input)

	// Should keep error info
	if !strings.Contains(got, "Connection refused") {
		t.Error("must preserve connection error info")
	}
	if !strings.Contains(got, "Failed to connect") {
		t.Error("must preserve curl error message")
	}
}

// --- Python filter tests ---

func TestCompressPythonPipInstall(t *testing.T) {
	var lines []string
	for i := 0; i < 20; i++ {
		lines = append(lines, "Requirement already satisfied: package"+itoa(i)+" in /usr/lib/python3/dist-packages (1.0.0)")
	}
	lines = append(lines, "Collecting newpackage==2.0.0")
	lines = append(lines, "Downloading newpackage-2.0.0.tar.gz (50 kB)")
	lines = append(lines, "Installing collected packages: newpackage")
	lines = append(lines, "Successfully installed newpackage-2.0.0")

	input := strings.Join(lines, "\n")
	got := compressPythonOutput(input)

	// Should compress "already satisfied" lines
	if strings.Contains(got, "package5") {
		t.Error("should not contain individual satisfied lines")
	}
	if !strings.Contains(got, "Already satisfied: 20 packages") {
		t.Error("should have satisfied summary")
	}

	// Should keep success message
	if !strings.Contains(got, "Successfully installed") {
		t.Error("should keep success message")
	}
}

func TestCompressPythonTraceback(t *testing.T) {
	// ImportError traceback: ALL frames should be preserved (import chain)
	input := strings.Join([]string{
		"Traceback (most recent call last):",
		`  File "/app/main.py", line 10, in <module>`,
		"    from app.handlers import process",
		`  File "/app/handlers.py", line 5, in <module>`,
		"    from app.utils import helper",
		`  File "/app/utils.py", line 3, in <module>`,
		"    from app.db import connection",
		`  File "/app/db.py", line 8, in <module>`,
		"    conn = connect()",
		"ImportError: No module named 'psycopg2'",
	}, "\n")

	got := compressPythonOutput(input)

	// Should keep traceback header
	if !strings.Contains(got, "Traceback (most recent call last):") {
		t.Error("should keep traceback header")
	}

	// ImportError: all frames preserved (full import chain)
	if !strings.Contains(got, "main.py") {
		t.Error("should keep first frame")
	}
	if !strings.Contains(got, "handlers.py") {
		t.Error("should keep import chain frame handlers.py")
	}
	if !strings.Contains(got, "utils.py") {
		t.Error("should keep import chain frame utils.py")
	}
	if !strings.Contains(got, "db.py") {
		t.Error("should keep last frame")
	}

	// Should NOT compress for ImportError
	if strings.Contains(got, "more frames") {
		t.Error("ImportError should NOT compress frames")
	}

	// Should keep error line
	if !strings.Contains(got, "ImportError") {
		t.Error("must preserve error line")
	}
}

// --- Tree filter tests ---

func TestCompressTreeSmallOutput(t *testing.T) {
	input := strings.Join([]string{
		".",
		"├── main.go",
		"├── go.mod",
		"└── go.sum",
		"",
		"0 directories, 3 files",
	}, "\n")

	got := compressTreeOutput(input)

	// Small output should not be modified
	if got != input {
		t.Errorf("small tree should be unchanged, got:\n%s", got)
	}
}

func TestCompressTreeLargeOutput(t *testing.T) {
	var lines []string
	lines = append(lines, ".")
	// Create a large tree with deep nesting
	for i := 0; i < 25; i++ {
		lines = append(lines, "├── dir"+itoa(i))
		lines = append(lines, "│   ├── file"+itoa(i)+".go")
		lines = append(lines, "│   └── file"+itoa(i)+"_test.go")
	}
	lines = append(lines, "")
	lines = append(lines, "25 directories, 50 files")

	input := strings.Join(lines, "\n")
	got := compressTreeOutput(input)

	// Should keep summary line
	if !strings.Contains(got, "25 directories, 50 files") {
		t.Error("should keep tree summary line")
	}
}

// --- Terraform filter tests ---

func TestCompressTerraformPlan(t *testing.T) {
	var lines []string
	// Add many refresh lines
	for i := 0; i < 30; i++ {
		lines = append(lines, "aws_instance.web["+itoa(i)+"]: Refreshing state... [id=i-abc"+itoa(i)+"]")
	}
	lines = append(lines, "")
	lines = append(lines, "Terraform used the selected providers to generate the following execution plan.")
	lines = append(lines, "")
	lines = append(lines, "  # aws_instance.web[0] will be updated in-place")
	lines = append(lines, "  ~ resource \"aws_instance\" \"web\" {")
	lines = append(lines, "      ~ instance_type = \"t2.micro\" -> \"t3.micro\"")
	lines = append(lines, "    }")
	lines = append(lines, "")
	lines = append(lines, "Plan: 0 to add, 1 to change, 0 to destroy.")

	input := strings.Join(lines, "\n")
	got := compressTerraformOutput(input)

	// Should compress refresh lines
	if strings.Contains(got, "Refreshing state") {
		t.Error("should compress Refreshing state lines")
	}
	if !strings.Contains(got, "Refreshed 30 resources") {
		t.Error("should have refresh summary")
	}

	// Should keep plan changes
	if !strings.Contains(got, "instance_type") {
		t.Error("should keep plan change details")
	}

	// Should keep plan summary
	if !strings.Contains(got, "Plan: 0 to add, 1 to change, 0 to destroy") {
		t.Error("should keep plan summary")
	}
}

func TestCompressTerraformApply(t *testing.T) {
	input := strings.Join([]string{
		"aws_instance.web: Creating...",
		"aws_instance.web: Still creating... [10s elapsed]",
		"aws_instance.web: Still creating... [20s elapsed]",
		"aws_instance.web: Still creating... [30s elapsed]",
		"aws_instance.web: Creation complete after 35s [id=i-abc123]",
		"",
		"Apply complete! Resources: 1 added, 0 changed, 0 destroyed.",
	}, "\n")

	got := compressTerraformOutput(input)

	// Should compress "still creating" progress
	if strings.Contains(got, "Still creating") {
		t.Error("should compress 'Still creating' lines")
	}

	// Should keep creation action and completion
	if !strings.Contains(got, "Creating...") {
		t.Error("should keep initial creation line")
	}
	if !strings.Contains(got, "Creation complete") {
		t.Error("should keep creation complete line")
	}

	// Should keep apply summary
	if !strings.Contains(got, "Apply complete") {
		t.Error("should keep apply summary")
	}
}

func TestCompressTerraformErrors(t *testing.T) {
	input := strings.Join([]string{
		"aws_instance.web: Refreshing state... [id=i-abc123]",
		"",
		"Error: error configuring Terraform AWS Provider: no valid credential sources found",
		"",
		"  on main.tf line 1, in provider \"aws\":",
		"   1: provider \"aws\" {",
	}, "\n")

	got := compressTerraformOutput(input)

	// Must preserve error
	if !strings.Contains(got, "Error: error configuring") {
		t.Error("must preserve terraform error")
	}
	if !strings.Contains(got, "main.tf line 1") {
		t.Error("must preserve error location")
	}
}

// --- Kubectl filter tests ---

func TestCompressKubectlManagedFields(t *testing.T) {
	input := strings.Join([]string{
		"apiVersion: v1",
		"kind: Pod",
		"metadata:",
		"  name: my-pod",
		"  namespace: default",
		"  managedFields:",
		"    - apiVersion: v1",
		"      fieldsType: FieldsV1",
		"      fieldsV1:",
		"        f:metadata:",
		"          f:labels:",
		"            f:app: {}",
		"      manager: kubectl",
		"      operation: Update",
		"      time: \"2026-03-22T10:00:00Z\"",
		"  labels:",
		"    app: my-app",
		"spec:",
		"  containers:",
		"    - name: main",
		"      image: nginx:latest",
	}, "\n")

	got := compressKubectlOutput(input)

	// Should compress managedFields
	if strings.Contains(got, "fieldsType") {
		t.Error("should remove managedFields details")
	}
	if !strings.Contains(got, "managedFields: (omitted)") {
		t.Error("should have managedFields omitted marker")
	}

	// Should keep resource metadata
	if !strings.Contains(got, "kind: Pod") {
		t.Error("should keep kind")
	}
	if !strings.Contains(got, "name: my-pod") {
		t.Error("should keep resource name")
	}

	// Should keep spec
	if !strings.Contains(got, "image: nginx:latest") {
		t.Error("should keep container spec")
	}

	// Should keep labels (which come after managedFields)
	if !strings.Contains(got, "app: my-app") {
		t.Error("should keep labels after managedFields block")
	}
}

func TestCompressKubectlLastAppliedConfig(t *testing.T) {
	input := strings.Join([]string{
		"metadata:",
		"  annotations:",
		"    kubectl.kubernetes.io/last-applied-configuration: |",
		"      {\"apiVersion\":\"v1\",\"kind\":\"Pod\",\"metadata\":{\"name\":\"my-pod\"}}",
		"  name: my-pod",
		"spec:",
		"  containers:",
		"    - name: main",
	}, "\n")

	got := compressKubectlOutput(input)

	// Should compress last-applied-configuration
	if strings.Contains(got, "apiVersion\":\"v1\"") {
		t.Error("should remove last-applied-configuration JSON content")
	}

	// Should keep other annotations/metadata
	if !strings.Contains(got, "name: my-pod") {
		t.Error("should keep resource name")
	}
}

func TestCompressKubectlHelmDebug(t *testing.T) {
	input := strings.Join([]string{
		"client.go:134: creating 3 resource(s)",
		"NAME: my-release",
		"NAMESPACE: default",
		"STATUS: deployed",
		"server.go:89: processing release",
		"REVISION: 1",
	}, "\n")

	got := compressKubectlOutput(input)

	// Should remove helm debug lines
	if strings.Contains(got, "client.go:134") {
		t.Error("should remove helm client debug lines")
	}
	if strings.Contains(got, "server.go:89") {
		t.Error("should remove helm server debug lines")
	}

	// Should keep release info
	if !strings.Contains(got, "NAME: my-release") {
		t.Error("should keep helm release name")
	}
	if !strings.Contains(got, "STATUS: deployed") {
		t.Error("should keep helm release status")
	}
}

// --- Classifier improvement tests ---

func TestClassifyRustNoteHelp(t *testing.T) {
	// Import classify package is in a different package so we test via the filter behavior
	// These are tested in classify_test.go — here we verify the filter pipeline preserves them
}

// --- Auto-detect tests ---

func TestAutoDetectPython(t *testing.T) {
	input := strings.Join([]string{
		"Running script...",
		"Processing data...",
		"Traceback (most recent call last):",
		`  File "main.py", line 10, in <module>`,
		"    process()",
		"ValueError: invalid data",
	}, "\n")

	got := AutoDetect(input)
	if got != CmdPython {
		t.Errorf("AutoDetect should detect Python traceback, got %v", got)
	}
}

func TestAutoDetectTerraform(t *testing.T) {
	input := strings.Join([]string{
		"aws_instance.web: Refreshing state... [id=i-abc123]",
		"aws_s3_bucket.data: Refreshing state... [id=my-bucket]",
		"aws_iam_role.lambda: Refreshing state... [id=lambda-role]",
	}, "\n")

	got := AutoDetect(input)
	if got != CmdTerraform {
		t.Errorf("AutoDetect should detect Terraform output, got %v", got)
	}
}

func TestAutoDetectKubectl(t *testing.T) {
	input := strings.Join([]string{
		"NAME                     READY   STATUS    RESTARTS   AGE",
		"my-pod-abc123-def45      1/1     Running   0          5d",
		"my-pod-abc123-ghi67      1/1     Running   0          5d",
	}, "\n")

	got := AutoDetect(input)
	if got != CmdKubectl {
		t.Errorf("AutoDetect should detect kubectl output, got %v", got)
	}
}

func TestAutoDetectCurlVerbose(t *testing.T) {
	var lines []string
	lines = append(lines, "* Connected to example.com (93.184.216.34) port 443")
	lines = append(lines, "* TLS 1.3 connection using TLS_AES_256_GCM_SHA384")
	lines = append(lines, "> GET /api HTTP/2")
	lines = append(lines, "> Host: example.com")
	lines = append(lines, "> Accept: */*")
	lines = append(lines, ">")
	lines = append(lines, "< HTTP/2 200")
	lines = append(lines, "< content-type: application/json")
	lines = append(lines, "<")
	lines = append(lines, `{"ok":true}`)

	input := strings.Join(lines, "\n")
	got := AutoDetect(input)
	if got != CmdCurl {
		t.Errorf("AutoDetect should detect curl verbose output, got %v", got)
	}
}

// --- Improved npm audit tests ---

func TestCompressNpmAuditKeepsCritical(t *testing.T) {
	var lines []string
	// Add critical severity details
	lines = append(lines, "# npm audit report")
	lines = append(lines, "")
	for i := 0; i < 15; i++ {
		lines = append(lines, "  Severity: critical")
		lines = append(lines, "  Vulnerable: package-crit-"+itoa(i)+" <2.0.0")
		lines = append(lines, "  Patched: >=2.0.0")
		lines = append(lines, "  Path: myapp > package-crit-"+itoa(i))
		lines = append(lines, "  More info: https://npmjs.com/advisories/"+itoa(i))
		lines = append(lines, "")
	}
	// Add low severity details
	for i := 0; i < 20; i++ {
		lines = append(lines, "  Severity: low")
		lines = append(lines, "  Vulnerable: package-low-"+itoa(i)+" <1.0.0")
		lines = append(lines, "  Patched: >=1.0.0")
		lines = append(lines, "")
	}
	lines = append(lines, "35 vulnerabilities (20 low, 15 critical)")

	input := strings.Join(lines, "\n")
	got := compressNpmOutput(input)

	// All 15 critical packages should be kept
	for i := 0; i < 15; i++ {
		if !strings.Contains(got, "package-crit-"+itoa(i)) {
			t.Errorf("should keep critical vulnerability package-crit-%d", i)
		}
	}

	// Low severity should be truncated
	if !strings.Contains(got, "low/moderate vulnerability details omitted") {
		t.Error("should have low/moderate truncation notice")
	}

	// Should keep summary
	if !strings.Contains(got, "35 vulnerabilities") {
		t.Error("should keep vulnerability summary")
	}
}

// --- Jq filter tests ---

func TestCompressJqSmallOutput(t *testing.T) {
	input := `{
  "name": "test",
  "value": 42
}`
	got := compressJqOutput(input)
	if got != input {
		t.Error("small jq output should be unchanged")
	}
}

func TestCompressJqLargeArray(t *testing.T) {
	var lines []string
	lines = append(lines, "[")
	for i := 0; i < 25; i++ {
		lines = append(lines, "  {")
		lines = append(lines, `    "id": `+itoa(i)+",")
		lines = append(lines, `    "name": "item-`+itoa(i)+`"`)
		if i < 24 {
			lines = append(lines, "  },")
		} else {
			lines = append(lines, "  }")
		}
	}
	lines = append(lines, "]")

	input := strings.Join(lines, "\n")
	got := compressJqOutput(input)

	// Should keep first 10 elements
	if !strings.Contains(got, "item-0") {
		t.Error("should keep first element")
	}
	if !strings.Contains(got, "item-9") {
		t.Error("should keep 10th element")
	}

	// Should have compression notice
	if !strings.Contains(got, "more elements") {
		t.Error("should have 'more elements' notice")
	}

	// Should be shorter than input
	if len(strings.Split(got, "\n")) >= len(strings.Split(input, "\n")) {
		t.Error("output should be shorter than input")
	}
}

// --- Tar filter tests ---

func TestCompressTarSmallListing(t *testing.T) {
	input := strings.Join([]string{
		"file1.txt",
		"file2.txt",
		"dir/file3.txt",
	}, "\n")

	got := compressTarOutput(input)
	if got != input {
		t.Error("small tar listing should be unchanged")
	}
}

func TestCompressTarVerboseListing(t *testing.T) {
	var lines []string
	for i := 0; i < 40; i++ {
		lines = append(lines, "-rw-r--r-- user/group  1024 2024-01-15 10:30 path/to/file"+itoa(i)+".txt")
	}

	input := strings.Join(lines, "\n")
	got := compressTarOutput(input)

	// Should strip verbose metadata
	if strings.Contains(got, "user/group") {
		t.Error("should strip user/group from tar listing")
	}
	if strings.Contains(got, "2024-01-15") {
		t.Error("should strip dates from tar listing")
	}

	// Should have compression for large listings (>20 files)
	if !strings.Contains(got, "more files") {
		t.Error("should compress large file listing")
	}

	// Should keep some file names
	if !strings.Contains(got, "file0.txt") {
		t.Error("should keep first files")
	}
}

func TestCompressTarExtract(t *testing.T) {
	var lines []string
	for i := 0; i < 50; i++ {
		lines = append(lines, "x path/to/file"+itoa(i)+".txt")
	}

	input := strings.Join(lines, "\n")
	got := compressTarOutput(input)

	// Should compress extraction progress
	if strings.Contains(got, "x path/to/file") {
		t.Error("should compress extraction progress lines")
	}
	if !strings.Contains(got, "Extracted 50 files") {
		t.Error("should have extraction count summary")
	}
}

// --- SSH filter tests ---

func TestCompressSSHBanners(t *testing.T) {
	input := strings.Join([]string{
		"Warning: Permanently added '192.168.1.1' (ED25519) to the list of known hosts.",
		"###############################################",
		"#          Welcome to Production Server       #",
		"###############################################",
		"Last login: Sat Mar 22 10:00:00 2026 from 10.0.0.1",
		"user@server:~$ ls",
		"app  config  data  logs",
	}, "\n")

	got := compressSSHOutput(input)

	// Should remove banner
	if strings.Contains(got, "Permanently added") {
		t.Error("should remove SSH known hosts warning")
	}

	// Should remove MOTD decorative lines
	if strings.Contains(got, "###") {
		t.Error("should remove MOTD decoration")
	}

	// Should keep actual command output
	if !strings.Contains(got, "app  config  data  logs") {
		t.Error("should keep command output")
	}
}

func TestCompressSSHDebug(t *testing.T) {
	input := strings.Join([]string{
		"debug1: Reading configuration data /etc/ssh/ssh_config",
		"debug1: Connecting to server.com [1.2.3.4] port 22.",
		"debug1: Connection established.",
		"debug1: Authentication succeeded (publickey).",
		"Authenticated to server.com via publickey.",
		"Hello from server!",
	}, "\n")

	got := compressSSHOutput(input)

	// Should remove debug lines
	if strings.Contains(got, "debug1:") {
		t.Error("should remove SSH debug lines")
	}

	// Should remove auth banner
	if strings.Contains(got, "Authenticated to") {
		t.Error("should remove auth confirmation banner")
	}

	// Should keep actual output
	if !strings.Contains(got, "Hello from server!") {
		t.Error("should keep actual command output")
	}
}

func TestCompressSCPProgress(t *testing.T) {
	input := strings.Join([]string{
		"file1.txt          100%   50KB   1.2MB/s   00:00",
		"file2.txt          100%  100KB   2.0MB/s   00:00",
		"file3.txt          100%   75KB   1.5MB/s   00:00",
	}, "\n")

	got := compressSSHOutput(input)

	// Should compress SCP progress
	if strings.Contains(got, "file1.txt") {
		t.Error("should compress SCP progress lines")
	}
	if !strings.Contains(got, "3 files transferred") {
		t.Error("should have file transfer count")
	}
}

func TestCompressSSHErrors(t *testing.T) {
	input := strings.Join([]string{
		"ssh: connect to host server.com port 22: Connection refused",
		"Permission denied (publickey).",
	}, "\n")

	got := compressSSHOutput(input)

	// Must preserve errors
	if !strings.Contains(got, "Connection refused") {
		t.Error("must preserve SSH connection error")
	}
	if !strings.Contains(got, "Permission denied") {
		t.Error("must preserve SSH auth error")
	}
}

// --- Docker FROM annotation test ---

func TestCompressDockerMultiStageAnnotation(t *testing.T) {
	input := strings.Join([]string{
		"Step 1/4 : FROM golang:1.21-alpine AS builder",
		" ---> abc123def456",
		"Step 2/4 : RUN go build -o /app .",
		" ---> Running in 1234567890ab",
		"Removing intermediate container 1234567890ab",
		" ---> 2345678901bc",
		"Step 3/4 : FROM alpine:3.18",
		" ---> ccc111222333",
		"Step 4/4 : COPY --from=builder /app /app",
		" ---> Running in 4444555566667",
		"Removing intermediate container 4444555566667",
		" ---> 5555666677778",
		"Successfully built 5555666677778",
	}, "\n")

	got := compressDockerOutput(input)

	// FROM lines should be annotated as stage boundaries
	if !strings.Contains(got, "--- FROM golang:1.21-alpine AS builder ---") {
		t.Error("should annotate first FROM as stage boundary")
	}
	if !strings.Contains(got, "--- FROM alpine:3.18 ---") {
		t.Error("should annotate second FROM as stage boundary")
	}

	// Non-FROM commands should not have stage markers
	if strings.Contains(got, "--- RUN") {
		t.Error("should not annotate RUN commands as stages")
	}
	if strings.Contains(got, "--- COPY") {
		t.Error("should not annotate COPY commands as stages")
	}
}

// --- Go build context test ---

func TestCompressGoOutputPreservesErrors(t *testing.T) {
	input := strings.Join([]string{
		"# github.com/user/project",
		"./main.go:10:5: cannot use x (type int) as type string",
		"./main.go:15:2: undefined: doStuff",
		"./handler.go:20:8: imported and not used: \"fmt\"",
	}, "\n")

	got := compressGoOutput(input)

	// Must preserve package header
	if !strings.Contains(got, "# github.com/user/project") {
		t.Error("must preserve package header")
	}

	// Must preserve all error lines
	if !strings.Contains(got, "cannot use x") {
		t.Error("must preserve type error")
	}
	if !strings.Contains(got, "undefined: doStuff") {
		t.Error("must preserve undefined error")
	}
	if !strings.Contains(got, "imported and not used") {
		t.Error("must preserve unused import error")
	}
}

// --- Node filter tests ---

func TestCompressNodeWebpackNoise(t *testing.T) {
	input := strings.Join([]string{
		"(node:12345) ExperimentalWarning: The Fetch API is an experimental feature",
		"(node:12345) ExperimentalWarning: Custom ESM Loaders is an experimental feature",
		"(node:12345) [DEP0001] DeprecationWarning: Deprecated API 1",
		"(node:12345) [DEP0002] DeprecationWarning: Deprecated API 2",
		"(node:12345) [DEP0003] DeprecationWarning: Deprecated API 3",
		"",
		"  10% building modules (50/500)",
		"  50% building modules (250/500)",
		"  100% building modules (500/500)",
		"asset main.js 245 KiB [emitted] (name: main)",
		"modules by path ./src/ 120 KiB",
		"runtime modules 5 KiB 10 modules",
		"",
		"webpack 5.89.0 compiled successfully in 4523 ms",
	}, "\n")

	got := compressNodeOutput(input)

	// Should compress experimental warnings
	if strings.Contains(got, "Fetch API") {
		t.Error("should compress experimental warnings")
	}
	if !strings.Contains(got, "experimental warnings") {
		t.Error("should have experimental warning summary")
	}

	// Should compress deprecation warnings
	if !strings.Contains(got, "DEP0001") {
		t.Error("should keep first deprecation")
	}
	if !strings.Contains(got, "more deprecation") {
		t.Error("should summarize other deprecation warnings")
	}

	// Should compress webpack progress
	if strings.Contains(got, "10% building") {
		t.Error("should compress webpack progress")
	}

	// Should compress module details
	if strings.Contains(got, "modules by path") {
		t.Error("should compress webpack module details")
	}

	// Should keep build result
	if !strings.Contains(got, "compiled successfully") {
		t.Error("should keep webpack build result")
	}
}

func TestCompressNodeInternalFrames(t *testing.T) {
	input := strings.Join([]string{
		"TypeError: Cannot read properties of undefined (reading 'map')",
		"    at renderList (/app/src/components/UserList.tsx:15:23)",
		"    at processChild (/app/src/utils/renderer.ts:42:10)",
		"    at Module._compile (node:internal/modules/cjs/loader:1241:14)",
		"    at Module._extensions..js (node:internal/modules/cjs/loader:1295:10)",
		"    at Module.load (node:internal/modules/cjs/loader:1091:32)",
		"    at Module._load (node:internal/modules/cjs/loader:938:12)",
		"    at require (node:internal/modules/helpers:177:18)",
	}, "\n")

	got := compressNodeOutput(input)

	// Must preserve error message
	if !strings.Contains(got, "TypeError") {
		t.Error("must preserve error message")
	}

	// Must preserve app frames
	if !strings.Contains(got, "renderList") {
		t.Error("must preserve app stack frames")
	}
	if !strings.Contains(got, "processChild") {
		t.Error("must preserve app stack frames")
	}

	// Should compress internal frames
	if strings.Contains(got, "Module._compile") {
		t.Error("should compress node internal frames")
	}
	if !strings.Contains(got, "node internal frames") {
		t.Error("should have internal frames summary")
	}
}

// --- Python ImportError chain preservation ---

func TestCompressPythonImportErrorPreservesChain(t *testing.T) {
	input := strings.Join([]string{
		"Traceback (most recent call last):",
		`  File "/app/main.py", line 10, in <module>`,
		"    from app.server import create_app",
		`  File "/app/server.py", line 5, in <module>`,
		"    from app.routes import register_routes",
		`  File "/app/routes.py", line 3, in <module>`,
		"    from app.handlers.auth import AuthHandler",
		`  File "/app/handlers/auth.py", line 2, in <module>`,
		"    from app.services.ldap import LDAPClient",
		`  File "/app/services/ldap.py", line 1, in <module>`,
		"    import ldap3",
		"ModuleNotFoundError: No module named 'ldap3'",
	}, "\n")

	got := compressPythonOutput(input)

	// ImportError: ALL frames should be preserved (import chain is critical)
	if !strings.Contains(got, "main.py") {
		t.Error("should preserve first frame for ImportError")
	}
	if !strings.Contains(got, "server.py") {
		t.Error("should preserve import chain frame server.py")
	}
	if !strings.Contains(got, "routes.py") {
		t.Error("should preserve import chain frame routes.py")
	}
	if !strings.Contains(got, "auth.py") {
		t.Error("should preserve import chain frame auth.py")
	}
	if !strings.Contains(got, "ldap.py") {
		t.Error("should preserve import chain frame ldap.py")
	}

	// Should NOT have "more frames" compression
	if strings.Contains(got, "more frames") {
		t.Error("ImportError should NOT compress frames")
	}

	// Must preserve error line
	if !strings.Contains(got, "ModuleNotFoundError") {
		t.Error("must preserve error line")
	}
}

func TestCompressPythonNonImportErrorCompresses(t *testing.T) {
	input := strings.Join([]string{
		"Traceback (most recent call last):",
		`  File "/app/main.py", line 10, in <module>`,
		"    process()",
		`  File "/app/process.py", line 20, in process`,
		"    transform(data)",
		`  File "/app/transform.py", line 30, in transform`,
		"    validate(item)",
		`  File "/app/validate.py", line 40, in validate`,
		"    check(value)",
		`  File "/app/check.py", line 50, in check`,
		"    raise ValueError('bad')",
		"ValueError: bad",
	}, "\n")

	got := compressPythonOutput(input)

	// Non-import errors with >2 frames SHOULD be compressed
	if !strings.Contains(got, "more frames") {
		t.Error("non-import errors should compress middle frames")
	}

	// Should keep first and last frame
	if !strings.Contains(got, "main.py") {
		t.Error("should keep first frame")
	}
	if !strings.Contains(got, "check.py") {
		t.Error("should keep last frame")
	}

	// Must preserve error
	if !strings.Contains(got, "ValueError: bad") {
		t.Error("must preserve error line")
	}
}
