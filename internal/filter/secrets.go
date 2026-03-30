package filter

import (
	"regexp"
	"strings"
)

// secretPatterns defines patterns for known secret/token formats.
var secretPatterns = []*regexp.Regexp{
	// API keys and tokens with known prefixes
	regexp.MustCompile(`\bsk-[A-Za-z0-9]{20,}\b`),      // OpenAI / Stripe secret keys
	regexp.MustCompile(`\bghp_[A-Za-z0-9]{36,}\b`),     // GitHub personal access tokens
	regexp.MustCompile(`\bghu_[A-Za-z0-9]{36,}\b`),     // GitHub user-to-server tokens
	regexp.MustCompile(`\bghs_[A-Za-z0-9]{36,}\b`),     // GitHub server-to-server tokens
	regexp.MustCompile(`\bglpat-[A-Za-z0-9\-]{20,}\b`), // GitLab personal access tokens
	regexp.MustCompile(`\bxoxb-[A-Za-z0-9\-]{10,}\b`),  // Slack bot tokens
	regexp.MustCompile(`\bxoxp-[A-Za-z0-9\-]{10,}\b`),  // Slack user tokens

	// AWS access key IDs (start with AKIA, 20 uppercase alphanumeric chars)
	regexp.MustCompile(`\bAKIA[A-Z0-9]{16}\b`),

	// Bearer tokens in HTTP headers (Authorization: Bearer <token>)
	regexp.MustCompile(`(?i)(Bearer\s+)[A-Za-z0-9_\-\.]{10,}`),
}

// jwtPattern matches JWT tokens (three base64url segments separated by dots).
// Covers both long and short JWTs (minimum 5 chars per segment).
var jwtPattern = regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}\.[A-Za-z0-9_-]{5,}\b`)

// privateKeyPattern matches PEM private key blocks (RSA, EC, DSA, ED25519, OPENSSH, etc.).
var privateKeyPattern = regexp.MustCompile(`(?s)-----BEGIN [A-Z0-9 ]{0,20}PRIVATE KEY-----.*?-----END [A-Z0-9 ]{0,20}PRIVATE KEY-----`)

// connectionStringPattern matches passwords in connection strings like
// scheme://user:password@host
var connectionStringPattern = regexp.MustCompile(`(://[^:/@\s]+:)([^@\s]+)(@)`)

// envSecretPattern matches lines with KEY=VALUE where the key name suggests a secret.
// It captures the key name (group 1), the separator (group 2), and the value (group 3).
var envSecretPattern = regexp.MustCompile(`(?i)((?:^|[\s"']|export\s+)(?:[A-Z_]*(?:KEY|SECRET|TOKEN|PASSWORD|PASSWD|APIKEY|API_KEY)[A-Z_]*)\s*=\s*)(.+)`)

const redacted = "[REDACTED]"

// RedactSecrets replaces potential secrets with [REDACTED] markers.
// It preserves key names and only redacts the secret values.
func RedactSecrets(input string) string {
	result := input

	// Redact PEM private key blocks (must be done before line-based processing)
	result = privateKeyPattern.ReplaceAllString(result, "-----BEGIN PRIVATE KEY-----\n"+redacted+"\n-----END PRIVATE KEY-----")

	// Redact JWT tokens
	result = jwtPattern.ReplaceAllString(result, redacted)

	// Redact known secret prefixes
	for _, p := range secretPatterns {
		result = p.ReplaceAllString(result, redacted)
	}

	// Redact connection string passwords
	result = connectionStringPattern.ReplaceAllString(result, "${1}"+redacted+"${3}")

	// Redact environment variable values that look like secrets
	// Process line by line to handle the KEY=VALUE pattern correctly
	lines := strings.Split(result, "\n")
	for i, line := range lines {
		lines[i] = envSecretPattern.ReplaceAllString(line, "${1}"+redacted)
	}
	result = strings.Join(lines, "\n")

	return result
}
