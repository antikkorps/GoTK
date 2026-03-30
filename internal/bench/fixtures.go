package bench

import (
	"fmt"
	"strings"
)

// generateGrepFixture generates 500 lines of grep -rn output across 20 files with repeated paths.
func generateGrepFixture() string {
	files := []string{
		"src/server/handler.go", "src/server/middleware.go", "src/server/routes.go",
		"src/db/postgres.go", "src/db/migrations.go", "src/db/models.go",
		"src/api/users.go", "src/api/auth.go", "src/api/products.go",
		"src/utils/helpers.go", "src/utils/validator.go", "src/utils/logger.go",
		"pkg/config/config.go", "pkg/config/defaults.go",
		"internal/cache/redis.go", "internal/cache/memory.go",
		"cmd/server/main.go", "cmd/worker/main.go",
		"test/integration/api_test.go", "test/integration/db_test.go",
	}
	patterns := []string{
		"func ", "return ", "if err != nil", "log.Printf",
		"TODO:", "import (", "type ", "defer ", "ctx context.Context",
		"json.Marshal", "http.Handler", "fmt.Sprintf",
	}

	var lines []string
	for i := 0; i < 500; i++ {
		file := files[i%len(files)]
		pattern := patterns[i%len(patterns)]
		lineNum := 10 + (i*3)%200
		lines = append(lines, fmt.Sprintf("%s:%d:\t%s%s // line content %d",
			file, lineNum, pattern, "SomeValue", i))
	}
	return strings.Join(lines, "\n") + "\n"
}

// generateGitLogFixture generates 50 commits with full Author/Date/message and --stat.
func generateGitLogFixture() string {
	var lines []string
	authors := []string{"Alice Smith", "Bob Jones", "Charlie Brown", "Diana Ross", "Eve Williams"}
	messages := []string{
		"feat: add user authentication module",
		"fix: resolve database connection leak",
		"refactor: extract validation logic",
		"docs: update API documentation",
		"chore: bump dependency versions",
		"fix: handle edge case in parser",
		"feat: implement caching layer",
		"test: add integration test coverage",
	}
	statFiles := []string{
		"src/auth/handler.go", "src/db/pool.go", "src/api/routes.go",
		"pkg/validator/rules.go", "internal/cache/store.go",
	}

	for i := 0; i < 50; i++ {
		hash := fmt.Sprintf("%040x", i*12345+67890)
		author := authors[i%len(authors)]
		msg := messages[i%len(messages)]
		lines = append(lines,
			fmt.Sprintf("commit %s", hash),
			fmt.Sprintf("Author: %s <%s@example.com>", author, strings.ToLower(strings.Split(author, " ")[0])),
			fmt.Sprintf("Date:   Mon Jan %d 10:%02d:00 2025 -0700", (i%28)+1, i%60),
			"",
			"    "+msg,
			"",
		)
		// Add --stat output
		numFiles := 2 + i%4
		for j := 0; j < numFiles; j++ {
			f := statFiles[(i+j)%len(statFiles)]
			insertions := 5 + (i*j+1)%50
			deletions := 1 + (i*j)%20
			plus := strings.Repeat("+", min(insertions, 30))
			minus := strings.Repeat("-", min(deletions, 10))
			lines = append(lines, fmt.Sprintf(" %s | %3d %s%s", f, insertions+deletions, plus, minus))
		}
		lines = append(lines, fmt.Sprintf(" %d files changed, %d insertions(+), %d deletions(-)", numFiles, 20+i*3, 5+i))
		lines = append(lines, "")
		// Add index metadata that should be filtered
		lines = append(lines, fmt.Sprintf("index %07x..%07x 100644", i*111, i*222+333))
	}
	return strings.Join(lines, "\n") + "\n"
}

