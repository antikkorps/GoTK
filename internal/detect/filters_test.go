package detect

import (
	"strconv"
	"strings"
	"testing"
)

func TestCompressDockerOutput(t *testing.T) {
	input := strings.Join([]string{
		"Sending build context to Docker daemon  2.048kB",
		"Step 1/12 : FROM golang:1.21-alpine AS builder",
		" ---> abc123def456",
		"Step 2/12 : WORKDIR /app",
		" ---> Running in 1234567890ab",
		"Removing intermediate container 1234567890ab",
		" ---> 2345678901bc",
		"Step 3/12 : COPY go.mod go.sum ./",
		" ---> Running in 3456789012cd",
		"Removing intermediate container 3456789012cd",
		" ---> 4567890123de",
		"Step 4/12 : RUN go mod download",
		" ---> Running in 5678901234ef",
		"Removing intermediate container 5678901234ef",
		" ---> 6789012345fa",
		"Step 5/12 : COPY . .",
		" ---> Running in 7890123456ab",
		"Removing intermediate container 7890123456ab",
		" ---> 8901234567bc",
		"Step 6/12 : RUN go build -o /app/server .",
		" ---> Running in 9012345678cd",
		"Removing intermediate container 9012345678cd",
		" ---> 0123456789de",
		"Step 7/12 : FROM alpine:3.18",
		" ---> aabbccddee11",
		"Step 8/12 : COPY --from=builder /app/server /server",
		" ---> Running in bbccddee2233",
		"Removing intermediate container bbccddee2233",
		" ---> ccddee334455",
		"Step 9/12 : EXPOSE 8080",
		" ---> Running in ddee44556677",
		"Removing intermediate container ddee44556677",
		" ---> eeff55667788",
		"Step 10/12 : ENV APP_ENV=production",
		" ---> Running in ff0066778899",
		"Removing intermediate container ff0066778899",
		" ---> 001177889900",
		"Step 11/12 : RUN adduser -D appuser",
		" ---> Running in 112288990011",
		"Removing intermediate container 112288990011",
		" ---> 223399001122",
		"Step 12/12 : CMD [\"/server\"]",
		" ---> Running in 334400112233",
		"Removing intermediate container 334400112233",
		" ---> 445511223344",
		"Successfully built 445511223344",
		"Successfully tagged myapp:latest",
	}, "\n")

	got := compressDockerOutput(input)

	// Should not contain intermediate container lines
	if strings.Contains(got, "Running in") {
		t.Error("output should not contain 'Running in' lines")
	}
	if strings.Contains(got, "Removing intermediate") {
		t.Error("output should not contain 'Removing intermediate' lines")
	}
	// Should not contain bare hash lines
	if strings.Contains(got, " ---> abc123") {
		t.Error("output should not contain arrow-hash lines")
	}

	// Should contain the step commands with FROM stage annotation
	if !strings.Contains(got, "--- FROM golang:1.21-alpine AS builder ---") {
		t.Error("output should contain annotated FROM command")
	}
	if !strings.Contains(got, "COPY go.mod go.sum ./") {
		t.Error("output should contain COPY command")
	}
	if !strings.Contains(got, "RUN go build -o /app/server .") {
		t.Error("output should contain RUN command")
	}

	// Should keep success lines
	if !strings.Contains(got, "Successfully built") {
		t.Error("output should contain 'Successfully built'")
	}
	if !strings.Contains(got, "Successfully tagged") {
		t.Error("output should contain 'Successfully tagged'")
	}

	// Should be significantly shorter
	inputLines := len(strings.Split(input, "\n"))
	gotLines := len(strings.Split(got, "\n"))
	if gotLines >= inputLines {
		t.Errorf("expected compressed output to be shorter: input=%d lines, got=%d lines", inputLines, gotLines)
	}
}

func TestCompressDockerPull(t *testing.T) {
	input := strings.Join([]string{
		"Using default tag: latest",
		"Pulling from library/nginx",
		"abc123def456: Pulling fs layer",
		"bcd234ef5678: Pulling fs layer",
		"cde345fa6789: Pulling fs layer",
		"abc123def456: Downloading",
		"bcd234ef5678: Downloading",
		"cde345fa6789: Downloading",
		"abc123def456: Download complete",
		"bcd234ef5678: Download complete",
		"cde345fa6789: Pull complete",
		"abc123def456: Pull complete",
		"bcd234ef5678: Pull complete",
		"Digest: sha256:abcdef1234567890",
		"Status: Downloaded newer image for nginx:latest",
	}, "\n")

	got := compressDockerOutput(input)

	// Should compress layer progress lines
	if strings.Contains(got, "Pulling fs layer") {
		t.Error("should not contain individual layer progress lines")
	}

	// Should keep digest and status
	if !strings.Contains(got, "Digest:") {
		t.Error("should contain Digest line")
	}
	if !strings.Contains(got, "Status:") {
		t.Error("should contain Status line")
	}
}

