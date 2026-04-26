//go:build !windows

package shell

import (
	"os"
	"strings"
)

// Default returns a shell path and the flag that asks it to execute a
// single command string. Honours GOTK_SHELL, then SHELL (unless it points
// at gotk itself, which would recurse), then falls back to /bin/bash and
// /bin/sh.
func Default() (path, flag string) {
	if s := os.Getenv("GOTK_SHELL"); s != "" {
		return s, "-c"
	}
	if s := os.Getenv("SHELL"); s != "" {
		base := s
		if idx := strings.LastIndexByte(s, '/'); idx >= 0 {
			base = s[idx+1:]
		}
		if base != "gotk" {
			return s, "-c"
		}
	}
	for _, sh := range []string{"/bin/bash", "/bin/sh"} {
		if _, err := os.Stat(sh); err == nil {
			return sh, "-c"
		}
	}
	return "sh", "-c"
}
