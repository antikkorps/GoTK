# Filter Catalog

All filters follow the `func(string) string` signature. For how they are composed, see [docs/architecture.md](architecture.md#filter-chain-pattern).

## Generic Filters

These run on every invocation, regardless of command type.

### StripANSI

**File:** `internal/filter/ansi.go`

Removes ANSI escape sequences — color codes, cursor movement, terminal control characters.

**Before:**
```
[32m./cmd/gotk/main.go[0m:[36m17[0m:func main() {
```

**After:**
```
./cmd/gotk/main.go:17:func main() {
```

### NormalizeWhitespace

**File:** `internal/filter/whitespace.go`

- Strips trailing whitespace from every line
- Collapses 3+ consecutive blank lines into 1
- Trims leading/trailing whitespace from the entire output

**Before:**
```
result one





result two
```

**After:**
```
result one

result two
```

### Dedup

**File:** `internal/filter/dedup.go`

Collapses runs of consecutive identical lines into the first occurrence plus a count marker.

**Before:**
```
  PASS
  PASS
  PASS
  PASS
  PASS
  FAIL: TestBroken
```

**After:**
```
  PASS
  ... (4 duplicate lines)
  FAIL: TestBroken
```

### CompressPaths

**File:** `internal/filter/paths.go`

Replaces the current working directory with `./` and the home directory with `~/` in all paths.

**Before:**
```
/Users/dev/projects/GoTK/internal/filter/ansi.go
/Users/dev/projects/GoTK/internal/filter/chain.go
```

**After** (when cwd is `/Users/dev/projects/GoTK`):
```
./internal/filter/ansi.go
./internal/filter/chain.go
```

### TrimEmpty

**File:** `internal/filter/trim.go`

Removes decorative separator lines — lines consisting entirely of repeated `-`, `=`, `_`, `~`, `*`, `#`, or `+` characters (3+ chars long). Preserves blank lines and content lines.

**Before:**
```
Test Results
============
PASS: 5
FAIL: 0
-----------
Done.
```

**After:**
```
Test Results
PASS: 5
FAIL: 0
Done.
```

### Truncate

**File:** `internal/filter/truncate.go`

Caps output at `--max-lines` (default 50). When truncating, keeps 70% from the head and 30% from the tail, with an omission marker in between. Disabled by `--no-truncate`.

**Before** (200 lines):
```
line 1
line 2
...
line 200
```

**After** (with default 50 max):
```
line 1
line 2
...
line 35

[... 150 lines omitted ...]

line 186
...
line 200
```

## Stack Trace Filters

These run on all output (generic), since panics and tracebacks can appear in any command's output.

### CompressStackTraces

**File:** `internal/filter/stacktrace.go`

Condenses repetitive stack traces for Go, Python, and Node.js. Keeps the error cause and top frame(s), collapses the rest.

**Go stack trace before:**
```
goroutine 1 [running]:
main.handler(0xc0000b2000)
	/app/server.go:45 +0x1a0
main.middleware(0xc0000b2000)
	/app/middleware.go:22 +0x80
main.router(0xc0000b2000)
	/app/router.go:15 +0x60
net/http.(*ServeMux).ServeHTTP(...)
	/usr/local/go/src/net/http/server.go:2424
net/http.serverHandler.ServeHTTP(...)
	/usr/local/go/src/net/http/server.go:2938
```

**After:**
```
goroutine 1 [running]:
main.handler(0xc0000b2000)
	/app/server.go:45 +0x1a0
  ... (4 frames collapsed)
```

**Python traceback** and **Node.js stack traces** follow the same pattern: keep the cause and top frames, collapse deep internal frames.

### RedactSecrets

**File:** `internal/filter/secrets.go`

Detects and redacts sensitive values: API keys, bearer tokens, JWTs, AWS credentials, private keys, and password patterns. Replaces with `[REDACTED]`.

### Summarize

**File:** `internal/filter/summarize.go`

For large outputs, prepends a structured summary with error/warning counts, file paths mentioned in errors, and key error lines. Helps LLMs quickly understand the output before reading details.

## Command-Specific Filters

These are selected based on command detection. They live in `internal/detect/`. See [architecture.md](architecture.md#command-detection) for how detection works.

### Grep: `compressGrepOutput`

**Applies to:** grep, rg, ag, ack

Groups results by file. Strips the repeated filename prefix from each line and emits a file header instead. Also strips line numbers from `file:linenum:content` format.

**Before:**
```
./internal/filter/ansi.go:9:func StripANSI(input string) string {
./internal/filter/ansi.go:10:	return ansiPattern.ReplaceAllString(input, "")
./internal/filter/chain.go:7:type Chain struct {
./internal/filter/chain.go:22:func (c *Chain) Apply(input string) string {
```

**After:**
```
>> ./internal/filter/ansi.go
  func StripANSI(input string) string {
  	return ansiPattern.ReplaceAllString(input, "")

>> ./internal/filter/chain.go
  type Chain struct {
  func (c *Chain) Apply(input string) string {
```

### Find: `compressFindOutput`

**Applies to:** find, fd

Factors out the common path prefix and groups files by immediate subdirectory.

**Before:**
```
./internal/filter/ansi.go
./internal/filter/chain.go
./internal/filter/dedup.go
./internal/detect/detect.go
./internal/detect/autodetect.go
```

**After:**
```
[base: ./internal/]
filter/ansi.go
filter/chain.go
filter/dedup.go
detect/detect.go
detect/autodetect.go
```

### Git: `compressGitOutput`

**Applies to:** git, gh

- **Diff output:** Strips verbose diff headers (`diff --git`, `index`, `old mode`, `new mode`, `---`, `+++`) and replaces them with a single `>> filename` header per file
- **Log output:** Strips `Author:` and `Date:` lines, keeping only commit hashes and messages

**Before (diff):**
```
diff --git a/main.go b/main.go
index abc1234..def5678 100644
--- a/main.go
+++ b/main.go
@@ -10,3 +10,4 @@
 import "fmt"
+import "os"
```

**After:**
```
>> main.go
@@ -10,3 +10,4 @@
 import "fmt"
+import "os"
```

**Before (log):**
```
commit a1b2c3d4e5f6
Author: Dev <dev@example.com>
Date:   Mon Mar 10 14:30:00 2026

    Fix filter chain ordering
```

**After:**
```
commit a1b2c3d4e5f6

    Fix filter chain ordering
```

### Go: `compressGoOutput`

**Applies to:** go (test, build, etc.)

Collapses consecutive `ok` (passing package) lines into a summary count. Preserves `FAIL` lines and actual test output verbatim.

**Before:**
```
ok  	github.com/antikkorps/GoTK/internal/filter	0.003s
ok  	github.com/antikkorps/GoTK/internal/detect	0.002s
ok  	github.com/antikkorps/GoTK/internal/exec	0.001s
FAIL	github.com/antikkorps/GoTK/cmd/gotk	0.005s
```

**After:**
```
  3 packages passed
FAIL	github.com/antikkorps/GoTK/cmd/gotk	0.005s
```

### Ls: `compressLsOutput`

**Applies to:** ls, exa, eza, lsd

Strips verbose metadata from long-format (`ls -la`) output. Removes the `total` line and reduces each entry to permissions, size, and name — dropping user, group, date, and time fields.

**Before:**
```
total 48
drwxr-xr-x  12 dev  staff   384 Mar 10 14:30 .
drwxr-xr-x   5 dev  staff   160 Mar  9 10:00 ..
-rw-r--r--   1 dev  staff  1234 Mar 10 14:30 main.go
-rw-r--r--   1 dev  staff   567 Mar 10 14:28 go.mod
```

**After:**
```
drwxr-xr-x 384 .
drwxr-xr-x 160 ..
-rw-r--r-- 1234 main.go
-rw-r--r-- 567 go.mod
```

### Docker: `compressDockerOutput`

**Applies to:** docker, docker-compose, podman

Strips pull progress bars (`Pulling fs layer`, `Downloading`, `Extracting`), build step prefixes, and redundant layer hashes. Preserves build errors, warnings, and final image IDs.

**Before:**
```
Step 1/5 : FROM node:18-alpine
 ---> abc1234def56
Step 2/5 : COPY package.json .
 ---> Using cache
 ---> 789abc012def
Downloading [========>          ] 45.2MB/120MB
Downloading [===========>       ] 67.8MB/120MB
Successfully built abc1234def56
```

**After:**
```
Step 1/5 : FROM node:18-alpine
Step 2/5 : COPY package.json .
Successfully built abc1234def56
```

### Npm: `compressNpmOutput`

**Applies to:** npm, yarn, pnpm, npx, bun

Strips install progress, download counts, deprecation warnings (kept as summary), and audit noise. Preserves errors and the final install summary.

**Before:**
```
npm warn deprecated inflight@1.0.6: This module is not supported
npm warn deprecated glob@7.2.3: Glob versions prior to v9 are no longer supported
added 847 packages, and audited 848 packages in 12s
found 0 vulnerabilities
```

**After:**
```
added 847 packages, and audited 848 packages in 12s
found 0 vulnerabilities
```

### Cargo: `compressCargoOutput`

**Applies to:** cargo, rustc

Collapses `Compiling` and `Downloading` lines into summary counts. Preserves all error and warning messages verbatim.

**Before:**
```
   Compiling serde v1.0.152
   Compiling tokio v1.28.0
   Compiling hyper v0.14.26
   Compiling my-project v0.1.0
    Finished dev [unoptimized + debuginfo] target(s) in 15.3s
```

**After:**
```
  compiled 4 crates
    Finished dev [unoptimized + debuginfo] target(s) in 15.3s
```

### Make: `compressMakeOutput`

**Applies to:** make, cmake, ninja

Strips `make[N]: Entering/Leaving directory` messages and collapses verbose compiler invocations (gcc/g++/cc with many flags) into a short form. Preserves all errors and warnings.

**Before:**
```
make[1]: Entering directory '/src/lib'
gcc -Wall -Wextra -O2 -fPIC -I../include -I/usr/local/include -DNDEBUG -c -o lib.o lib.c
make[1]: Leaving directory '/src/lib'
make[1]: Entering directory '/src/main'
gcc -Wall -Wextra -O2 -I../include -c -o main.o main.c
make[1]: Leaving directory '/src/main'
```

**After:**
```
gcc ... -o lib.o lib.c
gcc ... -o main.o main.c
```

### Curl: `compressCurlOutput`

**Applies to:** curl, wget, http (httpie)

Strips progress bars and compresses verbose headers. Keeps response body, HTTP status, important headers (content-type, location), and error messages. Request headers are summarized by count.

**Before:**
```
* Connected to api.example.com (93.184.216.34) port 443
* TLS 1.3 connection using TLS_AES_256_GCM_SHA384
> GET /api/users HTTP/2
> Host: api.example.com
> User-Agent: curl/8.1.2
> Accept: application/json
>
< HTTP/2 200
< content-type: application/json
< date: Sat, 22 Mar 2026 10:00:00 GMT
< cache-control: max-age=60
<
{"users":[{"id":1,"name":"Alice"}]}
```

**After:**
```
> (4 request headers)
< content-type: application/json
< (2 other headers)
{"users":[{"id":1,"name":"Alice"}]}
```

### Python: `compressPythonOutput`

**Applies to:** python, python3, pip, pip3

Compresses pip install noise (`Requirement already satisfied` spam, download progress). Condenses Python tracebacks: keeps first frame, last frame, and error line, compresses middle frames.

**Before:**
```
Requirement already satisfied: flask in /usr/lib/... (2.3.0)
Requirement already satisfied: requests in /usr/lib/... (2.31.0)
... (8 more)
Collecting sqlalchemy==2.0.21
Successfully installed sqlalchemy-2.0.21
```

**After:**
```
Already satisfied: 10 packages
pip: 1 packages downloaded/installed
Successfully installed sqlalchemy-2.0.21
```

### Tree: `compressTreeOutput`

**Applies to:** tree

Compresses deep directory chains (single-child directories collapsed into one path). Small trees (<20 lines) are kept as-is. Summary line always preserved.

### Terraform: `compressTerraformOutput`

**Applies to:** terraform, tofu, tf

Strips `Refreshing state...` lines (summarized by count), `Still creating/modifying...` progress lines, and `Read complete` lines. Preserves plan changes, error messages, and the plan summary.

**Before:**
```
aws_instance.web[0]: Refreshing state... [id=i-abc123]
aws_instance.web[1]: Refreshing state... [id=i-def456]
... (10 more resources)

  # aws_instance.web[0] will be updated in-place
  ~ instance_type = "t3.micro" -> "t3.small"

Plan: 0 to add, 1 to change, 0 to destroy.
```

**After:**
```
Refreshed 12 resources

  # aws_instance.web[0] will be updated in-place
  ~ instance_type = "t3.micro" -> "t3.small"

Plan: 0 to add, 1 to change, 0 to destroy.
```

### Kubectl: `compressKubectlOutput`

**Applies to:** kubectl, helm, k9s, oc

Strips `managedFields` blocks (replaced with `(omitted)`), `last-applied-configuration` annotations, and helm debug lines. Preserves resource specs, status, labels, and conditions.

**Before:** A 40-line Pod YAML with `managedFields` and `last-applied-configuration`.

**After:** 28-line Pod YAML with `managedFields: (omitted)` and annotation replaced.

### Jq: `compressJqOutput`

**Applies to:** jq, yq, gojq

Compresses large JSON array output by keeping the first 10 elements and summarizing the rest. Small outputs (<50 lines) are kept as-is.

### Tar: `compressTarOutput`

**Applies to:** tar, zip, unzip, gzip, 7z

Strips verbose metadata (permissions, user/group, dates) from listings. For large listings (>20 files), keeps first 10 and last 5 files with a count summary. Compresses extraction progress (`x file`) into a count.

**Before:**
```
-rw-r--r-- user/group  1024 2024-01-15 10:30 project/main.go
-rw-r--r-- user/group  2048 2024-01-15 10:30 project/utils.go
```

**After:**
```
project/main.go
project/utils.go
```

### SSH: `compressSSHOutput`

**Applies to:** ssh, scp, sftp, rsync

Strips SSH connection banners (`Warning: Permanently added`), debug lines, host key fingerprints, and MOTD decorations. Compresses SCP progress lines into a file count. Preserves actual command output and errors.