func TestCompressNpmOutput(t *testing.T) {
	var lines []string
	// Add a bunch of npm warn lines
	lines = append(lines, "npm warn deprecated inflight@1.0.6: This module is not supported")
	lines = append(lines, "npm warn deprecated rimraf@3.0.2: Rimraf versions prior to v4 are no longer supported")
	lines = append(lines, "npm warn deprecated glob@7.2.0: Glob versions prior to v9 are no longer supported")
	lines = append(lines, "npm warn deprecated mkdirp@0.5.6: Legacy versions of mkdirp are no longer supported")
	lines = append(lines, "npm warn deprecated gauge@4.0.4: This package is no longer supported")
	lines = append(lines, "npm warn deprecated npmlog@6.0.2: This package is no longer supported")
	lines = append(lines, "npm warn deprecated are-we-there-yet@3.0.1: This package is no longer supported")
	for i := 0; i < 43; i++ {
		lines = append(lines, "npm warn deprecated somepackage@"+strconv.Itoa(i)+".0.0: deprecated")
	}
	lines = append(lines, "")
	lines = append(lines, "added 523 packages, and audited 524 packages in 12s")
	lines = append(lines, "")
	lines = append(lines, "67 packages are looking for funding")
	lines = append(lines, "  run `npm fund` for details")
	lines = append(lines, "")
	lines = append(lines, "found 0 vulnerabilities")

	input := strings.Join(lines, "\n")
	got := compressNpmOutput(input)

	// Should keep first warning
	if !strings.Contains(got, "inflight@1.0.6") {
		t.Error("should keep first npm warn")
	}

	// Should have summary of remaining warnings
	if !strings.Contains(got, "more npm warnings") {
		t.Error("should have warning count summary")
	}

	// Should keep the "added N packages" summary
	if !strings.Contains(got, "added 523 packages") {
		t.Error("should keep 'added packages' summary")
	}

	// Should keep vulnerability info
	if !strings.Contains(got, "found 0 vulnerabilities") {
		t.Error("should keep vulnerability summary")
	}

	// Total warnings in original: 50. Output should only have 2 warning-related lines.
	warnLines := 0
	for _, l := range strings.Split(got, "\n") {
		if strings.Contains(l, "npm warn") || strings.Contains(l, "more npm warnings") {
			warnLines++
		}
	}
	if warnLines > 2 {
		t.Errorf("expected at most 2 warning lines, got %d", warnLines)
	}
}

func TestCompressNpmErrors(t *testing.T) {
	input := strings.Join([]string{
		"npm warn deprecated inflight@1.0.6: This module is not supported",
		"npm ERR! code ERESOLVE",
		"npm ERR! ERESOLVE unable to resolve dependency tree",
		"npm ERR! Found: react@17.0.2",
		"npm ERR! Could not resolve dependency:",
	}, "\n")

	got := compressNpmOutput(input)

	// Errors must be preserved
	if !strings.Contains(got, "npm ERR! code ERESOLVE") {
		t.Error("must preserve npm error lines")
	}
	if !strings.Contains(got, "ERESOLVE unable to resolve") {
		t.Error("must preserve npm error details")
	}
}

func TestCompressCargoOutput(t *testing.T) {
	var lines []string
	lines = append(lines, "   Downloading crates ...")
	lines = append(lines, "   Downloaded serde v1.0.188")
	lines = append(lines, "   Downloaded serde_json v1.0.107")
	lines = append(lines, "   Downloaded tokio v1.32.0")
	lines = append(lines, "   Downloaded axum v0.6.20")
	lines = append(lines, "   Downloaded hyper v0.14.27")
	for i := 0; i < 15; i++ {
		lines = append(lines, "   Downloaded somecrate v0."+strconv.Itoa(i)+".0")
	}
	lines = append(lines, "   Compiling proc-macro2 v1.0.67")
	lines = append(lines, "   Compiling unicode-ident v1.0.12")
	lines = append(lines, "   Compiling quote v1.0.33")
	lines = append(lines, "   Compiling syn v2.0.37")
	lines = append(lines, "   Compiling serde v1.0.188")
	lines = append(lines, "   Compiling serde_json v1.0.107")
	lines = append(lines, "   Compiling tokio v1.32.0")
	lines = append(lines, "   Compiling axum v0.6.20")
	lines = append(lines, "   Compiling hyper v0.14.27")
	for i := 0; i < 11; i++ {
		lines = append(lines, "   Compiling crate"+strconv.Itoa(i)+" v0.1.0")
	}
	lines = append(lines, "   Compiling myproject v0.1.0 (/home/user/project)")
	lines = append(lines, "    Finished dev [unoptimized + debuginfo] target(s) in 45.23s")

	input := strings.Join(lines, "\n")
	got := compressCargoOutput(input)

	// Should have compiled crates summary (not individual lines)
	if !strings.Contains(got, "Compiled") {
		t.Error("should contain Compiled summary")
	}
	if strings.Contains(got, "Compiling proc-macro2") {
		t.Error("should not contain individual Compiling lines")
	}

	// Should have downloaded summary
	if !strings.Contains(got, "Downloaded") && !strings.Contains(got, "crates") {
		t.Error("should contain Downloaded summary")
	}
	if strings.Contains(got, "Downloaded serde v") {
		t.Error("should not contain individual Downloaded lines")
	}

	// Should keep Finished line
	if !strings.Contains(got, "Finished dev") {
		t.Error("should keep Finished summary line")
	}
}

