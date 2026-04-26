package paths

import (
	"runtime"
	"strings"
	"testing"
)

func TestConfigDirNotEmpty(t *testing.T) {
	dir, ok := ConfigDir()
	if !ok {
		t.Fatal("ConfigDir() reported failure")
	}
	if dir == "" {
		t.Fatal("ConfigDir() returned empty path")
	}
	if !strings.HasSuffix(dir, "gotk") {
		t.Errorf("ConfigDir() = %q, expected suffix 'gotk'", dir)
	}
}

func TestDataDirCollapsesOnWindows(t *testing.T) {
	cfg, _ := ConfigDir()
	data, _ := DataDir()
	if runtime.GOOS == "windows" {
		if cfg != data {
			t.Errorf("on Windows, ConfigDir (%q) and DataDir (%q) should match", cfg, data)
		}
	} else {
		if cfg == data {
			t.Errorf("on %s, ConfigDir and DataDir should differ; both = %q", runtime.GOOS, cfg)
		}
	}
}

func TestConfigFile(t *testing.T) {
	path, ok := ConfigFile()
	if !ok {
		t.Fatal("ConfigFile() reported failure")
	}
	if !strings.HasSuffix(path, "config.toml") {
		t.Errorf("ConfigFile() = %q, expected suffix 'config.toml'", path)
	}
}
