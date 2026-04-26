//go:build windows

package shell

import "os"

// Default returns cmd.exe (or COMSPEC if set) with the /c flag. GOTK_SHELL
// can override for users who prefer pwsh or bash from WSL/Git Bash; in that
// case the caller is responsible for picking a compatible flag via
// GOTK_SHELL_FLAG (defaults to /c, which is what cmd.exe expects).
func Default() (path, flag string) {
	flag = os.Getenv("GOTK_SHELL_FLAG")
	if flag == "" {
		flag = "/c"
	}
	if s := os.Getenv("GOTK_SHELL"); s != "" {
		return s, flag
	}
	if s := os.Getenv("COMSPEC"); s != "" {
		return s, flag
	}
	return "cmd.exe", flag
}
