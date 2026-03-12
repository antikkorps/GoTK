package filter

import (
	"os"
	"strings"
)

// CompressPaths shortens absolute paths by replacing the working directory
// prefix with "./" to reduce token usage.
func CompressPaths(input string) string {
	cwd, err := os.Getwd()
	if err != nil {
		return input
	}

	// Replace cwd prefix with "./"
	result := strings.ReplaceAll(input, cwd+"/", "./")
	result = strings.ReplaceAll(result, cwd, ".")

	// Also compress home directory
	home, err := os.UserHomeDir()
	if err == nil && home != "" {
		result = strings.ReplaceAll(result, home+"/", "~/")
		result = strings.ReplaceAll(result, home, "~")
	}

	return result
}
