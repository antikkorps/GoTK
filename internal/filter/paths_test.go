package filter

import (
	"os"
	"testing"
)

func TestCompressPaths(t *testing.T) {
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("could not get working directory: %v", err)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("could not get home directory: %v", err)
	}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "cwd file replaced with dot-slash",
			input: cwd + "/main.go",
			want:  "./main.go",
		},
		{
			name:  "cwd alone replaced with dot",
			input: cwd,
			want:  ".",
		},
		{
			name:  "home dir replaced with tilde-slash",
			input: home + "/Documents/file.txt",
			want:  "~/Documents/file.txt",
		},
		{
			name:  "home dir alone replaced with tilde",
			input: home,
			want:  "~",
		},
		{
			name:  "mixed cwd and home paths",
			input: cwd + "/foo.go\n" + home + "/bar.txt",
			want:  "./foo.go\n~/bar.txt",
		},
		{
			name:  "no matching paths unchanged",
			input: "/usr/local/bin/go",
			want:  "/usr/local/bin/go",
		},
		{
			name:  "empty string",
			input: "",
			want:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompressPaths(tt.input)
			if got != tt.want {
				t.Errorf("CompressPaths(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