func TestCompressCargoErrors(t *testing.T) {
	input := strings.Join([]string{
		"   Compiling myproject v0.1.0 (/home/user/project)",
		"error[E0382]: borrow of moved value: `x`",
		" --> src/main.rs:10:5",
		"  |",
		"9 |     let y = x;",
		"  |             - value moved here",
		"10|     println!(\"{}\", x);",
		"  |                      ^ value borrowed here after move",
		"",
		"warning[unused_variable]: unused variable: `z`",
		" --> src/main.rs:15:9",
		"",
		"error: aborting due to previous error",
	}, "\n")

	got := compressCargoOutput(input)

	// Must preserve all error lines
	if !strings.Contains(got, "error[E0382]") {
		t.Error("must preserve cargo error lines")
	}
	if !strings.Contains(got, "value moved here") {
		t.Error("must preserve error context")
	}
	if !strings.Contains(got, "warning[unused_variable]") {
		t.Error("must preserve warning lines")
	}
}

func TestCompressMakeOutput(t *testing.T) {
	input := strings.Join([]string{
		"make[1]: Entering directory '/home/user/project/src'",
		"gcc -Wall -Wextra -O2 -I/usr/include/libxml2 -I../include -DNDEBUG -fPIC -std=c11 -c main.c -o main.o",
		"gcc -Wall -Wextra -O2 -I/usr/include/libxml2 -I../include -DNDEBUG -fPIC -std=c11 -c util.c -o util.o",
		"gcc -Wall -Wextra -O2 -I/usr/include/libxml2 -I../include -DNDEBUG -fPIC -std=c11 -c parser.c -o parser.o",
		"gcc -Wall -Wextra -O2 -I/usr/include/libxml2 -I../include -DNDEBUG -fPIC -std=c11 -c config.c -o config.o",
		"make[1]: Leaving directory '/home/user/project/src'",
		"make[1]: Entering directory '/home/user/project/lib'",
		"g++ -std=c++17 -Wall -O2 -I../include -c library.cpp -o library.o",
		"make[1]: Nothing to be done for 'clean'",
		"make[1]: Leaving directory '/home/user/project/lib'",
		"gcc -o myapp main.o util.o parser.o config.o library.o -lxml2 -lpthread",
		"make: *** [Makefile:42: all] Error 2",
	}, "\n")

	got := compressMakeOutput(input)

	// Should remove entering/leaving directory lines
	if strings.Contains(got, "Entering directory") {
		t.Error("should remove 'Entering directory' lines")
	}
	if strings.Contains(got, "Leaving directory") {
		t.Error("should remove 'Leaving directory' lines")
	}

	// Should remove "Nothing to be done" lines
	if strings.Contains(got, "Nothing to be done") {
		t.Error("should remove 'Nothing to be done' lines")
	}

	// Should compress gcc commands to show just source files
	if strings.Contains(got, "-Wall -Wextra -O2") {
		t.Error("should compress gcc flags")
	}
	if !strings.Contains(got, "main.c") {
		t.Error("should keep source file name in compressed gcc line")
	}
	if !strings.Contains(got, "util.c") {
		t.Error("should keep source file name in compressed gcc line")
	}

	// Should keep make error lines
	if !strings.Contains(got, "make: ***") {
		t.Error("must preserve make error lines")
	}
}

func TestCompressMakePreservesErrors(t *testing.T) {
	input := strings.Join([]string{
		"gcc -Wall -c main.c -o main.o",
		"main.c:10:5: error: expected ';' before 'return'",
		"   10 |     return 0",
		"      |     ^~~~~~",
		"make: *** [Makefile:5: main.o] Error 1",
	}, "\n")

	got := compressMakeOutput(input)

	// All error lines must be preserved
	if !strings.Contains(got, "error: expected ';'") {
		t.Error("must preserve compiler error")
	}
	if !strings.Contains(got, "make: ***") {
		t.Error("must preserve make error summary")
	}
}
