package filter

import (
	"strings"
	"testing"
)

func TestRedactSecrets_APIKeys(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "OpenAI key",
			input: "key: sk-abc123def456ghi789jkl012mno345pqr678",
			want:  "key: [REDACTED]",
		},
		{
			name:  "GitHub PAT",
			input: "token=ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijkl",
			want:  "token=[REDACTED]",
		},
		{
			name:  "GitHub user token",
			input: "auth: ghu_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijkl",
			want:  "auth: [REDACTED]",
		},
		{
			name:  "GitLab PAT",
			input: "GITLAB_TOKEN=glpat-abcdefghij1234567890",
			want:  "GITLAB_TOKEN=[REDACTED]",
		},
		{
			name:  "Slack bot token",
			input: "SLACK_TOKEN=xoxb-123456789-abcdefgh",
			want:  "SLACK_TOKEN=[REDACTED]",
		},
		{
			name:  "Slack user token",
			input: "xoxp-1234567890-abcdefghij",
			want:  "[REDACTED]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactSecrets(tt.input)
			if got != tt.want {
				t.Errorf("RedactSecrets(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestRedactSecrets_AWSKeys(t *testing.T) {
	input := "aws_access_key_id = AKIAIOSFODNN7EXAMPLE"
	got := RedactSecrets(input)
	if !strings.Contains(got, "[REDACTED]") {
		t.Errorf("expected AWS key to be redacted, got: %s", got)
	}
	if strings.Contains(got, "AKIAIOSFODNN7EXAMPLE") {
		t.Errorf("AWS key was not redacted: %s", got)
	}
}

func TestRedactSecrets_JWT(t *testing.T) {
	input := "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
	got := RedactSecrets(input)
	if strings.Contains(got, "eyJ") {
		t.Errorf("JWT token was not redacted: %s", got)
	}
	if !strings.Contains(got, "Authorization: Bearer [REDACTED]") {
		t.Errorf("expected 'Authorization: Bearer [REDACTED]', got: %s", got)
	}
}

func TestRedactSecrets_PrivateKey(t *testing.T) {
	input := `some text
-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF8PbnGOY
-----END RSA PRIVATE KEY-----
more text`

	got := RedactSecrets(input)
	if strings.Contains(got, "MIIEpAIBAAKCAQEA0Z3VS5JJcds3xfn") {
		t.Errorf("private key content was not redacted: %s", got)
	}
	if !strings.Contains(got, "[REDACTED]") {
		t.Errorf("expected [REDACTED] marker, got: %s", got)
	}
	if !strings.Contains(got, "some text") || !strings.Contains(got, "more text") {
		t.Errorf("surrounding text should be preserved, got: %s", got)
	}
}

func TestRedactSecrets_EnvVars(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantKey   string // key should be preserved
		wantNoVal string // value should not appear
	}{
		{
			name:      "SECRET_KEY",
			input:     "SECRET_KEY=mysecretvalue123",
			wantKey:   "SECRET_KEY=",
			wantNoVal: "mysecretvalue123",
		},
		{
			name:      "API_TOKEN",
			input:     "API_TOKEN=tok_abc123",
			wantKey:   "API_TOKEN=",
			wantNoVal: "tok_abc123",
		},
		{
			name:      "DB_PASSWORD",
			input:     "DB_PASSWORD=hunter2",
			wantKey:   "DB_PASSWORD=",
			wantNoVal: "hunter2",
		},
		{
			name:      "MY_APIKEY",
			input:     "MY_APIKEY=key123456",
			wantKey:   "MY_APIKEY=",
			wantNoVal: "key123456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := RedactSecrets(tt.input)
			if !strings.Contains(got, tt.wantKey) {
				t.Errorf("key name should be preserved: want %q in %q", tt.wantKey, got)
			}
			if strings.Contains(got, tt.wantNoVal) {
				t.Errorf("secret value should be redacted: found %q in %q", tt.wantNoVal, got)
			}
		})
	}
}

func TestRedactSecrets_ConnectionString(t *testing.T) {
	input := "DATABASE_URL=postgres://admin:s3cret_p4ss@db.example.com:5432/mydb"
	got := RedactSecrets(input)
	if strings.Contains(got, "s3cret_p4ss") {
		t.Errorf("connection string password was not redacted: %s", got)
	}
	if !strings.Contains(got, "admin:") {
		t.Errorf("username should be preserved: %s", got)
	}
	if !strings.Contains(got, "@db.example.com") {
		t.Errorf("host should be preserved: %s", got)
	}
}

func TestRedactSecrets_NoFalsePositives(t *testing.T) {
	inputs := []string{
		"PATH=/usr/bin:/usr/local/bin",
		"GOPATH=/home/user/go",
		"normal log output here",
		"func main() {",
		"error: file not found",
	}

	for _, input := range inputs {
		got := RedactSecrets(input)
		if got != input {
			t.Errorf("false positive: input %q was modified to %q", input, got)
		}
	}
}

func TestRedactSecrets_MultipleSecretsInOneLine(t *testing.T) {
	input := "keys: sk-abcdefghijklmnopqrstuvwxyz and ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijkl"
	got := RedactSecrets(input)
	count := strings.Count(got, "[REDACTED]")
	if count < 2 {
		t.Errorf("expected at least 2 redactions, got %d in: %s", count, got)
	}
}
