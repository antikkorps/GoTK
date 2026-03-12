package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// Config holds all gotk configuration options.
type Config struct {
	General  GeneralConfig
	Filters  FiltersConfig
	Commands map[string]string // custom command-type mappings
}

// GeneralConfig holds general settings.
type GeneralConfig struct {
	MaxLines  int
	Stats     bool
	ShellMode bool
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
		Commands: map[string]string{},
	}
}

// Load reads config from disk. Local ./gotk.toml takes precedence over
// ~/.config/gotk/config.toml. Missing files are not an error; defaults
// are returned in that case.
func Load() *Config {
	cfg := Default()

	// Try global config first
	if home, err := os.UserHomeDir(); err == nil {
		globalPath := filepath.Join(home, ".config", "gotk", "config.toml")
		if data, err := os.ReadFile(globalPath); err == nil {
			applyTOML(cfg, string(data))
		}
	}

	// Local config overrides global
	if data, err := os.ReadFile("gotk.toml"); err == nil {
		applyTOML(cfg, string(data))
	}

	return cfg
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
		case "commands":
			cfg.Commands[key] = val
		}
	}
}

func parseBool(s string) bool {
	return s == "true" || s == "1" || s == "yes"
}