// generateGitDiffFixture generates 10 file diffs with index/mode headers and 200 lines of changes.
func generateGitDiffFixture() string {
	files := []string{
		"src/main.go", "src/handler.go", "src/config.go", "src/db.go", "src/cache.go",
		"pkg/util.go", "pkg/errors.go", "internal/api.go", "cmd/server.go", "test/main_test.go",
	}

	var lines []string
	for i, file := range files {
		lines = append(lines,
			fmt.Sprintf("diff --git a/%s b/%s", file, file),
			fmt.Sprintf("index %07x..%07x 100644", i*1111, i*2222+3333),
			"old mode 100644",
			"new mode 100755",
			"similarity index 95%",
			fmt.Sprintf("--- a/%s", file),
			fmt.Sprintf("+++ b/%s", file),
		)
		// Generate hunks with 20 lines of changes per file
		for h := 0; h < 2; h++ {
			startLine := 10 + h*50
			lines = append(lines, fmt.Sprintf("@@ -%d,10 +%d,12 @@ func handler%d() {", startLine, startLine, i*2+h))
			for j := 0; j < 10; j++ {
				switch j % 5 {
				case 0:
					lines = append(lines, fmt.Sprintf("+\t// Added comment in %s line %d", file, j))
				case 1:
					lines = append(lines, fmt.Sprintf("-\t// Removed old comment line %d", j))
				case 2:
					lines = append(lines, fmt.Sprintf("+\tfmt.Println(\"new code %d\")", j))
				default:
					lines = append(lines, fmt.Sprintf(" \texistingCode(%d)", j))
				}
			}
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n") + "\n"
}

// generateGoTestFixture generates 30 packages: 25 ok, 3 FAIL with verbose output, 2 cached.
func generateGoTestFixture() string {
	var lines []string

	// 25 passing packages
	for i := 0; i < 25; i++ {
		pkg := fmt.Sprintf("github.com/myproject/pkg/module%d", i)
		duration := fmt.Sprintf("%.3fs", 0.1+float64(i)*0.05)
		lines = append(lines, fmt.Sprintf("ok  \t%s\t%s", pkg, duration))
	}

	// 3 failing packages with verbose output
	failPkgs := []string{"github.com/myproject/pkg/auth", "github.com/myproject/pkg/db", "github.com/myproject/pkg/api"}
	for _, pkg := range failPkgs {
		lines = append(lines,
			"=== RUN   TestMain",
			"=== RUN   TestMain/SubTest1",
			"    handler_test.go:45: expected 200, got 500",
			"    handler_test.go:46: response body: {\"error\": \"internal server error\"}",
			"--- FAIL: TestMain/SubTest1 (0.02s)",
			"=== RUN   TestMain/SubTest2",
			"    handler_test.go:60: context deadline exceeded",
			"--- FAIL: TestMain/SubTest2 (5.00s)",
			"--- FAIL: TestMain (5.02s)",
			"FAIL",
			fmt.Sprintf("FAIL\t%s\t5.031s", pkg),
		)
	}

	// 2 cached packages
	for i := 0; i < 2; i++ {
		pkg := fmt.Sprintf("github.com/myproject/pkg/utils%d", i)
		lines = append(lines, fmt.Sprintf("ok  \t%s\t(cached)", pkg))
	}

	return strings.Join(lines, "\n") + "\n"
}

// generateFindFixture generates 300 file paths in a deep directory structure.
func generateFindFixture() string {
	dirs := []string{
		"src/server/handlers", "src/server/middleware", "src/server/routes",
		"src/db/postgres/migrations", "src/db/redis/cache",
		"src/api/v1/controllers", "src/api/v2/controllers",
		"pkg/config/internal", "pkg/utils/helpers",
		"internal/auth/jwt", "internal/auth/oauth",
		"test/integration/api", "test/unit/handlers",
		"docs/api/schemas", "scripts/deploy/k8s",
	}
	extensions := []string{".go", ".go", ".go", ".yaml", ".json", ".sql", ".md", ".sh", ".toml", ".mod"}

	var lines []string
	for i := 0; i < 300; i++ {
		dir := "/home/user/project/" + dirs[i%len(dirs)]
		ext := extensions[i%len(extensions)]
		filename := fmt.Sprintf("file_%03d%s", i, ext)
		lines = append(lines, dir+"/"+filename)
	}
	return strings.Join(lines, "\n") + "\n"
}

// generateDockerBuildFixture generates a 20-step Dockerfile build with intermediate containers.
func generateDockerBuildFixture() string {
	var lines []string
	steps := []string{
		"FROM golang:1.21-alpine AS builder",
		"WORKDIR /app",
		"COPY go.mod go.sum ./",
		"RUN go mod download",
		"COPY . .",
		"RUN go build -o /server ./cmd/server",
		"FROM alpine:3.18",
		"RUN apk add --no-cache ca-certificates",
		"COPY --from=builder /server /server",
		"RUN adduser -D appuser",
		"USER appuser",
		"EXPOSE 8080",
		"ENV PORT=8080",
		"ENV GIN_MODE=release",
		"HEALTHCHECK CMD wget -q -O /dev/null http://localhost:8080/health",
		"RUN mkdir -p /data",
		"VOLUME /data",
		"COPY config.yaml /etc/app/config.yaml",
		"LABEL maintainer=dev@example.com",
		"ENTRYPOINT [\"/server\"]",
	}

	lines = append(lines, "Sending build context to Docker daemon  45.2MB")
	for i, step := range steps {
		containerID := fmt.Sprintf("%012x", i*0xabcdef+0x123456)
		hashID := fmt.Sprintf("%012x", i*0xfedcba+0x654321)
		lines = append(lines,
			fmt.Sprintf("Step %d/%d : %s", i+1, len(steps), step),
			fmt.Sprintf(" ---> Running in %s", containerID),
		)
		if strings.HasPrefix(step, "RUN") {
			lines = append(lines, fmt.Sprintf("Running command: %s", strings.TrimPrefix(step, "RUN ")))
		}
		lines = append(lines,
			fmt.Sprintf("Removing intermediate container %s", containerID),
			fmt.Sprintf(" ---> %s", hashID),
		)
	}
	lines = append(lines,
		fmt.Sprintf("Successfully built %s", "abc123def456"),
		"Successfully tagged myapp:latest",
	)
	return strings.Join(lines, "\n") + "\n"
}

// generateNpmInstallFixture generates 50 packages with deprecation warnings and progress.
func generateNpmInstallFixture() string {
	var lines []string

	packages := []string{
		"express", "lodash", "axios", "moment", "chalk", "debug", "commander",
		"inquirer", "glob", "minimist", "mkdirp", "rimraf", "semver", "uuid",
		"yargs", "cross-env", "dotenv", "cors", "helmet", "morgan",
	}

	// HTTP fetch lines
	for i := 0; i < 30; i++ {
		pkg := packages[i%len(packages)]
		lines = append(lines, fmt.Sprintf("npm http fetch GET 200 https://registry.npmjs.org/%s 50ms", pkg))
	}

	// Deprecation warnings
	deprecatedPkgs := []string{
		"request", "har-validator", "uuid@3.4.0", "querystring", "chokidar@2.1.8",
		"fsevents@1.2.13", "resolve-url", "urix", "source-map-resolve", "source-map-url",
		"mkdirp@0.5.5", "rimraf@2.7.1", "glob@7.2.3", "inflight", "npmlog",
	}
	for _, pkg := range deprecatedPkgs {
		lines = append(lines, fmt.Sprintf("npm warn deprecated %s: This package is no longer maintained", pkg))
	}

	// Timing lines
	for i := 0; i < 10; i++ {
		lines = append(lines, fmt.Sprintf("npm timing idealTree:buildDeps Completed in %dms", 100+i*50))
	}

	// Additional packages being added
	for i := 0; i < 50; i++ {
		pkg := packages[i%len(packages)]
		ver := fmt.Sprintf("%d.%d.%d", 1+i%5, i%10, i%20)
		lines = append(lines, fmt.Sprintf("added %s@%s", pkg, ver))
	}

	lines = append(lines, "")
	lines = append(lines, "added 487 packages, and audited 488 packages in 12s")
	lines = append(lines, "")
	lines = append(lines, "45 packages are looking for funding")
	lines = append(lines, "  run `npm fund` for details")
	lines = append(lines, "")
	lines = append(lines, "3 moderate severity vulnerabilities")
	lines = append(lines, "")
	lines = append(lines, "To address all issues, run:")
	lines = append(lines, "  npm audit fix")

	return strings.Join(lines, "\n") + "\n"
}

// generateCargoBuildFixture generates 40 crates compiling with 2 warnings.
func generateCargoBuildFixture() string {
	var lines []string

	crates := []string{
		"libc", "cfg-if", "unicode-ident", "proc-macro2", "quote", "syn",
		"serde_derive", "serde", "itoa", "ryu", "serde_json", "memchr",
		"aho-corasick", "regex-syntax", "regex", "log", "env_logger",
		"humantime", "termcolor", "atty", "clap_lex", "clap_derive",
		"clap_builder", "clap", "tokio-macros", "pin-project-lite",
		"tokio", "bytes", "mio", "socket2", "http", "httparse",
		"h2", "hyper", "tower-service", "tower-layer", "tower",
		"axum-core", "axum", "myproject",
	}

	// Download phase
	for i := 0; i < 15; i++ {
		lines = append(lines, fmt.Sprintf("  Downloading %s v0.%d.%d", crates[i], i%5+1, i%10))
	}
	for i := 0; i < 15; i++ {
		lines = append(lines, fmt.Sprintf("   Downloaded %s v0.%d.%d", crates[i], i%5+1, i%10))
	}

	// Compile phase
	for i, crate := range crates {
		ver := fmt.Sprintf("v%d.%d.%d", i/10+1, i%10, i%5)
		lines = append(lines, fmt.Sprintf("   Compiling %s %s", crate, ver))
	}

	// Two warnings
	lines = append(lines,
		"warning: unused variable: `x`",
		"  --> src/main.rs:42:9",
		"   |",
		"42 |     let x = compute_value();",
		"   |         ^ help: if this is intentional, prefix it with an underscore: `_x`",
		"   |",
		"   = note: `#[warn(unused_variables)]` on by default",
		"",
		"warning: function `old_handler` is never used",
		"  --> src/handlers.rs:15:4",
		"   |",
		"15 | fn old_handler() {",
		"   |    ^^^^^^^^^^^",
		"   |",
		"   = note: `#[warn(dead_code)]` on by default",
		"",
		"warning: `myproject` (bin \"myproject\") generated 2 warnings",
		"    Finished dev [unoptimized + debuginfo] target(s) in 34.21s",
	)
	return strings.Join(lines, "\n") + "\n"
}

// generateMakeBuildFixture generates gcc compilation of 30 files with entering/leaving directory.
func generateMakeBuildFixture() string {
	var lines []string

	sourceFiles := []string{
		"main.c", "server.c", "handler.c", "router.c", "config.c",
		"db.c", "pool.c", "cache.c", "logger.c", "utils.c",
		"auth.c", "session.c", "crypto.c", "hash.c", "base64.c",
		"json.c", "xml.c", "csv.c", "http.c", "websocket.c",
		"thread.c", "mutex.c", "signal.c", "timer.c", "queue.c",
		"buffer.c", "string.c", "memory.c", "file.c", "net.c",
	}

	dirs := []string{"/home/user/project/src", "/home/user/project/lib", "/home/user/project/modules"}

	for i, src := range sourceFiles {
		dir := dirs[i%len(dirs)]
		obj := strings.TrimSuffix(src, ".c") + ".o"

		lines = append(lines,
			fmt.Sprintf("make[1]: Entering directory '%s'", dir),
			fmt.Sprintf("gcc -Wall -Wextra -O2 -g -I/usr/local/include -I../include -DVERSION=\"1.0.%d\" -DDEBUG=%d -std=c11 -fPIC -pthread -c %s/%s -o build/%s",
				i, i%2, dir, src, obj),
			fmt.Sprintf("make[1]: Leaving directory '%s'", dir),
		)
	}

	// Linking step
	lines = append(lines,
		"make[1]: Entering directory '/home/user/project'",
		"gcc -Wall -Wextra -O2 -g -pthread -o build/server build/*.o -lssl -lcrypto -lpthread -lz",
		"make[1]: Leaving directory '/home/user/project'",
		"make[1]: Nothing to be done for 'clean'.",
	)

	return strings.Join(lines, "\n") + "\n"
}

// generateStackTraceFixture generates a Go panic with 5 identical goroutines.
func generateStackTraceFixture() string {
	var lines []string

	lines = append(lines,
		"panic: runtime error: invalid memory address or nil pointer dereference",
		"[signal SIGSEGV: segmentation violation code=0x1 addr=0x0 pc=0x4a2b3c]",
		"",
	)

	for i := 0; i < 5; i++ {
		lines = append(lines,
			fmt.Sprintf("goroutine %d [running]:", i+1),
			"main.(*Server).handleRequest(0x0, 0xc000123456, 0xc000789abc)",
			"\t/home/user/project/src/server/handler.go:142 +0x3c",
			"main.(*Server).serve(0xc0001a2b3c, 0xc000456def)",
			"\t/home/user/project/src/server/server.go:89 +0x1a5",
			"net/http.(*conn).serve(0xc0001b2c3d, {0x7f1234, 0xc0001c3d4e})",
			"\t/usr/local/go/src/net/http/server.go:1995 +0x612",
			"runtime.goexit()",
			"\t/usr/local/go/src/runtime/asm_amd64.s:1598 +0x1",
			"",
		)
	}

	lines = append(lines,
		"exit status 2",
	)

	return strings.Join(lines, "\n") + "\n"
}

// generateNoisyLogFixture generates 500 lines of timestamped log with ANSI colors and duplicates.
func generateNoisyLogFixture() string {
	var lines []string

	levels := []string{
		"\033[32mINFO\033[0m",
		"\033[33mWARN\033[0m",
		"\033[31mERROR\033[0m",
		"\033[36mDEBUG\033[0m",
	}
	messages := []string{
		"Processing request from 10.0.0.%d",
		"Database query completed in %dms",
		"Cache hit ratio: %d%%",
		"Connection pool size: %d",
		"Request latency p99: %dms",
		"Garbage collection pause: %dus",
		"Memory usage: %dMB",
		"Active goroutines: %d",
	}

	for i := 0; i < 500; i++ {
		level := levels[i%len(levels)]
		msg := messages[i%len(messages)]
		ts := fmt.Sprintf("2025-01-15T10:%02d:%02d.%03dZ", i/60%60, i%60, i%1000)

		// Create duplicate runs every 20 lines
		if i%20 < 5 && i > 0 {
			// Duplicate the previous line
			if len(lines) > 0 {
				lines = append(lines, lines[len(lines)-1])
				continue
			}
		}

		formatted := fmt.Sprintf("%s [%s] "+msg, ts, level, i%256)
		lines = append(lines, formatted)
	}
	return strings.Join(lines, "\n") + "\n"
}

// generateMixedErrorsFixture generates output with errors, warnings, and secrets to test redaction.
func generateMixedErrorsFixture() string {
	var lines []string

	// Normal output
	lines = append(lines,
		"Starting application server...",
		"Loading configuration from /etc/app/config.yaml",
		"",
	)

	// Config with secrets
	lines = append(lines,
		"Environment variables:",
		"  DATABASE_URL=postgres://admin:faketestpassword@db.example.com:5432/mydb",
		"  API_KEY=sk-fake00test00000000000000000000000000",
		"  GITHUB_TOKEN=ghp_FAKE00TEST00VALUE00NOT00REAL00TOKEN00xxx",
		"  AWS_ACCESS_KEY_ID=AKIAFAKETEST000000000",
		"  SECRET_TOKEN=eyJGQUtFIjoiVEVTVCJ9.eyJzdWIiOiJ0ZXN0In0.FAKE_TEST_SIG",
		"",
	)

	// Errors and warnings interleaved
	for i := 0; i < 30; i++ {
		switch i % 5 {
		case 0:
			lines = append(lines, fmt.Sprintf("ERROR: failed to connect to service-%d: connection refused", i))
		case 1:
			lines = append(lines, fmt.Sprintf("WARNING: deprecated API call in handler%d.go:42", i))
		case 2:
			lines = append(lines, fmt.Sprintf("INFO: request processed in %dms", 50+i*10))
		case 3:
			lines = append(lines, fmt.Sprintf("ERROR: timeout after 30s waiting for response from service-%d", i))
		case 4:
			lines = append(lines, fmt.Sprintf("DEBUG: cache miss for key user:%d", i*100))
		}
	}

	// Private key block
	lines = append(lines,
		"",
		"Found credential file:",
		"-----BEGIN RSA PRIVATE KEY-----",
		"MIIEpAIBAAKCAQEA2Z3qX2BTLS4e0ek55tFNaKMmMpnKGJbq",
		"xmGHkpiJOInvalidKeyDataHereForTestingPurposes1234",
		"abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUV",
		"-----END RSA PRIVATE KEY-----",
		"",
	)

	// More log output
	for i := 0; i < 20; i++ {
		lines = append(lines, fmt.Sprintf("INFO: health check passed (attempt %d/100)", i+1))
	}

	return strings.Join(lines, "\n") + "\n"
}

// generateCtxScanFixture generates realistic gotk ctx scan mode output
// with 30 files and 200+ matches across a project structure.
func generateCtxScanFixture() string {
	files := []string{
		"src/server/handler.go", "src/server/middleware.go", "src/server/routes.go",
		"src/db/postgres.go", "src/db/migrations.go", "src/db/models.go",
		"src/api/users.go", "src/api/auth.go", "src/api/products.go",
		"src/utils/helpers.go", "src/utils/validator.go", "src/utils/logger.go",
		"pkg/config/config.go", "pkg/config/defaults.go", "pkg/config/loader.go",
		"internal/cache/redis.go", "internal/cache/memory.go", "internal/cache/lru.go",
		"cmd/server/main.go", "cmd/worker/main.go", "cmd/cli/main.go",
		"test/integration/api_test.go", "test/integration/db_test.go",
		"test/unit/handler_test.go", "test/unit/auth_test.go",
		"pkg/errors/errors.go", "pkg/errors/wrap.go",
		"internal/metrics/prometheus.go", "internal/metrics/collector.go",
		"docs/api.go",
	}
	matchLines := []string{
		"func HandleRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) error {",
		"type Config struct {",
		"func NewServer(cfg *Config) *Server {",
		"if err != nil { return fmt.Errorf(\"failed: %w\", err) }",
		"func (s *Server) Start() error {",
		"type Handler interface {",
		"func Validate(input string) error {",
		"const MaxRetries = 3",
		"func init() { registerMetrics() }",
		"var ErrNotFound = errors.New(\"not found\")",
	}

	var b strings.Builder
	for i, file := range files {
		matchCount := 3 + i%8
		fmt.Fprintf(&b, "%dx %s\n", matchCount, file)
		for j := 0; j < matchCount; j++ {
			lineNum := 10 + j*15 + i*3
			line := matchLines[(i+j)%len(matchLines)]
			// Truncate long lines like scan mode does
			display := fmt.Sprintf("  %d: %s", lineNum, line)
			if len(display) > 120 {
				display = display[:117] + "..."
			}
			b.WriteString(display)
			b.WriteByte('\n')
		}
		b.WriteByte('\n')
	}
	return b.String()
}

// generateCtxDetailFixture generates realistic gotk ctx detail mode output
// with context windows around matches.
func generateCtxDetailFixture() string {
	files := []string{
		"src/server/handler.go", "src/db/postgres.go", "src/api/auth.go",
		"pkg/config/config.go", "internal/cache/redis.go",
		"cmd/server/main.go", "test/integration/api_test.go",
		"pkg/errors/errors.go", "internal/metrics/prometheus.go",
		"src/utils/validator.go",
	}
	contextLines := []string{
		"package server",
		"import (",
		"\t\"context\"",
		"\t\"fmt\"",
		"\t\"net/http\"",
		")",
		"",
		"// HandleRequest processes incoming HTTP requests.",
		"func HandleRequest(ctx context.Context, w http.ResponseWriter, r *http.Request) error {",
		"\tif r.Method != http.MethodPost {",
		"\t\treturn fmt.Errorf(\"unsupported method: %s\", r.Method)",
		"\t}",
		"\tresult, err := processBody(r.Body)",
		"\tif err != nil {",
		"\t\treturn fmt.Errorf(\"process error: %w\", err)",
		"\t}",
		"\tw.WriteHeader(http.StatusOK)",
		"\treturn json.NewEncoder(w).Encode(result)",
		"}",
		"",
	}

	var b strings.Builder
	for _, file := range files {
		for window := 0; window < 2+len(file)%3; window++ {
			startLine := 10 + window*25
			fmt.Fprintf(&b, "--- %s:%d ---\n", file, startLine)
			for j := 0; j < 7; j++ {
				lineNum := startLine + j
				content := contextLines[(window*3+j)%len(contextLines)]
				prefix := "  "
				if j == 3 { // match line
					prefix = "> "
				}
				fmt.Fprintf(&b, "%s%d: %s\n", prefix, lineNum, content)
			}
			b.WriteString("  ...\n")
		}
		b.WriteByte('\n')
	}
	return b.String()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
