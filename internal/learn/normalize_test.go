package learn

import (
	"regexp"
	"testing"
)

func TestNormalize(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "empty line",
			input: "",
			want:  "",
		},
		{
			name:  "plain text",
			input: "   Compiling serde",
			want:  "Compiling serde",
		},
		{
			name:  "version number",
			input: "   Compiling serde v1.0.152",
			want:  "Compiling serde <VERSION>",
		},
		{
			name:  "git hash",
			input: "commit abc1234def5678 (HEAD -> main)",
			want:  "commit <HASH> (HEAD -> main)",
		},
		{
			name:  "UUID",
			input: "request-id: 550e8400-e29b-41d4-a716-446655440000",
			want:  "request-id: <UUID>",
		},
		{
			name:  "ISO timestamp",
			input: "2024-01-15T10:30:00Z INFO starting",
			want:  "<TIMESTAMP> INFO starting",
		},
		{
			name:  "numbers",
			input: "Downloaded 42 files (1024 bytes)",
			want:  "Downloaded <N> files (<SIZE>)",
		},
		{
			name:  "file size",
			input: "Total: 1.5MB compressed",
			want:  "Total: <SIZE> compressed",
		},
		{
			name:  "duration",
			input: "Build took 3.2s",
			want:  "Build took <DURATION>",
		},
		{
			name:  "IP address",
			input: "Connected to 192.168.1.100:8080",
			want:  "Connected to <IP>",
		},
		{
			name:  "multiple replacements",
			input: "   Compiling tokio v1.28.0 (3.2s, 42 files)",
			want:  "Compiling tokio <VERSION> (<DURATION>, <N> files)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := Normalize(tt.input)
			if got != tt.want {
				t.Errorf("Normalize(%q)\n  got  = %q\n  want = %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToRegex(t *testing.T) {
	tests := []struct {
		name       string
		normalized string
		shouldMatch []string
		shouldNot   []string
	}{
		{
			name:       "version pattern",
			normalized: "Compiling serde <VERSION>",
			shouldMatch: []string{
				"   Compiling serde v1.0.152",
				"  Compiling serde v2.0.0-beta",
			},
			shouldNot: []string{
				"Error: compilation failed",
			},
		},
		{
			name:       "number pattern",
			normalized: "Downloaded <N> files",
			shouldMatch: []string{
				"Downloaded 42 files",
				"  Downloaded 1 files",
			},
			shouldNot: []string{
				"Upload complete",
			},
		},
		{
			name:       "timestamp pattern",
			normalized: "<TIMESTAMP> INFO starting",
			shouldMatch: []string{
				"  2024-01-15T10:30:00Z INFO starting",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pattern := ToRegex(tt.normalized)
			re, err := regexp.Compile(pattern)
			if err != nil {
				t.Fatalf("ToRegex(%q) produced invalid regex %q: %v", tt.normalized, pattern, err)
			}

			for _, s := range tt.shouldMatch {
				if !re.MatchString(s) {
					t.Errorf("pattern %q should match %q but didn't", pattern, s)
				}
			}

			for _, s := range tt.shouldNot {
				if re.MatchString(s) {
					t.Errorf("pattern %q should NOT match %q but did", pattern, s)
				}
			}
		})
	}
}

func TestRegexEscape(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"hello", "hello"},
		{"file.go", "file\\.go"},
		{"(a+b)*c", "\\(a\\+b\\)\\*c"},
		{"no[1]", "no\\[1\\]"},
	}

	for _, tt := range tests {
		got := regexEscape(tt.input)
		if got != tt.want {
			t.Errorf("regexEscape(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
