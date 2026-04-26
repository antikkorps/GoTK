// Package paths centralizes platform-aware filesystem paths used by gotk
// (config files, measurement logs, update-check state). On Unix, gotk keeps
// its long-standing XDG-style layout under ~/.config and ~/.local/share so
// existing installs are not relocated. On Windows, paths resolve under
// %AppData%/gotk via os.UserConfigDir.
package paths

import (
	"os"
	"path/filepath"
	"runtime"
)

// ConfigDir returns the directory holding gotk's user config (config.toml).
// Returns ("", false) if neither the platform-specific config dir nor the
// home directory can be resolved.
func ConfigDir() (string, bool) {
	if runtime.GOOS == "windows" {
		if dir, err := os.UserConfigDir(); err == nil {
			return filepath.Join(dir, "gotk"), true
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config", "gotk"), true
	}
	return "", false
}

// DataDir returns the directory holding gotk's persistent data (measure logs,
// update-check timestamp, learned patterns). On Windows this collapses onto
// the same %AppData%/gotk directory as ConfigDir; on Unix it follows the
// XDG_DATA_HOME convention under ~/.local/share/gotk.
func DataDir() (string, bool) {
	if runtime.GOOS == "windows" {
		return ConfigDir()
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share", "gotk"), true
	}
	return "", false
}

// ConfigFile returns the absolute path of gotk's global config file.
func ConfigFile() (string, bool) {
	dir, ok := ConfigDir()
	if !ok {
		return "", false
	}
	return filepath.Join(dir, "config.toml"), true
}
