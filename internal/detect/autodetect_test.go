package detect

import (
	"strings"
	"testing"
)

func TestAutoDetect(t *testing.T) {
	tests := []struct {
		name   string
		output string
		want   CmdType
	}{
		{
			name: "grep format output",
			output: strings.Join([]string{
				"main.go:10:func main() {",
				"main.go:15:    fmt.Println()",
				"util.go:3:package util",
				"util.go:20:func helper() {",
				"util.go:25:    return nil",
			}, "\n"),
			want: CmdGrep,
		},
		{
			name: "find format output",
			output: strings.Join([]string{
				"./src/main.go",
				"./src/util.go",
				"./src/handler.go",
				"./pkg/config.go",
				"./internal/filter.go",
			}, "\n"),
			want: CmdFind,
		},
		{
			name: "git log format",
			output: strings.Join([]string{
				"commit abc1234567890abcdef1234567890abcdef123456",
				"    Initial commit",
				"commit def4567890abcdef1234567890abcdef12345678",
				"    Add feature",
				"commit 1234567890abcdef1234567890abcdef12345678",
				"    Fix bug",
			}, "\n"),
			want: CmdGit,
		},
		{
			name: "git diff format",
			output: strings.Join([]string{
				"diff --git a/main.go b/main.go",
				"--- a/main.go",
				"+++ b/main.go",
				"@@ -1,3 +1,4 @@",
				" package main",
				"+import \"fmt\"",
				"diff --git a/util.go b/util.go",
				"--- a/util.go",
				"+++ b/util.go",
			}, "\n"),
			want: CmdGit,
		},
		{
			name: "ls -l format",
			output: strings.Join([]string{
				"drwxr-xr-x  5 user group 160 Jan  1 12:00 dir1",
				"-rw-r--r--  1 user group  42 Jan  1 12:00 file1.txt",
				"-rw-r--r--  1 user group 100 Jan  1 12:00 file2.txt",
				"drwxr-xr-x  3 user group  96 Jan  1 12:00 dir2",
				"-rwxr-xr-x  1 user group 200 Jan  1 12:00 script.sh",
			}, "\n"),
			want: CmdLs,
		},
		{
			name:   "unknown format returns generic",
			output: "some random text\nthat does not match\nany known pattern",
			want:   CmdGeneric,
		},
		{
			name:   "empty output returns generic",
			output: "",
			want:   CmdGeneric,
		},
		{
			name:   "only whitespace returns generic",
			output: "   \n  \n\t",
			want:   CmdGeneric,
		},
		{
			name: "mixed output below threshold returns generic",
			output: strings.Join([]string{
				"main.go:10:func main() {",
				"some random text",
				"another random line",
				"more text",
				"yet more text",
			}, "\n"),
			want: CmdGeneric,
		},
		{
			name: "go test format",
			output: strings.Join([]string{
				"ok  \tgithub.com/example/pkg1\t0.5s",
				"ok  \tgithub.com/example/pkg2\t1.2s",
				"FAIL\tgithub.com/example/pkg3\t0.8s",
				"ok  \tgithub.com/example/pkg4\t0.3s",
				"ok  \tgithub.com/example/pkg5\t0.1s",
			}, "\n"),
			want: CmdGoTool,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AutoDetect(tt.output)
			if got != tt.want {
				t.Errorf("AutoDetect() = %d, want %d", got, tt.want)
			}
		})
	}
}
