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

## Command-Specific Filters

These are selected based on command detection. They live in `internal/detect/detect.go`. See [architecture.md](architecture.md#command-detection) for how detection works.

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
ok  	github.com/fMusic/GoTK/internal/filter	0.003s
ok  	github.com/fMusic/GoTK/internal/detect	0.002s
ok  	github.com/fMusic/GoTK/internal/exec	0.001s
FAIL	github.com/fMusic/GoTK/cmd/gotk	0.005s
```

**After:**
```
  3 packages passed
FAIL	github.com/fMusic/GoTK/cmd/gotk	0.005s
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
