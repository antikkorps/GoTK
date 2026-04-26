package shell

import (
	"os"
	"runtime"
	"testing"
)

func TestDefaultRespectsGotkShell(t *testing.T) {
	t.Setenv("GOTK_SHELL", "/custom/bin/myshell")
	t.Setenv("SHELL", "/bin/zsh")
	path, flag := Default()
	if path != "/custom/bin/myshell" {
		t.Errorf("path = %q, want /custom/bin/myshell", path)
	}
	if flag == "" {
		t.Errorf("flag is empty")
	}
}

func TestDefaultIgnoresGotkSelfRef(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("SHELL env semantics are POSIX-only")
	}
	t.Setenv("GOTK_SHELL", "")
	t.Setenv("SHELL", "/usr/local/bin/gotk")
	path, _ := Default()
	if path == "/usr/local/bin/gotk" {
		t.Errorf("Default() returned gotk itself, would recurse")
	}
}

func TestDefaultProvidesUsableFlag(t *testing.T) {
	t.Setenv("GOTK_SHELL", "")
	_, flag := Default()
	switch flag {
	case "-c", "/c", "/C":
	default:
		if runtime.GOOS == "windows" {
			t.Errorf("Windows flag = %q, want /c or /C", flag)
		} else {
			t.Errorf("Unix flag = %q, want -c", flag)
		}
	}
}

func TestDefaultFallback(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix fallback path")
	}
	os.Unsetenv("GOTK_SHELL")
	os.Unsetenv("SHELL")
	path, _ := Default()
	if path == "" {
		t.Errorf("path is empty")
	}
}
