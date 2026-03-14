package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// FilterMode controls how aggressively GoTK filters output.
type FilterMode string

const (
	ModeConservative FilterMode = "conservative" // minimal reduction, zero info loss
	ModeBalanced     FilterMode = "balanced"     // default — good reduction, preserves important lines
	ModeAggressive   FilterMode = "aggressive"   // maximum reduction, acceptable info loss
)

// Config holds all gotk configuration options.
type Config struct {
	General    GeneralConfig
	Filters    FiltersConfig
	Security   SecurityConfig
	Commands   map[string]string // custom command-type mappings
	Rules      RulesConfig       // whitelist/blacklist patterns
	Truncation map[string]int    // per-command max_lines overrides
}

// SecurityConfig controls security-related settings.
type SecurityConfig struct {
	CommandTimeout int  // seconds, 0 means no timeout
	MaxOutputBytes int  // max bytes captured per stream (stdout/stderr)
	RedactSecrets  bool // whether to redact secrets from output
	RateLimit      int  // max requests per minute for MCP tools/call, 0 = disabled
	RateBurst      int  // max burst size for rate limiter
}

// GeneralConfig holds general settings.
type GeneralConfig struct {
	MaxLines int
	Stats    bool
	ShellMode bool
	Mode     FilterMode
}

// RulesConfig holds whitelist/blacklist regex patterns.
// Whitelist patterns force lines to be kept even if a filter would remove them.
// Blacklist patterns force lines to be removed regardless of other filters.
// Blacklist is applied after whitelist — if a line matches both, it is removed.
type RulesConfig struct {
	AlwaysKeep   []string // regex patterns: matching lines are never removed
	AlwaysRemove []string // regex patterns: matching lines are always removed
}

// FiltersConfig controls which filters are enabled.
type FiltersConfig struct {
	StripANSI           bool
	NormalizeWhitespace bool
	Dedup               bool
	CompressPaths       bool
	TrimDecorative      bool
	Truncate            bool
}

// Default returns a Config with all default values.
func Default() *Config {
	return &Config{
		General: GeneralConfig{
			MaxLines:  50,
			Stats:     false,
			ShellMode: false,
		},
		Filters: FiltersConfig{
			StripANSI:           true,
			NormalizeWhitespace: true,
			Dedup:               true,
			CompressPaths:       true,
			TrimDecorative:      true,
			Truncate:            true,
		},
		Security: SecurityConfig{
			CommandTimeout: 30,
			MaxOutputBytes: 10 * 1024 * 1024, // 10MB
			RedactSecrets:  true,
			RateLimit:      0, // disabled by default
			RateBurst:      10,
		},
		Commands:   map[string]string{},
		Rules:      RulesConfig{},
		Truncation: map[string]int{},
	}
}

// Load reads config from disk with three levels of precedence:
//  1. Global: ~/.config/gotk/config.toml
//  2. Project: .gotk.toml found by traversing parent directories toward root
//  3. Local: ./gotk.toml in the current working directory
//
// Missing files are not an error; defaults are returned in that case.
// After loading, ApplyMode() is called to apply any mode-based overrides.
func Load() *Config {
	cfg := Default()

	// 1. Global config
	if home, err := os.UserHomeDir(); err == nil {
		globalPath := filepath.Join(home, ".config", "gotk", "config.toml")
		if data, err := os.ReadFile(globalPath); err == nil {
			applyTOML(cfg, string(data))
		}
	}

	// 2. Project config — walk up from cwd to find .gotk.toml
	if projectPath := findProjectConfig(); projectPath != "" {
		if data, err := os.ReadFile(projectPath); err == nil {
			applyTOML(cfg, string(data))
		}
	}

	// 3. Local config overrides everything
	if data, err := os.ReadFile("gotk.toml"); err == nil {
		applyTOML(cfg, string(data))
	}

	return cfg
}

// findProjectConfig walks up from the current directory looking for .gotk.toml.
// Stops at the filesystem root or after 50 levels to prevent infinite loops.
func findProjectConfig() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for i := 0; i < 50; i++ {
		candidate := filepath.Join(dir, ".gotk.toml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break // reached root
		}
		dir = parent
	}

	return ""
}

