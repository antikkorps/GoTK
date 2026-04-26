package filter

import (
	"os"
	"strings"
)

var (
	cachedCwd  string
	cachedHome string
	pathSep    = string(os.PathSeparator)
)

func init() {
	cachedCwd, _ = os.Getwd()
	cachedHome, _ = os.UserHomeDir()
}

// CompressPaths shortens absolute paths by replacing the working directory
// prefix with "./" to reduce token usage.
func CompressPaths(input string) string {
	if cachedCwd == "" {
		return input
	}

	// Replace cwd prefix with "./"
	result := strings.ReplaceAll(input, cachedCwd+pathSep, "."+pathSep)
	result = strings.ReplaceAll(result, cachedCwd, ".")

	// Also compress home directory
	if cachedHome != "" {
		result = strings.ReplaceAll(result, cachedHome+pathSep, "~"+pathSep)
		result = strings.ReplaceAll(result, cachedHome, "~")
	}

	return result
}