// applyTOML parses a basic TOML file (key=value with [sections]) and applies
// values to the config. This is intentionally minimal — no arrays, no inline
// tables, no multi-line strings. Just what we need.
func applyTOML(cfg *Config, data string) {
	section := ""

	for _, rawLine := range strings.Split(data, "\n") {
		line := strings.TrimSpace(rawLine)

		// Skip empty lines and comments
		if line == "" || line[0] == '#' {
			continue
		}

		// Section header
		if line[0] == '[' {
			end := strings.IndexByte(line, ']')
			if end > 0 {
				section = strings.TrimSpace(line[1:end])
			}
			continue
		}

		// Key = value
		eqIdx := strings.IndexByte(line, '=')
		if eqIdx <= 0 {
			continue
		}

		key := strings.TrimSpace(line[:eqIdx])
		val := strings.TrimSpace(line[eqIdx+1:])

		// Strip inline comments (not inside quotes)
		if !strings.HasPrefix(val, "\"") {
			if ci := strings.IndexByte(val, '#'); ci > 0 {
				val = strings.TrimSpace(val[:ci])
			}
		}

		// Strip surrounding quotes
		val = strings.Trim(val, "\"")

		switch section {
		case "general":
			switch key {
			case "max_lines":
				if n, err := strconv.Atoi(val); err == nil {
					cfg.General.MaxLines = n
				}
			case "stats":
				cfg.General.Stats = parseBool(val)
			case "shell_mode":
				cfg.General.ShellMode = parseBool(val)
			case "mode":
				cfg.General.Mode = ParseMode(val)
			}
		case "filters":
			b := parseBool(val)
			switch key {
			case "strip_ansi":
				cfg.Filters.StripANSI = b
			case "normalize_whitespace":
				cfg.Filters.NormalizeWhitespace = b
			case "dedup":
				cfg.Filters.Dedup = b
			case "compress_paths":
				cfg.Filters.CompressPaths = b
			case "trim_decorative":
				cfg.Filters.TrimDecorative = b
			case "truncate":
				cfg.Filters.Truncate = b
			}
		case "security":
			switch key {
			case "command_timeout":
				if n, err := strconv.Atoi(val); err == nil {
					cfg.Security.CommandTimeout = n
				}
			case "max_output_bytes":
				if n, err := strconv.Atoi(val); err == nil {
					cfg.Security.MaxOutputBytes = n
				}
			case "redact_secrets":
				cfg.Security.RedactSecrets = parseBool(val)
			case "rate_limit":
				if n, err := strconv.Atoi(val); err == nil {
					cfg.Security.RateLimit = n
				}
			case "rate_burst":
				if n, err := strconv.Atoi(val); err == nil {
					cfg.Security.RateBurst = n
				}
			}
		case "commands":
			cfg.Commands[key] = val
		case "rules":
			switch key {
			case "always_keep":
				cfg.Rules.AlwaysKeep = parseTOMLArray(val)
			case "always_remove":
				cfg.Rules.AlwaysRemove = parseTOMLArray(val)
			}
		case "truncation":
			if n, err := strconv.Atoi(val); err == nil {
				cfg.Truncation[key] = n
			}
		}
	}
}

// parseTOMLArray parses a simple TOML array of strings: ["a", "b", "c"].
func parseTOMLArray(s string) []string {
	s = strings.TrimSpace(s)
	if !strings.HasPrefix(s, "[") || !strings.HasSuffix(s, "]") {
		return nil
	}
	s = s[1 : len(s)-1]
	if strings.TrimSpace(s) == "" {
		return nil
	}
	var result []string
	for _, item := range strings.Split(s, ",") {
		item = strings.TrimSpace(item)
		item = strings.Trim(item, "\"")
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func parseBool(s string) bool {
	return s == "true" || s == "1" || s == "yes"
}

// ApplyMode adjusts filter and general settings based on the configured mode.
// Call after loading config and parsing CLI flags to apply mode overrides.
func (c *Config) ApplyMode() {
	switch c.General.Mode {
	case ModeConservative:
		c.General.MaxLines = 200
		c.Filters.Truncate = false
		c.Filters.TrimDecorative = false
	case ModeAggressive:
		c.General.MaxLines = 30
		// All filters stay enabled — they already compress aggressively
	case ModeBalanced:
		// Default values — no changes needed
	}
}

// MaxLinesForCommand returns the max_lines value for a specific command type.
// Falls back to General.MaxLines if no per-command override exists.
func (c *Config) MaxLinesForCommand(cmdName string) int {
	if n, ok := c.Truncation[cmdName]; ok {
		return n
	}
	return c.General.MaxLines
}

// ParseMode converts a string to a FilterMode, returning ModeBalanced for unknown values.
func ParseMode(s string) FilterMode {
	switch strings.ToLower(s) {
	case "conservative":
		return ModeConservative
	case "aggressive":
		return ModeAggressive
	case "balanced":
		return ModeBalanced
	default:
		return ModeBalanced
	}
}
